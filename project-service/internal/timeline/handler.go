package timeline

import (
	projectv1 "building-services/gen/project/v1"
	"context"
	"log"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Handler struct {
	projectv1.UnimplementedProjectTimelineServiceServer
	service *Service
}

func NewHandler(s *Service) *Handler {
	return &Handler{
		service: s,
	}
}

func (h *Handler) GetTimeline(ctx context.Context, req *projectv1.GetTimelineRequest) (*projectv1.ProjectTimeline, error) {
	start := time.Now()
	log.Printf("[REQUEST] GetTimeline: project_id=%s", req.ProjectId)

	timeline, err := h.service.GetTimeline(ctx, req)
	if err != nil {
		log.Printf("[ERROR] GetTimeline failed: %v", err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	log.Printf("[SUCCESS] GetTimeline: project_id=%s, duration=%v", req.ProjectId, time.Since(start))
	return timeline, nil
}

func (h *Handler) UpdateTimeline(ctx context.Context, req *projectv1.UpdateTimelineRequest) (*projectv1.ProjectTimeline, error) {
	start := time.Now()
	log.Printf("[REQUEST] UpdateTimeline: project_id=%s", req.ProjectId)

	timeline, err := h.service.UpdateTimeline(ctx, req)
	if err != nil {
		log.Printf("[ERROR] UpdateTimeline failed: %v", err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	log.Printf("[SUCCESS] UpdateTimeline: project_id=%s, duration=%v", req.ProjectId, time.Since(start))
	return timeline, nil
}
