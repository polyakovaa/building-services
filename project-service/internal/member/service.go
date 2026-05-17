package member

import (
	projectv1 "building-services/gen/project/v1"
	"building-services/project-service/internal/errs"
	"building-services/project-service/internal/util"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"building-services/project-service/internal/user"
	"building-services/project-service/internal/events"

	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Service struct {
	projectRepo ProjectRepo
	memberRepo  MemberRepo
	userRepo    UserRepo
	events      events.Publisher
}

func NewService(projectRepo ProjectRepo,
	memberRepo MemberRepo, userRepo UserRepo, eventPublisher events.Publisher) *Service {
	return &Service{
		projectRepo: projectRepo,
		memberRepo:  memberRepo,
		userRepo:    userRepo,
		events:      eventPublisher,
	}
}

type MemberRepo interface {
	Add(ctx context.Context, member *projectv1.ProjectMember) error
	FindByID(ctx context.Context, userID string) (*projectv1.ProjectMember, error)
	Update(ctx context.Context, member *projectv1.ProjectMember) error
	IsProjectMember(ctx context.Context, projectID, userID string) (*projectv1.ProjectMember, error)
	Remove(ctx context.Context, projectID, userID string) error
	GetProjectMembers(ctx context.Context, projectID string) ([]*projectv1.ProjectMember, error)
}
type UserRepo interface {
	FindByID(ctx context.Context, id string) (*user.User, error)
}

type ProjectRepo interface {
	FindByID(ctx context.Context, id string) (*projectv1.Project, error)
}

func (s *Service) AddMember(ctx context.Context, req *projectv1.AddMemberRequest) (*projectv1.ProjectMember, error) {
	if req.UserId == "" {
		return nil, fmt.Errorf("%w: user_id required", errs.ErrInvalidInput)
	}
	if req.ProjectId == "" {
		return nil, fmt.Errorf("%w: project_id required", errs.ErrInvalidInput)
	}
	user, err := s.userRepo.FindByID(ctx, req.UserId)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	member := &projectv1.ProjectMember{
		ProjectId: req.ProjectId,
		UserId:    req.UserId,
		JoinedAt:  timestamppb.Now(),
	}

	if user.DepartmentID != nil {
		member.DepartmentId = *user.DepartmentID
	}

	err = s.memberRepo.Add(ctx, member)
	if err != nil {
		return nil, fmt.Errorf("failed to add member: %w", err)
	}

	actorUserID, err := util.GetFromContext(ctx, "user_id")
	if err == nil && s.events != nil {
		projectName := s.projectName(ctx, member.ProjectId)
		event := map[string]interface{}{
			"event_type":    "project.member_added",
			"occurred_at":   time.Now().UTC().Format(time.RFC3339Nano),
			"actor_user_id": actorUserID,
			"project_id":    member.ProjectId,
			"project_name":  projectName,
			"user_id":       member.UserId,
			"department_id": member.DepartmentId,
		}
		if err := s.events.Publish(ctx, "project.member_added", event); err != nil {
			log.Printf("Failed to publish project.member_added: %v", err)
		}
	}

	return member, nil
}

func (s *Service) projectName(ctx context.Context, projectID string) string {
	if s.projectRepo == nil || projectID == "" {
		return ""
	}
	project, err := s.projectRepo.FindByID(ctx, projectID)
	if err != nil {
		log.Printf("failed to enrich member event with project name: %v", err)
		return ""
	}
	return project.Name
}

func (s *Service) UpdateMember(ctx context.Context, req *projectv1.UpdateMemberRequest) (*projectv1.ProjectMember, error) {
	if req.ProjectId == "" {
		return nil, fmt.Errorf("%w: project id required", errs.ErrInvalidInput)
	}

	if req.UserId == "" {
		return nil, fmt.Errorf("%w: user id required", errs.ErrInvalidInput)
	}

	existingProject, err := s.projectRepo.FindByID(ctx, req.ProjectId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errs.ErrProjectNotFound
		}
		return nil, fmt.Errorf("failed to get project: %w", err)
	}
	existingMember, err := s.memberRepo.FindByID(ctx, req.UserId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errs.ErrMemberNotFound
		}
		return nil, fmt.Errorf("failed to get member: %w", err)
	}

	updatedMember := &projectv1.ProjectMember{
		ProjectId:    util.NonEmpty(req.ProjectId, existingProject.Id),
		UserId:       util.NonEmpty(req.UserId, existingMember.UserId),
		DepartmentId: util.NonEmpty(req.DepartmentId, existingMember.DepartmentId),
		JoinedAt:     existingMember.JoinedAt,
	}

	if err := s.memberRepo.Update(ctx, updatedMember); err != nil {
		return nil, err
	}

	return updatedMember, nil

}

func (s *Service) RemoveMember(ctx context.Context, req *projectv1.RemoveMemberRequest) (*emptypb.Empty, error) {
	if req.ProjectId == "" {
		return nil, fmt.Errorf("%w: project id required", errs.ErrInvalidInput)
	}
	if req.UserId == "" {
		return nil, fmt.Errorf("%w: user id required", errs.ErrInvalidInput)
	}

	_, err := s.projectRepo.FindByID(ctx, req.ProjectId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errs.ErrProjectNotFound
		}
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	err = s.memberRepo.Remove(ctx, req.ProjectId, req.UserId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errs.ErrMemberNotFound
		}
		return nil, fmt.Errorf("failed to remove member: %w", err)
	}

	actorUserID, err := util.GetFromContext(ctx, "user_id")
	if err == nil && s.events != nil {
		event := map[string]interface{}{
			"event_type":    "project.member_removed",
			"occurred_at":   time.Now().UTC().Format(time.RFC3339Nano),
			"actor_user_id": actorUserID,
			"project_id":    req.ProjectId,
			"user_id":       req.UserId,
		}
		if err := s.events.Publish(ctx, "project.member_removed", event); err != nil {
			log.Printf("Failed to publish project.member_removed: %v", err)
		}
	}

	return &emptypb.Empty{}, nil

}

func (s *Service) ListMembers(ctx context.Context, req *projectv1.ListMembersRequest) (*projectv1.ListMembersResponse, error) {
	if req.ProjectId == "" {
		return nil, fmt.Errorf("%w: project id required", errs.ErrInvalidInput)
	}

	_, err := s.projectRepo.FindByID(ctx, req.ProjectId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errs.ErrProjectNotFound
		}
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	members, err := s.memberRepo.GetProjectMembers(ctx, req.ProjectId)
	if err != nil {
		return nil, fmt.Errorf("failed to list members: %w", err)
	}

	return &projectv1.ListMembersResponse{
		Members: members,
	}, nil
}

func (s *Service) GetMember(ctx context.Context, req *projectv1.GetMemberRequest) (*projectv1.ProjectMember, error) {
	if req.ProjectId == "" {
		return nil, fmt.Errorf("%w: project id required", errs.ErrInvalidInput)
	}
	if req.UserId == "" {
		return nil, fmt.Errorf("%w: user id required", errs.ErrInvalidInput)
	}

	project, err := s.projectRepo.FindByID(ctx, req.ProjectId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errs.ErrProjectNotFound
		}
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	member, err := s.memberRepo.IsProjectMember(ctx, project.Id, req.UserId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errs.ErrProjectNotFound
		}
		return nil, fmt.Errorf("failed to get member: %w", err)
	}

	return member, nil
}
