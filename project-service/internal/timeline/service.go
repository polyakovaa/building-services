package timeline

import (
	projectv1 "building-services/gen/project/v1"
	"building-services/project-service/internal/errs"
	"building-services/project-service/internal/util"
	"context"
	"fmt"
)

type Service struct {
	timelineRepo *Repository
	projectRepo  ProjectRepo
}

func NewService(timelineRepo *Repository, projectRepo ProjectRepo) *Service {
	return &Service{
		timelineRepo: timelineRepo,
		projectRepo:  projectRepo,
	}
}

type ProjectRepo interface {
	Exists(ctx context.Context, id string) (bool, error)
}

func (s *Service) GetTimeline(ctx context.Context, req *projectv1.GetTimelineRequest) (*projectv1.ProjectTimeline, error) {
	if req.ProjectId == "" {
		return nil, fmt.Errorf("%w: project id required", errs.ErrInvalidInput)
	}

	exists, err := s.projectRepo.Exists(ctx, req.ProjectId)
	if err != nil {
		return nil, fmt.Errorf("failed to check project: %w", err)
	}
	if !exists {
		return nil, errs.ErrProjectNotFound
	}

	timeline, err := s.timelineRepo.Get(ctx, req.ProjectId)
	if err != nil {
		return nil, fmt.Errorf("failed to get timeline: %w", err)
	}

	return timeline, nil
}

func (s *Service) UpdateTimeline(ctx context.Context, req *projectv1.UpdateTimelineRequest) (*projectv1.ProjectTimeline, error) {
	userID, err := util.GetFromContext(ctx, "user_id")
	if err != nil {
		return nil, fmt.Errorf("failed to get user_id: %w", err)
	}

	if req.ProjectId == "" {
		return nil, fmt.Errorf("%w: project id required", errs.ErrInvalidInput)
	}

	exists, err := s.projectRepo.Exists(ctx, req.ProjectId)
	if err != nil {
		return nil, fmt.Errorf("failed to check project: %w", err)
	}
	if !exists {
		return nil, errs.ErrProjectNotFound
	}

	timeline, err := s.timelineRepo.Get(ctx, req.ProjectId)
	if err != nil {
		return nil, fmt.Errorf("failed to get timeline: %w", err)
	}

	if req.ContractDate != nil {
		timeline.ContractDate = req.ContractDate
	}
	if req.WorkStartDate != nil {
		timeline.WorkStartDate = req.WorkStartDate
	}
	if req.WorkEndDate != nil {
		timeline.WorkEndDate = req.WorkEndDate
	}
	if req.HandoverDate != nil {
		timeline.HandoverDate = req.HandoverDate
	}
	if req.CommentsDate != nil {
		timeline.CommentsDate = req.CommentsDate
	}
	if req.CommentsFixedDate != nil {
		timeline.CommentsFixedDate = req.CommentsFixedDate
	}
	if req.AcceptanceDate != nil {
		timeline.AcceptanceDate = req.AcceptanceDate
	}
	if req.FinalPaymentDate != nil {
		timeline.FinalPaymentDate = req.FinalPaymentDate
	}

	timeline.UpdatedBy = userID

	if err := s.timelineRepo.Upsert(ctx, timeline); err != nil {
		return nil, fmt.Errorf("failed to update timeline: %w", err)
	}

	return timeline, nil
}
