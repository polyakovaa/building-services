package project

import (
	projectv1 "building-services/gen/project/v1"
	"context"
	"errors"
	"log"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
		log.Printf("[ERROR] CreateProject failed: %v", err)
		switch {
		case errors.Is(err, ErrInvalidInput):
			return nil, status.Error(codes.InvalidArgument, err.Error())
		case errors.Is(err, ErrNoPermission):
			return nil, status.Error(codes.PermissionDenied, err.Error())
		case errors.Is(err, ErrProjectNotFound):
			return nil, status.Error(codes.NotFound, err.Error())
		default:
			return nil, status.Error(codes.Internal, "internal server error")
		}
	}

	log.Printf("[SUCCESS] CreateProject: id=%s, duration=%v", project.Id, time.Since(start))
	return project, nil

}
