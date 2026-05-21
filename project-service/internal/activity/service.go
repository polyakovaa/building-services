package activity

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	projectv1 "building-services/gen/project/v1"
	"building-services/project-service/internal/authz"
	"building-services/project-service/internal/errs"
	"building-services/project-service/internal/events"
	"building-services/project-service/internal/util"

	"github.com/google/uuid"
)

type PermissionChecker interface {
	Check(ctx context.Context, userID string, resourceType string, resourceID string, action string) (bool, error)
}

type Service struct {
	repo   *Repository
	authz  PermissionChecker
	events events.Publisher
}

func NewService(repo *Repository, authz PermissionChecker, eventPublisher events.Publisher) *Service {
	return &Service{repo: repo, authz: authz, events: eventPublisher}
}

func (s *Service) ListActivityTypes(ctx context.Context, _ *projectv1.ListActivityTypesRequest) (*projectv1.ListActivityTypesResponse, error) {
	types, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	return &projectv1.ListActivityTypesResponse{ActivityTypes: types}, nil
}

func (s *Service) CreateActivityType(ctx context.Context, req *projectv1.CreateActivityTypeRequest) (*projectv1.ActivityType, error) {
	userID, err := util.GetFromContext(ctx, "user_id")
	if err != nil {
		return nil, fmt.Errorf("failed to get user_id: %w", err)
	}
	ok, err := s.authz.Check(ctx, userID, authz.ResourceActivityType, "", authz.ActionCreate)
	if err != nil || !ok {
		return nil, errs.ErrNoPermission
	}

	name := strings.TrimSpace(req.GetName())
	if name == "" {
		return nil, fmt.Errorf("%w: activity type name required", errs.ErrInvalidInput)
	}

	sortOrder, err := s.repo.NextSortOrder(ctx)
	if err != nil {
		return nil, err
	}

	at := &projectv1.ActivityType{
		Id:        uuid.New().String(),
		Name:      name,
		SortOrder: sortOrder,
	}
	if err := s.repo.Create(ctx, at); err != nil {
		return nil, err
	}

	if s.events != nil {
		event := map[string]interface{}{
			"event_type":  "activity_type.created",
			"occurred_at": time.Now().UTC().Format(time.RFC3339Nano),
			"actor_user_id": userID,
			"activity_type_id": at.Id,
			"id":          at.Id,
			"name":        at.Name,
			"sort_order":  at.SortOrder,
		}
		if err := s.events.Publish(ctx, "activity_type.created", event); err != nil {
			log.Printf("Failed to publish activity_type.created: %v", err)
		}
	}

	return at, nil
}
