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
	taskRepo    TaskRepo
	projectRepo ProjectRepo
	userRepo    UserRepo
	authz       PermissionChecker
	events      events.Publisher
}

func NewService(taskRepo TaskRepo,
	projectRepo ProjectRepo,
	userRepo UserRepo,
	authz PermissionChecker,
	eventPublisher events.Publisher) *Service {
	return &Service{
		taskRepo:    taskRepo,
		projectRepo: projectRepo,
		userRepo:    userRepo,
		authz:       authz,
		events:      eventPublisher,
	}
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
	UpdateStatus(ctx context.Context, id string, status projectv1.TaskStatus) error
	Assign(ctx context.Context, id string, assignedID string) (*projectv1.Task, error)
}

func (s *Service) CreateTask(ctx context.Context, req *projectv1.CreateTaskRequest) (*projectv1.Task, error) {
	userID, err := util.GetFromContext(ctx, "user_id")
	if err != nil {
		return nil, fmt.Errorf("failed to get user_id: %w", err)
	}
	ok, err := s.authz.Check(ctx, userID, authz.ResourceTask, req.ProjectId, authz.ActionCreate)
	if err != nil || !ok {
		return nil, errs.ErrNoPermission
	}

	if req.AssignedTo == "" {
		return nil, fmt.Errorf("%w:assigned name required", errs.ErrInvalidInput)
	}
	if req.Title == "" {
		return nil, fmt.Errorf("%w: title required", errs.ErrInvalidInput)
	}
	if req.ProjectId == "" {
		return nil, fmt.Errorf("%w: project id required", errs.ErrInvalidInput)
	}

	task := &projectv1.Task{
		ProjectId:    req.ProjectId,
		Title:        req.Title,
		Description:  req.Description,
		Status:       projectv1.TaskStatus_TASK_STATUS_TODO,
		Priority:     req.Priority,
		Deadline:     req.Deadline,
		AssignedTo:   req.AssignedTo,
		CreatedBy:    userID,
		ParentTaskId: req.ParentTaskId,
	}

	if err := s.taskRepo.Create(ctx, task); err != nil {
		return nil, fmt.Errorf("failed to create task in service: %w", err)
	}

	// Publish task.created event
	if s.events != nil {
		assigneeDept := ""
		if s.userRepo != nil && task.AssignedTo != "" {
			if u, err := s.userRepo.FindByID(ctx, task.AssignedTo); err == nil && u.DepartmentID != nil {
				assigneeDept = *u.DepartmentID
			}
		}
		event := map[string]interface{}{
			"event_type":    "task.created",
			"occurred_at":   time.Now().UTC().Format(time.RFC3339Nano),
			"actor_user_id": userID,
			"task_id":       task.Id,
			"project_id":    task.ProjectId,
			"user_id":       task.AssignedTo,
			"department_id": assigneeDept,
			"title":         task.Title,
			"description":   task.Description,
			"status":        int32(task.Status),
			"priority":      int32(task.Priority),
			"deadline":      tsToFormat(task.Deadline),
		}
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

	updatedTask := &projectv1.Task{
		Id:          existing.Id,
		Title:       util.NonEmpty(req.Title, existing.Title),
		Description: util.NonEmpty(req.Description, existing.Description),
		Status:      existing.Status,
		Priority:    req.Priority,
		Deadline:    util.FirstNonNil(req.Deadline, existing.Deadline),
		AssignedTo:  existing.AssignedTo,
		UpdatedAt:   timestamppb.Now(),
		CreatedBy:   existing.CreatedBy,
	}

	if err := s.taskRepo.Update(ctx, updatedTask); err != nil {
		return nil, err
	}

	if s.events != nil && deadlineChanged(existing.Deadline, updatedTask.Deadline) {
		assigneeDept := ""
		if s.userRepo != nil && updatedTask.AssignedTo != "" {
			if u, err := s.userRepo.FindByID(ctx, updatedTask.AssignedTo); err == nil && u.DepartmentID != nil {
				assigneeDept = *u.DepartmentID
			}
		}
		event := map[string]interface{}{
			"event_type":             "task.deadline_changed",
			"occurred_at":            time.Now().UTC().Format(time.RFC3339Nano),
			"actor_user_id":          userID,
			"task_id":                updatedTask.Id,
			"project_id":             existing.ProjectId,
			"assignee_user_id":       updatedTask.AssignedTo,
			"assignee_department_id": assigneeDept,
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

	if err := s.taskRepo.UpdateStatus(ctx, req.Id, req.Status); err != nil {
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
		assigneeDept := ""
		if s.userRepo != nil && updated.AssignedTo != "" {
			if u, err := s.userRepo.FindByID(ctx, updated.AssignedTo); err == nil && u.DepartmentID != nil {
				assigneeDept = *u.DepartmentID
			}
		}
		event := map[string]interface{}{
			"event_type":             "task.status_changed",
			"occurred_at":            time.Now().UTC().Format(time.RFC3339Nano),
			"actor_user_id":          userID,
			"task_id":                updated.Id,
			"project_id":             updated.ProjectId,
			"from_status":            int32(existing.Status),
			"to_status":              int32(updated.Status),
			"assignee_user_id":       updated.AssignedTo,
			"assignee_department_id": assigneeDept,
			"deadline":               tsToFormat(updated.Deadline),
		}
		if err := s.events.Publish(ctx, "task.status_changed", event); err != nil {
			log.Printf("Failed to publish task.status_changed: %v", err)
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
		toDept := ""
		if s.userRepo != nil {
			if u, err := s.userRepo.FindByID(ctx, req.AssigneeId); err == nil && u.DepartmentID != nil {
				toDept = *u.DepartmentID
			}
		}
		event := map[string]interface{}{
			"event_type":       "task.assigned",
			"occurred_at":      time.Now().UTC().Format(time.RFC3339Nano),
			"actor_user_id":    userID,
			"task_id":          task.Id,
			"project_id":       task.ProjectId,
			"from_user_id":     existing.AssignedTo,
			"to_user_id":       req.AssigneeId,
			"to_department_id": toDept,
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
