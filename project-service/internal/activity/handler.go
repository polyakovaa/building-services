package activity

import (
	"context"
	"log"
	"time"

	projectv1 "building-services/gen/project/v1"
	"building-services/project-service/internal/errs"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Handler struct {
	projectv1.UnimplementedActivityTypeServiceServer
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) ListActivityTypes(ctx context.Context, req *projectv1.ListActivityTypesRequest) (*projectv1.ListActivityTypesResponse, error) {
	start := time.Now()
	log.Printf("[REQUEST] ListActivityTypes")
	resp, err := h.service.ListActivityTypes(ctx, req)
	if err != nil {
		log.Printf("[ERROR] ListActivityTypes: %v", err)
		return nil, status.Error(codes.Internal, err.Error())
	}
	log.Printf("[SUCCESS] ListActivityTypes: count=%d duration=%v", len(resp.ActivityTypes), time.Since(start))
	return resp, nil
}

func (h *Handler) CreateActivityType(ctx context.Context, req *projectv1.CreateActivityTypeRequest) (*projectv1.ActivityType, error) {
	start := time.Now()
	log.Printf("[REQUEST] CreateActivityType: name=%s", req.GetName())
	resp, err := h.service.CreateActivityType(ctx, req)
	if err != nil {
		if grpcErr := errs.Handle(err, "CreateActivityType"); grpcErr != nil {
			return nil, grpcErr
		}
	}
	log.Printf("[SUCCESS] CreateActivityType: id=%s duration=%v", resp.GetId(), time.Since(start))
	return resp, nil
}
