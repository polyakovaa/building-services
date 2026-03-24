package timeline

import (
	projectv1 "building-services/gen/project/v1"
)

type Handler struct {
	projectv1.UnimplementedProjectTimelineServiceServer
	service *Service
}

func NewTaskHandler(s *Service) *Handler {
	return &Handler{
		service: s,
	}
}
