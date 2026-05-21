package task

import (
	projectv1 "building-services/gen/project/v1"
	"building-services/project-service/internal/authz"
	"building-services/project-service/internal/errs"
	"building-services/project-service/internal/util"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"building-services/project-service/internal/events"
	"building-services/project-service/internal/user"

	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Service struct {
	taskRepo     TaskRepo
	projectRepo  ProjectRepo
	userRepo     UserRepo
	activityRepo ActivityRepo
	authz        PermissionChecker
	events       events.Publisher
}

func NewService(taskRepo TaskRepo,
	projectRepo ProjectRepo,
	userRepo UserRepo,
	activityRepo ActivityRepo,
	authz PermissionChecker,
	eventPublisher events.Publisher) *Service {
	return &Service{
		taskRepo:     taskRepo,
		projectRepo:  projectRepo,
		userRepo:     userRepo,
		activityRepo: activityRepo,
		authz:        authz,
		events:      eventPublisher,
	}
}

type ActivityRepo interface {
	Exists(ctx context.Context, id string) (bool, error)
}

type ProjectRepo interface {
	FindByID(ctx context.Context, id string) (*projectv1.Project, error)
}
type PermissionChecker interface {
	Check(ctx context.Context, userID string, resourceType string, resourceID string, action string) (bool, error)
}
type UserRepo interface {
	FindByID(ctx context.Context, id string) (*user.User, error)
}

type TaskRepo interface {
	Create(ctx context.Context, project *projectv1.Task) error
	FindByID(ctx context.Context, id string) (*projectv1.Task, error)
	Update(ctx context.Context, project *projectv1.Task) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, filter *TaskFilter) ([]*projectv1.Task, error)
	UpdateStatus(ctx context.Context, id string, status projectv1.TaskStatus, actualHours float64) error
	UpdateLabor(ctx context.Context, id string, activityTypeID string, plannedHours, actualHours float64) error
	Assign(ctx context.Context, id string, assignedID string) (*projectv1.Task, error)
}

func (s *Service) CreateTask(ctx context.Context, req *projectv1.CreateTaskRequest) (*projectv1.Task, error) {
	userID, err := util.GetFromContext(ctx, "user_id")
	if err != nil {
		return nil, fmt.Errorf("failed to get user_id: %w", err)
	}
	log.Printf("[DEBUG] CreateTask: userID=%s, projectID=%s", userID, req.ProjectId)

	ok, err := s.authz.Check(ctx, userID, authz.ResourceTask, req.ProjectId, authz.ActionCreate)
	if err != nil || !ok {
		log.Printf("[DEBUG] CreateTask: permission denied for user %s on project %s: ok=%v, err=%v", userID, req.ProjectId, ok, err)
		return nil, errs.ErrNoPermission
	}
	log.Printf("[DEBUG] CreateTask: permission granted for user %s on project %s", userID, req.ProjectId)

	if req.AssignedTo == "" {
		return nil, fmt.Errorf("%w:assigned name required", errs.ErrInvalidInput)
	}
	if req.Title == "" {
		return nil, fmt.Errorf("%w: title required", errs.ErrInvalidInput)
	}
	if req.ProjectId == "" {
		return nil, fmt.Errorf("%w: project id required", errs.ErrInvalidInput)
	}

	if err := s.validateActivityType(ctx, req.ActivityTypeId); err != nil {
		return nil, err
	}

	task := &projectv1.Task{
		ProjectId:      req.ProjectId,
		Title:          req.Title,
		Description:    req.Description,
		Status:         projectv1.TaskStatus_TASK_STATUS_TODO,
		Priority:       req.Priority,
		Deadline:       req.Deadline,
		AssignedTo:     req.AssignedTo,
		CreatedBy:      userID,
		ParentTaskId:   req.ParentTaskId,
		ActivityTypeId: req.ActivityTypeId,
		PlannedHours:   req.PlannedHours,
	}

	if err := s.taskRepo.Create(ctx, task); err != nil {
		return nil, fmt.Errorf("failed to create task in service: %w", err)
	}

	if s.events != nil {
		assigneeDept, assigneeName, assigneeEmail := s.assigneeProfile(ctx, task.AssignedTo)
		projectName := s.projectName(ctx, task.ProjectId)
		event := map[string]interface{}{
			"event_type":         "task.created",
			"occurred_at":        time.Now().UTC().Format(time.RFC3339Nano),
			"actor_user_id":      userID,
			"task_id":            task.Id,
			"project_id":         task.ProjectId,
			"project_name":       projectName,
			"user_id":            task.AssignedTo,
			"department_id":      assigneeDept,
			"assignee_full_name": assigneeName,
			"assignee_email":     assigneeEmail,
			"title":         task.Title,
			"task_title":    task.Title,
			"description":   task.Description,
			"status":        int32(task.Status),
			"priority":      int32(task.Priority),
			"deadline": tsToFormat(task.Deadline),
		}
		s.applyLaborEventFields(event, task)
		if err := s.events.Publish(ctx, "task.created", event); err != nil {
			log.Printf("Failed to publish task.created: %v", err)
		}
	}

	return task, nil

}

func (s *Service) GetTask(ctx context.Context, req *projectv1.GetTaskRequest) (*projectv1.Task, error) {
	userID, err := util.GetFromContext(ctx, "user_id")
	if err != nil {
		return nil, fmt.Errorf("failed to get user_id: %w", err)
	}

	ok, err := s.authz.Check(ctx, userID, authz.ResourceTask, req.Id, authz.ActionView)
	if err != nil || !ok {
		return nil, errs.ErrNoPermission
	}

	if req.Id == "" {
		return nil, fmt.Errorf("%w: task id required", errs.ErrInvalidInput)
	}

	task, err := s.taskRepo.FindByID(ctx, req.Id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errs.ErrProjectNotFound
		}
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	return task, nil
}

func (s *Service) UpdateTask(ctx context.Context, req *projectv1.UpdateTaskRequest) (*projectv1.Task, error) {
	userID, err := util.GetFromContext(ctx, "user_id")
	if err != nil {
		return nil, fmt.Errorf("failed to get user_id: %w", err)
	}
	ok, err := s.authz.Check(ctx, userID, authz.ResourceTask, req.Id, authz.ActionEdit)
	if err != nil || !ok {
		return nil, errs.ErrNoPermission
	}

	if req.Id == "" {
		return nil, fmt.Errorf("%w: task id required", errs.ErrInvalidInput)
	}

	existing, err := s.taskRepo.FindByID(ctx, req.Id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errs.ErrTaskNotFound
		}
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	activityTypeID := existing.ActivityTypeId
	if req.ActivityTypeId != "" {
		activityTypeID = req.ActivityTypeId
	}
	if err := s.validateActivityType(ctx, activityTypeID); err != nil {
		return nil, err
	}
	plannedHours := existing.PlannedHours
	if req.PlannedHours > 0 {
		plannedHours = req.PlannedHours
	}
	priority := existing.Priority
	if req.Priority != projectv1.TaskPriority_TASK_PRIORITY_UNSPECIFIED {
		priority = req.Priority
	}

	updatedTask := &projectv1.Task{
		Id:             existing.Id,
		ProjectId:      existing.ProjectId,
		Title:          util.NonEmpty(req.Title, existing.Title),
		Description:    util.NonEmpty(req.Description, existing.Description),
		Status:         existing.Status,
		Priority:       priority,
		Deadline:       util.FirstNonNil(req.Deadline, existing.Deadline),
		AssignedTo:     existing.AssignedTo,
		ParentTaskId:   existing.ParentTaskId,
		ActivityTypeId: activityTypeID,
		PlannedHours:   plannedHours,
		ActualHours:    existing.ActualHours,
		UpdatedAt:      timestamppb.Now(),
		CreatedBy:      existing.CreatedBy,
	}

	if err := s.taskRepo.Update(ctx, updatedTask); err != nil {
		return nil, err
	}

	laborChanged := existing.ActivityTypeId != updatedTask.ActivityTypeId || existing.PlannedHours != updatedTask.PlannedHours
	if s.events != nil && laborChanged {
		assigneeDept, assigneeName, assigneeEmail := s.assigneeProfile(ctx, updatedTask.AssignedTo)
		projectName := s.projectName(ctx, existing.ProjectId)
		event := map[string]interface{}{
			"event_type":             "task.updated",
			"occurred_at":            time.Now().UTC().Format(time.RFC3339Nano),
			"actor_user_id":          userID,
			"task_id":                updatedTask.Id,
			"project_id":             existing.ProjectId,
			"project_name":           projectName,
			"task_title":             updatedTask.Title,
			"assignee_user_id":       updatedTask.AssignedTo,
			"assignee_department_id": assigneeDept,
			"assignee_full_name":     assigneeName,
			"assignee_email":         assigneeEmail,
		}
		s.applyLaborEventFields(event, updatedTask)
		if err := s.events.Publish(ctx, "task.updated", event); err != nil {
			log.Printf("Failed to publish task.updated: %v", err)
		}
	}

	if s.events != nil && deadlineChanged(existing.Deadline, updatedTask.Deadline) {
		assigneeDept, assigneeName, assigneeEmail := s.assigneeProfile(ctx, updatedTask.AssignedTo)
		projectName := s.projectName(ctx, existing.ProjectId)
		event := map[string]interface{}{
			"event_type":             "task.deadline_changed",
			"occurred_at":            time.Now().UTC().Format(time.RFC3339Nano),
			"actor_user_id":          userID,
			"task_id":                updatedTask.Id,
			"project_id":             existing.ProjectId,
			"project_name":           projectName,
			"task_title":             updatedTask.Title,
			"assignee_user_id":       updatedTask.AssignedTo,
			"assignee_department_id": assigneeDept,
			"assignee_full_name":     assigneeName,
			"assignee_email":         assigneeEmail,
			"old_deadline":           tsToFormat(existing.Deadline),
			"new_deadline":           tsToFormat(updatedTask.Deadline),
		}
		if err := s.events.Publish(ctx, "task.deadline_changed", event); err != nil {
			log.Printf("Failed to publish task.deadline_changed: %v", err)
		}
	}

	return updatedTask, nil
}

func (s *Service) DeleteTask(ctx context.Context, req *projectv1.DeleteTaskRequest) (*emptypb.Empty, error) {
	userID, err := util.GetFromContext(ctx, "user_id")
	if err != nil {
		return nil, fmt.Errorf("failed to get user_id: %w", err)
	}
	ok, err := s.authz.Check(ctx, userID, authz.ResourceTask, req.Id, authz.ActionDelete)
	if err != nil || !ok {
		return nil, errs.ErrNoPermission
	}

	if req.Id == "" {
		return nil, fmt.Errorf("%w: task id required", errs.ErrInvalidInput)
	}

	if err := s.taskRepo.Delete(ctx, req.Id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &emptypb.Empty{}, nil
		}
		return nil, fmt.Errorf("failed to delete task: %w", err)
	}

	return &emptypb.Empty{}, nil
}

func (s *Service) UpdateTaskStatus(ctx context.Context, req *projectv1.UpdateTaskStatusRequest) (*projectv1.Task, error) {
	userID, err := util.GetFromContext(ctx, "user_id")
	if err != nil {
		return nil, fmt.Errorf("failed to get user_id: %w", err)
	}
	ok, err := s.authz.Check(ctx, userID, authz.ResourceTask, req.Id, authz.ActionChangeStatus)
	if err != nil || !ok {
		return nil, errs.ErrNoPermission
	}

	if req.Id == "" {
		return nil, fmt.Errorf("%w: task id required", errs.ErrInvalidInput)
	}
	if req.Status == projectv1.TaskStatus_TASK_STATUS_UNSPECIFIED {
		return nil, fmt.Errorf("%w: project status required", errs.ErrInvalidInput)
	}

	existing, err := s.taskRepo.FindByID(ctx, req.Id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errs.ErrTaskNotFound
		}
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	actualHours := req.ActualHours
	if req.Status == projectv1.TaskStatus_TASK_STATUS_COMPLETED && actualHours <= 0 {
		actualHours = existing.ActualHours
	}

	if err := s.taskRepo.UpdateStatus(ctx, req.Id, req.Status, actualHours); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errs.ErrProjectNotFound
		}
		return nil, fmt.Errorf("failed to update task status: %w", err)
	}

	updated, err := s.taskRepo.FindByID(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	if s.events != nil && existing.Status != updated.Status {
		assigneeDept, assigneeName, assigneeEmail := s.assigneeProfile(ctx, updated.AssignedTo)
		projectName := s.projectName(ctx, updated.ProjectId)
		event := map[string]interface{}{
			"event_type":             "task.status_changed",
			"occurred_at":            time.Now().UTC().Format(time.RFC3339Nano),
			"actor_user_id":          userID,
			"task_id":                updated.Id,
			"project_id":             updated.ProjectId,
			"project_name":           projectName,
			"task_title":             updated.Title,
			"from_status":            int32(existing.Status),
			"to_status":              int32(updated.Status),
			"assignee_user_id":       updated.AssignedTo,
			"assignee_department_id": assigneeDept,
			"assignee_full_name":     assigneeName,
			"assignee_email":         assigneeEmail,
			"deadline": tsToFormat(updated.Deadline),
		}
		s.applyLaborEventFields(event, updated)
		if actualHours > 0 {
			event["actual_hours"] = actualHours
		}
		if err := s.events.Publish(ctx, "task.status_changed", event); err != nil {
			log.Printf("Failed to publish task.status_changed: %v", err)
		}
	}

	return updated, nil
}

func (s *Service) UpdateTaskLabor(ctx context.Context, req *projectv1.UpdateTaskLaborRequest) (*projectv1.Task, error) {
	userID, err := util.GetFromContext(ctx, "user_id")
	if err != nil {
		return nil, fmt.Errorf("failed to get user_id: %w", err)
	}
	ok, err := s.authz.Check(ctx, userID, authz.ResourceTask, req.Id, authz.ActionUpdateLabor)
	if err != nil || !ok {
		return nil, errs.ErrNoPermission
	}
	if req.Id == "" {
		return nil, fmt.Errorf("%w: task id required", errs.ErrInvalidInput)
	}

	existing, err := s.taskRepo.FindByID(ctx, req.Id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errs.ErrTaskNotFound
		}
		return nil, fmt.Errorf("failed to get task: %w", err)
	}
	activityTypeID := existing.ActivityTypeId
	if req.ActivityTypeId != "" {
		activityTypeID = req.ActivityTypeId
	}
	if err := s.validateActivityType(ctx, activityTypeID); err != nil {
		return nil, err
	}
	plannedHours := existing.PlannedHours
	if req.PlannedHours > 0 {
		plannedHours = req.PlannedHours
	}
	actualHours := existing.ActualHours
	if req.ActualHours > 0 {
		actualHours = req.ActualHours
	}
	if err := s.taskRepo.UpdateLabor(ctx, req.Id, activityTypeID, plannedHours, actualHours); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errs.ErrTaskNotFound
		}
		return nil, fmt.Errorf("failed to update task labor: %w", err)
	}
	updated, err := s.taskRepo.FindByID(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	if s.events != nil {
		assigneeDept, assigneeName, assigneeEmail := s.assigneeProfile(ctx, updated.AssignedTo)
		projectName := s.projectName(ctx, updated.ProjectId)
		event := map[string]interface{}{
			"event_type":             "task.updated",
			"occurred_at":            time.Now().UTC().Format(time.RFC3339Nano),
			"actor_user_id":          userID,
			"task_id":                updated.Id,
			"project_id":             updated.ProjectId,
			"project_name":           projectName,
			"task_title":             updated.Title,
			"assignee_user_id":       updated.AssignedTo,
			"assignee_department_id": assigneeDept,
			"assignee_full_name":     assigneeName,
			"assignee_email":         assigneeEmail,
			"deadline":               tsToFormat(updated.Deadline),
			"status":                 int32(updated.Status),
		}
		s.applyLaborEventFields(event, updated)
		if err := s.events.Publish(ctx, "task.updated", event); err != nil {
			log.Printf("Failed to publish task.updated: %v", err)
		}
	}
	return updated, nil
}

func (s *Service) ListTasks(ctx context.Context, req *projectv1.ListTasksRequest) (*projectv1.ListTasksResponse, error) {
	userID, err := util.GetFromContext(ctx, "user_id")
	if err != nil {
		return nil, fmt.Errorf("failed to get user_id: %w", err)
	}
	ok, err := s.authz.Check(ctx, userID, authz.ResourceProject, req.ProjectId, authz.ActionView)
	if err != nil || !ok {
		return nil, errs.ErrNoPermission
	}

	_, err = s.projectRepo.FindByID(ctx, req.ProjectId)
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	filter := &TaskFilter{
		ProjectID: req.ProjectId,
	}
	if req.PriorityFilter != projectv1.TaskPriority_TASK_PRIORITY_UNSPECIFIED {
		filter.Priority = &req.PriorityFilter
	}
	if req.StatusFilter != projectv1.TaskStatus_TASK_STATUS_UNSPECIFIED {
		filter.Status = &req.StatusFilter
	}
	if req.AssignedToFilter != "" {
		filter.AssignedTo = &req.AssignedToFilter
	}
	if req.ParentTaskId != "" {
		filter.ParentTaskID = &req.ParentTaskId
	}

	task, err := s.taskRepo.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}

	return &projectv1.ListTasksResponse{
		Tasks:      task,
		TotalCount: int32(len(task)),
	}, nil

}

func (s *Service) AssignTask(ctx context.Context, req *projectv1.AssignTaskRequest) (*projectv1.Task, error) {
	userID, err := util.GetFromContext(ctx, "user_id")
	if err != nil {
		return nil, fmt.Errorf("failed to get user_id: %w", err)
	}
	ok, err := s.authz.Check(ctx, userID, authz.ResourceTask, req.TaskId, authz.ActionAssign)
	if err != nil || !ok {
		return nil, errs.ErrNoPermission
	}

	if req.TaskId == "" {
		return nil, fmt.Errorf("%w: task id required", errs.ErrInvalidInput)
	}

	if req.AssigneeId == "" {
		return nil, fmt.Errorf("%w: assignee id required", errs.ErrInvalidInput)
	}

	existing, err := s.taskRepo.FindByID(ctx, req.TaskId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errs.ErrTaskNotFound
		}
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	task, err := s.taskRepo.Assign(ctx, existing.Id, req.AssigneeId)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to assign task", err)
	}

	if s.events != nil && existing.AssignedTo != req.AssigneeId {
		toDept, toName, toEmail := s.assigneeProfile(ctx, req.AssigneeId)
		projectName := s.projectName(ctx, task.ProjectId)
		event := map[string]interface{}{
			"event_type":         "task.assigned",
			"occurred_at":        time.Now().UTC().Format(time.RFC3339Nano),
			"actor_user_id":      userID,
			"task_id":            task.Id,
			"project_id":         task.ProjectId,
			"project_name":       projectName,
			"task_title":         task.Title,
			"from_user_id":       existing.AssignedTo,
			"to_user_id":         req.AssigneeId,
			"user_id":            req.AssigneeId,
			"to_department_id":   toDept,
			"assignee_full_name": toName,
			"assignee_email":     toEmail,
			"deadline":           tsToFormat(task.Deadline),
			"status":             int32(task.Status),
		}
		if err := s.events.Publish(ctx, "task.assigned", event); err != nil {
			log.Printf("Failed to publish task.assigned: %v", err)
		}
	}
	return task, nil

}

func tsToFormat(ts *timestamppb.Timestamp) string {
	if ts == nil {
		return ""
	}
	return ts.AsTime().UTC().Format(time.RFC3339Nano)
}

func (s *Service) assigneeProfile(ctx context.Context, userID string) (departmentID, fullName, email string) {
	if s.userRepo == nil || userID == "" {
		return "", "", ""
	}
	u, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return "", "", ""
	}
	if u.DepartmentID != nil {
		departmentID = *u.DepartmentID
	}
	return departmentID, u.FullName, u.Email
}

func (s *Service) projectName(ctx context.Context, projectID string) string {
	if s.projectRepo == nil || projectID == "" {
		return ""
	}
	project, err := s.projectRepo.FindByID(ctx, projectID)
	if err != nil {
		log.Printf("failed to enrich task event with project name: %v", err)
		return ""
	}
	return project.Name
}

func (s *Service) validateActivityType(ctx context.Context, activityTypeID string) error {
	if activityTypeID == "" || s.activityRepo == nil {
		return nil
	}
	ok, err := s.activityRepo.Exists(ctx, activityTypeID)
	if err != nil {
		return fmt.Errorf("failed to validate activity type: %w", err)
	}
	if !ok {
		return fmt.Errorf("%w: unknown activity type", errs.ErrInvalidInput)
	}
	return nil
}

func (s *Service) applyLaborEventFields(event map[string]interface{}, task *projectv1.Task) {
	if task == nil {
		return
	}
	if task.ActivityTypeId != "" {
		event["activity_type_id"] = task.ActivityTypeId
	}
	if task.PlannedHours > 0 {
		event["planned_hours"] = task.PlannedHours
	}
	if task.ActualHours > 0 {
		event["actual_hours"] = task.ActualHours
	}
}

func deadlineChanged(a *timestamppb.Timestamp, b *timestamppb.Timestamp) bool {
	if a == nil && b == nil {
		return false
	}
	if a == nil || b == nil {
		return true
	}
	return !a.AsTime().Equal(b.AsTime())
}

func (s *Service) ListMyTasks(ctx context.Context, req *projectv1.ListMyTasksRequest) (*projectv1.ListTasksResponse, error) {
	userID, err := util.GetFromContext(ctx, "user_id")
	if err != nil {
		return nil, fmt.Errorf("failed to get user_id: %w", err)
	}

	filter := &TaskFilter{
		AssignedTo: &userID,
	}

	if req.StatusFilter != projectv1.TaskStatus_TASK_STATUS_UNSPECIFIED {
		filter.Status = &req.StatusFilter
	}

	tasks, err := s.taskRepo.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}

	return &projectv1.ListTasksResponse{
		Tasks:      tasks,
		TotalCount: int32(len(tasks)),
	}, nil
}
