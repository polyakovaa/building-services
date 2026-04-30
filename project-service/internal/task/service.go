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

	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Service struct {
	taskRepo    TaskRepo
	projectRepo ProjectRepo
	authz       PermissionChecker
}

func NewService(taskRepo TaskRepo,
	projectRepo ProjectRepo,
	authz PermissionChecker) *Service {
	return &Service{
		taskRepo:    taskRepo,
		projectRepo: projectRepo,
		authz:       authz,
	}
}

type ProjectRepo interface {
	FindByID(ctx context.Context, id string) (*projectv1.Project, error)
}
type PermissionChecker interface {
	Check(ctx context.Context, userID string, resourceType string, resourceID string, action string) (bool, error)
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

	if err := s.taskRepo.UpdateStatus(ctx, req.Id, req.Status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errs.ErrProjectNotFound
		}
		return nil, fmt.Errorf("failed to update task status: %w", err)
	}

	return s.taskRepo.FindByID(ctx, req.Id)
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
	return task, nil

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
