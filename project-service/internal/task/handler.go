package task

import (
	projectv1 "building-services/gen/project/v1"
	"building-services/project-service/internal/errs"
	"context"
	"log"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Handler struct {
	projectv1.UnimplementedTaskServiceServer
	service *Service
}

func NewHandler(s *Service) *Handler {
	return &Handler{
		service: s,
	}
}

func (h *Handler) CreateTask(ctx context.Context, req *projectv1.CreateTaskRequest) (*projectv1.Task, error) {
	start := time.Now()

	log.Printf("[REQUEST] CreateTask: project_id=%s", req.ProjectId)
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is nil")
	}

	task, err := h.service.CreateTask(ctx, req)

	if err != nil {
		if grpcErr := errs.Handle(err, "CreateTask"); grpcErr != nil {
			return nil, grpcErr
		}
	}

	log.Printf("[SUCCESS] CreateTask: id=%s, duration=%v", task.Id, time.Since(start))
	return task, nil

}

func (h *Handler) GetTask(ctx context.Context, req *projectv1.GetTaskRequest) (*projectv1.Task, error) {
	start := time.Now()

	log.Printf("[REQUEST] GetTask: id: %s", req.Id)
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is nil")
	}

	log.Printf("Received request: %+v", req)

	task, err := h.service.GetTask(ctx, req)
	if err != nil {
		if grpcErr := errs.Handle(err, "GetTask"); grpcErr != nil {
			return nil, grpcErr
		}
	}

	log.Printf("[SUCCESS] GetTask: id=%s, duration=%v", task.Id, time.Since(start))
	return task, nil

}

func (h *Handler) ListTasks(ctx context.Context, req *projectv1.ListTasksRequest) (*projectv1.ListTasksResponse, error) {
	start := time.Now()
	log.Printf("[REQUEST] ListProjects: project_id=%v", req.ProjectId)

	resp, err := h.service.ListTasks(ctx, req)
	if err != nil {
		if grpcErr := errs.Handle(err, "ListTasks"); grpcErr != nil {
			return nil, grpcErr
		}
	}

	log.Printf("[SUCCESS] ListTasks: count=%d, duration=%v", len(resp.Tasks), time.Since(start))
	return resp, nil
}

func (h *Handler) UpdateTask(ctx context.Context, req *projectv1.UpdateTaskRequest) (*projectv1.Task, error) {
	start := time.Now()
	log.Printf("[REQUEST] UpdateTask: id: %s", req.Id)

	resp, err := h.service.UpdateTask(ctx, req)
	if err != nil {
		if grpcErr := errs.Handle(err, "UpdateTask"); grpcErr != nil {
			return nil, grpcErr
		}
	}

	log.Printf("[SUCCESS] UpdateTask: duration=%v", time.Since(start))
	return resp, nil
}

func (h *Handler) DeleteTask(ctx context.Context, req *projectv1.DeleteTaskRequest) (*emptypb.Empty, error) {
	start := time.Now()
	log.Printf("[REQUEST] DeleteTask: id: %s", req.Id)

	resp, err := h.service.DeleteTask(ctx, req)
	if err != nil {
		if grpcErr := errs.Handle(err, "DeleteTask"); grpcErr != nil {
			return nil, grpcErr
		}
	}

	log.Printf("[SUCCESS] DeleteTask: duration=%v", time.Since(start))
	return resp, nil
}

func (h *Handler) UpdateTaskStatus(ctx context.Context, req *projectv1.UpdateTaskStatusRequest) (*projectv1.Task, error) {
	start := time.Now()
	log.Printf("[REQUEST] UpdateTaskStatus: id: %s", req.Id)

	resp, err := h.service.UpdateTaskStatus(ctx, req)
	if err != nil {
		if grpcErr := errs.Handle(err, "UpdateTaskStatus"); grpcErr != nil {
			return nil, grpcErr
		}
	}

	log.Printf("[SUCCESS] UpdateTaskStatus: duration=%v", time.Since(start))
	return resp, nil
}

func (h *Handler) AssignTask(ctx context.Context, req *projectv1.AssignTaskRequest) (*projectv1.Task, error) {
	start := time.Now()
	log.Printf("[REQUEST] AssignTask: id: %s", req.TaskId)

	resp, err := h.service.AssignTask(ctx, req)
	if err != nil {
		if grpcErr := errs.Handle(err, "AssignTask"); grpcErr != nil {
			return nil, grpcErr
		}
	}

	log.Printf("[SUCCESS] AssignTask: duration=%v", time.Since(start))
	return resp, nil
}

func (h *Handler) ListMyTasks(ctx context.Context, req *projectv1.ListMyTasksRequest) (*projectv1.ListTasksResponse, error) {
	start := time.Now()
	log.Printf("[REQUEST] ListMyTasks: status_filter=%v", req.StatusFilter)

	resp, err := h.service.ListMyTasks(ctx, req)
	if err != nil {
		log.Printf("[ERROR] ListMyTasks failed: %v", err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	log.Printf("[SUCCESS] ListMyTasks: count=%d, duration=%v", len(resp.Tasks), time.Since(start))
	return resp, nil
}
