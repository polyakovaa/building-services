package member

import (
	projectv1 "building-services/gen/project/v1"
	"building-services/project-service/internal/util"
	"context"
	"database/sql"
	"errors"
	"fmt"

	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	ErrProjectNotFound = errors.New("project not found")
	ErrMemberNotFound  = errors.New("member not found")
	ErrNoPermission    = errors.New("permission denied")
	ErrInvalidInput    = errors.New("invalid input")
)

type Service struct {
	projectRepo ProjectRepo
	memberRepo  MemberRepo
}

func NewService(projectRepo ProjectRepo,
	memberRepo MemberRepo) *Service {
	return &Service{
		projectRepo: projectRepo,
		memberRepo:  memberRepo,
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

type ProjectRepo interface {
	FindByID(ctx context.Context, id string) (*projectv1.Project, error)
}

func (s *Service) AddMember(ctx context.Context, req *projectv1.AddMemberRequest) (*projectv1.ProjectMember, error) {

	member := &projectv1.ProjectMember{
		ProjectId:    req.ProjectId,
		UserId:       req.UserId,
		DepartmentId: req.DepartmentId,
		JoinedAt:     timestamppb.Now()}

	err := s.memberRepo.Add(ctx, member)
	if err != nil {
		return nil, fmt.Errorf("failed to add member: %w", err)
	}

	return member, nil
}

func (s *Service) UpdateMember(ctx context.Context, req *projectv1.UpdateMemberRequest) (*projectv1.ProjectMember, error) {
	if req.ProjectId == "" {
		return nil, fmt.Errorf("%w: project id required", ErrInvalidInput)
	}

	if req.UserId == "" {
		return nil, fmt.Errorf("%w: user id required", ErrInvalidInput)
	}

	existingProject, err := s.projectRepo.FindByID(ctx, req.ProjectId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrProjectNotFound
		}
		return nil, fmt.Errorf("failed to get project: %w", err)
	}
	existingMember, err := s.memberRepo.FindByID(ctx, req.UserId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrMemberNotFound
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
		return nil, fmt.Errorf("%w: project id required", ErrInvalidInput)
	}
	if req.UserId == "" {
		return nil, fmt.Errorf("%w: user id required", ErrInvalidInput)
	}

	_, err := s.projectRepo.FindByID(ctx, req.ProjectId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrProjectNotFound
		}
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	err = s.memberRepo.Remove(ctx, req.ProjectId, req.UserId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrMemberNotFound
		}
		return nil, fmt.Errorf("failed to remove member: %w", err)
	}

	return &emptypb.Empty{}, nil

}

func (s *Service) ListMembers(ctx context.Context, req *projectv1.ListMembersRequest) (*projectv1.ListMembersResponse, error) {
	if req.ProjectId == "" {
		return nil, fmt.Errorf("%w: project id required", ErrInvalidInput)
	}

	_, err := s.projectRepo.FindByID(ctx, req.ProjectId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrProjectNotFound
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
		return nil, fmt.Errorf("%w: project id required", ErrInvalidInput)
	}
	if req.UserId == "" {
		return nil, fmt.Errorf("%w: user id required", ErrInvalidInput)
	}

	project, err := s.projectRepo.FindByID(ctx, req.ProjectId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrProjectNotFound
		}
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	member, err := s.memberRepo.IsProjectMember(ctx, project.Id, req.UserId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrProjectNotFound
		}
		return nil, fmt.Errorf("failed to get member: %w", err)
	}

	return member, nil
}
