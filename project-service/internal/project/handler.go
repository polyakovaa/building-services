package project

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
	projectv1.UnimplementedProjectServiceServer
	service *Service
}

func NewHandler(s *Service) *Handler {
	return &Handler{
		service: s,
	}
}

func (h *Handler) CreateProject(ctx context.Context, req *projectv1.CreateProjectRequest) (*projectv1.Project, error) {
	start := time.Now()

	log.Printf("[REQUEST] CreateProject: name=%s, customer=%s", req.Name, req.Customer)
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is nil")
	}

	log.Printf("Received request: %+v", req)

	project, err := h.service.CreateProject(ctx, req)

	if err != nil {
		if grpcErr := errs.Handle(err, "CreateProject"); grpcErr != nil {
			return nil, grpcErr
		}
	}

	log.Printf("[SUCCESS] CreateProject: id=%s, duration=%v", project.Id, time.Since(start))
	return project, nil

}

func (h *Handler) GetProject(ctx context.Context, req *projectv1.GetProjectRequest) (*projectv1.Project, error) {
	start := time.Now()

	log.Printf("[REQUEST] GetProject: id: %s", req.Id)
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is nil")
	}

	log.Printf("Received request: %+v", req)

	project, err := h.service.GetProject(ctx, req)
	if err != nil {
		if grpcErr := errs.Handle(err, "GetProject"); grpcErr != nil {
			return nil, grpcErr
		}
	}

	log.Printf("[SUCCESS] GetProject: id=%s, duration=%v", project.Id, time.Since(start))
	return project, nil

}

func (h *Handler) ListProjects(ctx context.Context, req *projectv1.ListProjectsRequest) (*projectv1.ListProjectsResponse, error) {
	start := time.Now()
	log.Printf("[REQUEST] ListProjects: status_filter=%v, manager_id=%s", req.StatusFilter, req.ManagerId)

	resp, err := h.service.ListProjects(ctx, req)
	if err != nil {
		if grpcErr := errs.Handle(err, "ListProjects"); grpcErr != nil {
			return nil, grpcErr
		}
	}

	log.Printf("[SUCCESS] ListProjects: count=%d, duration=%v", len(resp.Projects), time.Since(start))
	return resp, nil
}

func (h *Handler) UpdateProject(ctx context.Context, req *projectv1.UpdateProjectRequest) (*projectv1.Project, error) {
	start := time.Now()
	log.Printf("[REQUEST] UpdateProject: id: %s", req.Id)

	resp, err := h.service.UpdateProject(ctx, req)
	if err != nil {
		if grpcErr := errs.Handle(err, "UpdateProject"); grpcErr != nil {
			return nil, grpcErr
		}
	}

	log.Printf("[SUCCESS] UpdateProjects: duration=%v", time.Since(start))
	return resp, nil
}

func (h *Handler) DeleteProject(ctx context.Context, req *projectv1.DeleteProjectRequest) (*emptypb.Empty, error) {
	start := time.Now()
	log.Printf("[REQUEST] DeleteProject: id: %s", req.Id)

	resp, err := h.service.DeleteProject(ctx, req)
	if err != nil {
		if grpcErr := errs.Handle(err, "DeleteProject"); grpcErr != nil {
			return nil, grpcErr
		}
	}

	log.Printf("[SUCCESS] DeleteProject: duration=%v", time.Since(start))
	return resp, nil
}

func (h *Handler) ChangeProjectStatus(ctx context.Context, req *projectv1.ChangeProjectStatusRequest) (*projectv1.Project, error) {
	start := time.Now()
	log.Printf("[REQUEST] ChangeProjectStatus: id: %s", req.Id)

	resp, err := h.service.ChangeProjectStatus(ctx, req)
	if err != nil {
		if grpcErr := errs.Handle(err, "ChangeProjectStatus"); grpcErr != nil {
			return nil, grpcErr
		}
	}

	log.Printf("[SUCCESS] ChangeProjectStatus: duration=%v", time.Since(start))
	return resp, nil
}
