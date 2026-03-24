package project

import (
	projectv1 "building-services/gen/project/v1"
	"building-services/project-service/internal/user"

	"building-services/project-service/internal/util"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"

	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	ErrProjectNotFound = errors.New("project not found")
	ErrNoPermission    = errors.New("permission denied")
	ErrInvalidInput    = errors.New("invalid input")
)

type Service struct {
	projectRepo ProjectRepo
	memberRepo  MemberRepo
	userRepo    UserRepo
	permissions PermissionChecker
}

func NewService(projectRepo ProjectRepo,
	memberRepo MemberRepo, userRepo UserRepo,
	permissions PermissionChecker) *Service {
	return &Service{
		projectRepo: projectRepo,
		memberRepo:  memberRepo,
		userRepo:    userRepo,
		permissions: permissions,
	}
}

type ProjectRepo interface {
	Create(ctx context.Context, project *projectv1.Project) error
	FindByID(ctx context.Context, id string) (*projectv1.Project, error)
	Update(ctx context.Context, project *projectv1.Project) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, filter map[string]interface{}) ([]*projectv1.Project, error)
	UpdateStatus(ctx context.Context, id string, status projectv1.ProjectStatus) error
}
type UserRepo interface {
	FindByID(ctx context.Context, id string) (*user.User, error)
}

type MemberRepo interface {
	Add(ctx context.Context, member *projectv1.ProjectMember) error
}

type PermissionChecker interface {
	CanCreateProject(ctx context.Context, userID string) bool
	CanGetProject(ctx context.Context, userID, projectID string) bool
	CanUpdateProject(ctx context.Context, userID, projectID string) bool
	CanDeleteProject(ctx context.Context, userID string) bool
	CanChangeStatus(ctx context.Context, userID, projectID string) bool
}

func (s *Service) CreateProject(ctx context.Context, req *projectv1.CreateProjectRequest) (*projectv1.Project, error) {

	if req.Name == "" {
		return nil, fmt.Errorf("%w: name required", ErrInvalidInput)
	}

	if req.Customer == "" {
		return nil, fmt.Errorf("%w: customer required", ErrInvalidInput)
	}

	if req.ObjectAddress == "" {
		return nil, fmt.Errorf("%w: address required", ErrInvalidInput)
	}

	if err := s.checkPermission(ctx, "create", ""); err != nil {
		return nil, err
	}

	if req.StartDate == nil {
		return nil, fmt.Errorf("%w: start date required", ErrInvalidInput)
	}
	if req.EndDate != nil {
		if req.EndDate.AsTime().Before(req.StartDate.AsTime()) {
			return nil, errors.New("end date cannot be before start date")
		}
	}

	project := &projectv1.Project{
		Name:          req.Name,
		Description:   req.Description,
		ObjectAddress: req.ObjectAddress,
		Customer:      req.Customer,
		StartDate:     req.StartDate,
		EndDate:       req.EndDate,
		Status:        projectv1.ProjectStatus_PROJECT_STATUS_ACTIVE,
	}

	if err := s.projectRepo.Create(ctx, project); err != nil {
		return nil, fmt.Errorf("failed to create project in service: %w", err)
	}
	userID, err := util.GetFromContext(ctx, "user_id")
	if err != nil {
		log.Printf("failed to get user_id: %v", err)
		return project, nil
	}

	err = s.memberRepo.Add(ctx, &projectv1.ProjectMember{
		ProjectId: project.Id,
		UserId:    userID,
		JoinedAt:  timestamppb.Now(),
	})
	if err != nil {
		log.Printf("failed to add creator as member: %v", err)
	}

	return project, nil
}

func (s *Service) GetProject(ctx context.Context, req *projectv1.GetProjectRequest) (*projectv1.Project, error) {

	if req.Id == "" {
		return nil, fmt.Errorf("%w: project id required", ErrInvalidInput)
	}

	if err := s.checkPermission(ctx, "get", req.Id); err != nil {
		return nil, ErrNoPermission
	}

	project, err := s.projectRepo.FindByID(ctx, req.Id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrProjectNotFound
		}
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	return project, nil
}

func (s *Service) UpdateProject(ctx context.Context, req *projectv1.UpdateProjectRequest) (*projectv1.Project, error) {

	if req.Id == "" {
		return nil, fmt.Errorf("%w: project id required", ErrInvalidInput)
	}

	if err := s.checkPermission(ctx, "update", req.Id); err != nil {
		return nil, ErrNoPermission
	}

	existing, err := s.projectRepo.FindByID(ctx, req.Id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrProjectNotFound
		}
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	if req.EndDate != nil {
		if req.EndDate.AsTime().Before(existing.StartDate.AsTime()) {
			return nil, ErrInvalidInput
		}
	}

	updatedProject := &projectv1.Project{
		Id:            existing.Id,
		Name:          util.NonEmpty(req.Name, existing.Name),
		Description:   util.NonEmpty(req.Description, existing.Description),
		ObjectAddress: util.NonEmpty(req.ObjectAddress, existing.ObjectAddress),
		Customer:      util.NonEmpty(req.Customer, existing.Customer),
		StartDate:     existing.StartDate,
		EndDate:       util.FirstNonNil(req.EndDate, existing.EndDate),
		Status:        existing.Status,
		UpdatedAt:     timestamppb.Now(),
		CreatedBy:     existing.CreatedBy,
	}

	if err := s.projectRepo.Update(ctx, updatedProject); err != nil {
		return nil, err
	}

	return updatedProject, nil
}

func (s *Service) DeleteProject(ctx context.Context, req *projectv1.DeleteProjectRequest) (*emptypb.Empty, error) {

	if req.Id == "" {
		return nil, fmt.Errorf("%w: project id required", ErrInvalidInput)
	}

	if err := s.checkPermission(ctx, "delete", req.Id); err != nil {
		return nil, ErrNoPermission
	}

	if err := s.projectRepo.Delete(ctx, req.Id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &emptypb.Empty{}, nil
		}
		return nil, fmt.Errorf("failed to delete project: %w", err)
	}

	return &emptypb.Empty{}, nil
}

func (s *Service) ChangeProjectStatus(ctx context.Context, req *projectv1.ChangeProjectStatusRequest) (*projectv1.Project, error) {

	if req.Id == "" {
		return nil, fmt.Errorf("%w: project id required", ErrInvalidInput)
	}
	if req.Status == projectv1.ProjectStatus_PROJECT_STATUS_UNSPECIFIED {
		return nil, fmt.Errorf("%w: project status required", ErrInvalidInput)
	}

	if err := s.checkPermission(ctx, "change_status", req.Id); err != nil {
		return nil, ErrNoPermission
	}
	if err := s.projectRepo.UpdateStatus(ctx, req.Id, req.Status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrProjectNotFound
		}
		return nil, fmt.Errorf("failed to update project status: %w", err)
	}

	return s.projectRepo.FindByID(ctx, req.Id)
}

func (s *Service) checkPermission(ctx context.Context, action string, projectID string) error {
	userID, err := util.GetFromContext(ctx, "user_id")
	if err != nil {
		return err
	}

	switch action {
	case "create":
		if !s.permissions.CanCreateProject(ctx, userID) {
			return ErrNoPermission
		}

	case "get":
		if !s.permissions.CanGetProject(ctx, userID, projectID) {
			return ErrNoPermission
		}

	case "update":
		if !s.permissions.CanUpdateProject(ctx, userID, projectID) {
			return ErrNoPermission
		}

	case "delete":
		if !s.permissions.CanDeleteProject(ctx, userID) {
			return ErrNoPermission
		}

	case "change_status":
		if !s.permissions.CanChangeStatus(ctx, userID, projectID) {
			return ErrNoPermission
		}
	}

	return nil
}
