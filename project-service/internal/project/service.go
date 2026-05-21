package project

import (
	projectv1 "building-services/gen/project/v1"
	"building-services/project-service/internal/authz"
	"building-services/project-service/internal/events"
	"building-services/project-service/internal/user"

	"building-services/project-service/internal/errs"
	"building-services/project-service/internal/util"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Service struct {
	projectRepo  ProjectRepo
	memberRepo   MemberRepo
	userRepo     UserRepo
	timelineRepo TimelineRepo
	authz        PermissionChecker
	events       events.Publisher
}

func NewService(projectRepo ProjectRepo,
	memberRepo MemberRepo, userRepo UserRepo,
	timelineRepo TimelineRepo,
	authz PermissionChecker, eventPublisher events.Publisher) *Service {
	return &Service{
		projectRepo:  projectRepo,
		timelineRepo: timelineRepo,
		memberRepo:   memberRepo,
		userRepo:     userRepo,
		authz:        authz,
		events:       eventPublisher,
	}
}

type TimelineRepo interface {
	CreateEmpty(ctx context.Context, projectID string) error
}

type ProjectRepo interface {
	Create(ctx context.Context, project *projectv1.Project) error
	FindByID(ctx context.Context, id string) (*projectv1.Project, error)
	Update(ctx context.Context, project *projectv1.Project) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, filter *ProjectFilter) ([]*projectv1.Project, error)
	UpdateStatus(ctx context.Context, id string, status projectv1.ProjectStatus) error
}
type UserRepo interface {
	FindByID(ctx context.Context, id string) (*user.User, error)
	FindByEmail(ctx context.Context, email string) (*user.User, error)
	Find(ctx context.Context, query string, limit int) ([]*user.User, error)
}

type MemberRepo interface {
	Add(ctx context.Context, member *projectv1.ProjectMember) error
}

type PermissionChecker interface {
	Check(ctx context.Context, userID string, resourceType string, resourceID string, action string) (bool, error)
}

func (s *Service) CreateProject(ctx context.Context, req *projectv1.CreateProjectRequest) (*projectv1.Project, error) {
	userID, err := util.GetFromContext(ctx, "user_id")
	if err != nil {
		return nil, fmt.Errorf("failed to get user_id: %w", err)
	}

	ok, err := s.authz.Check(ctx, userID, authz.ResourceProject, "", authz.ActionCreate)
	if err != nil || !ok {
		return nil, errs.ErrNoPermission
	}

	if req.Name == "" {
		return nil, fmt.Errorf("%w: name required", errs.ErrInvalidInput)
	}

	if req.Customer == "" {
		return nil, fmt.Errorf("%w: customer required", errs.ErrInvalidInput)
	}

	if req.ObjectAddress == "" {
		return nil, fmt.Errorf("%w: address required", errs.ErrInvalidInput)
	}

	if req.StartDate == nil {
		return nil, fmt.Errorf("%w: start date required", errs.ErrInvalidInput)
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
		CreatedBy:     userID,
	}

	if err := s.projectRepo.Create(ctx, project); err != nil {
		return nil, fmt.Errorf("failed to create project in service: %w", err)
	}

	err = s.memberRepo.Add(ctx, &projectv1.ProjectMember{
		ProjectId: project.Id,
		UserId:    userID,
		JoinedAt:  timestamppb.Now(),
	})
	if err != nil {
		log.Printf("failed to add creator as member: %v", err)
	}
	if err := s.timelineRepo.CreateEmpty(ctx, project.Id); err != nil {
		log.Printf("failed to create timeline: %v", err)
	}

	if s.events != nil {
		event := map[string]interface{}{
			"event_type":     "project.created",
			"occurred_at":    time.Now().UTC().Format(time.RFC3339Nano),
			"actor_user_id":  userID,
			"project_id":     project.Id,
			"project_name":   project.Name,
			"description":    project.Description,
			"object_address": project.ObjectAddress,
			"customer":       project.Customer,
			"start_date":     tsToFormat(project.StartDate),
			"end_date":       tsToFormat(project.EndDate),
			"status":         int32(project.Status),
		}
		if err := s.events.Publish(ctx, "project.created", event); err != nil {
			log.Printf("Failed to publish project.created: %v", err)
		}
	}

	return project, nil
}

func tsToFormat(ts *timestamppb.Timestamp) string {
	if ts == nil {
		return ""
	}
	return ts.AsTime().UTC().Format(time.RFC3339Nano)
}

func (s *Service) GetProject(ctx context.Context, req *projectv1.GetProjectRequest) (*projectv1.Project, error) {
	userID, err := util.GetFromContext(ctx, "user_id")
	if err != nil {
		return nil, fmt.Errorf("failed to get user_id: %w", err)
	}

	if req.Id == "" {
		return nil, fmt.Errorf("%w: project id required", errs.ErrInvalidInput)
	}

	ok, err := s.authz.Check(ctx, userID, authz.ResourceProject, req.Id, authz.ActionView)
	if err != nil || !ok {
		return nil, errs.ErrNoPermission
	}

	project, err := s.projectRepo.FindByID(ctx, req.Id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errs.ErrProjectNotFound
		}
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	return project, nil
}

func (s *Service) UpdateProject(ctx context.Context, req *projectv1.UpdateProjectRequest) (*projectv1.Project, error) {
	userID, err := util.GetFromContext(ctx, "user_id")
	if err != nil {
		return nil, fmt.Errorf("failed to get user_id: %w", err)
	}
	ok, err := s.authz.Check(ctx, userID, authz.ResourceProject, req.Id, authz.ActionEdit)
	if err != nil || !ok {
		return nil, errs.ErrNoPermission
	}

	if req.Id == "" {
		return nil, fmt.Errorf("%w: project id required", errs.ErrInvalidInput)
	}

	existing, err := s.projectRepo.FindByID(ctx, req.Id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errs.ErrProjectNotFound
		}
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	if req.EndDate != nil {
		if req.EndDate.AsTime().Before(existing.StartDate.AsTime()) {
			return nil, errs.ErrInvalidInput
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

	if s.events != nil {
		event := map[string]interface{}{
			"event_type":     "project.updated",
			"occurred_at":    time.Now().UTC().Format(time.RFC3339Nano),
			"actor_user_id":  userID,
			"project_id":     updatedProject.Id,
			"project_name":   updatedProject.Name,
			"description":    updatedProject.Description,
			"object_address": updatedProject.ObjectAddress,
			"customer":       updatedProject.Customer,
			"start_date":     tsToFormat(updatedProject.StartDate),
			"end_date":       tsToFormat(updatedProject.EndDate),
			"status":         int32(updatedProject.Status),
		}
		if err := s.events.Publish(ctx, "project.updated", event); err != nil {
			log.Printf("Failed to publish project.updated: %v", err)
		}
	}

	return updatedProject, nil
}

func (s *Service) DeleteProject(ctx context.Context, req *projectv1.DeleteProjectRequest) (*emptypb.Empty, error) {
	userID, err := util.GetFromContext(ctx, "user_id")
	if err != nil {
		return nil, fmt.Errorf("failed to get user_id: %w", err)
	}
	ok, err := s.authz.Check(ctx, userID, authz.ResourceProject, req.Id, authz.ActionDelete)
	if err != nil || !ok {
		return nil, errs.ErrNoPermission
	}

	if req.Id == "" {
		return nil, fmt.Errorf("%w: project id required", errs.ErrInvalidInput)
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
	userID, err := util.GetFromContext(ctx, "user_id")
	if err != nil {
		return nil, fmt.Errorf("failed to get user_id: %w", err)
	}
	ok, err := s.authz.Check(ctx, userID, authz.ResourceProject, req.Id, authz.ActionChangeStatus)
	if err != nil || !ok {
		return nil, errs.ErrNoPermission
	}

	if req.Id == "" {
		return nil, fmt.Errorf("%w: project id required", errs.ErrInvalidInput)
	}
	if req.Status == projectv1.ProjectStatus_PROJECT_STATUS_UNSPECIFIED {
		return nil, fmt.Errorf("%w: project status required", errs.ErrInvalidInput)
	}

	existingProject, err := s.projectRepo.FindByID(ctx, req.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing project: %w", err)
	}

	if err := s.projectRepo.UpdateStatus(ctx, req.Id, req.Status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errs.ErrProjectNotFound
		}
		return nil, fmt.Errorf("failed to update project status: %w", err)
	}

	updatedProject, err := s.projectRepo.FindByID(ctx, req.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated project: %w", err)
	}

	if s.events != nil {
		event := map[string]interface{}{
			"event_type":    "project.status_changed",
			"occurred_at":   time.Now().UTC().Format(time.RFC3339Nano),
			"actor_user_id": userID,
			"project_id":    req.Id,
			"project_name":  updatedProject.Name,
			"from_status":   int32(existingProject.Status),
			"to_status":     int32(req.Status),
		}
		if err := s.events.Publish(ctx, "project.status_changed", event); err != nil {
			log.Printf("Failed to publish project.status_changed: %v", err)
		}
	}

	return updatedProject, nil
}

func (s *Service) ListProjects(ctx context.Context, req *projectv1.ListProjectsRequest) (*projectv1.ListProjectsResponse, error) {
	userID, err := util.GetFromContext(ctx, "user_id")
	if err != nil {
		return nil, fmt.Errorf("failed to get user_id: %w", err)
	}

	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	userRole := user.Role
	if roleFromContext, err := util.GetFromContext(ctx, "user_role"); err == nil && roleFromContext != "" {
		userRole = roleFromContext
	}

	filter := &ProjectFilter{
		Status:    req.StatusFilter,
		ManagerID: req.ManagerId,
		UserID:    userID,
		UserRole:  userRole,
	}
	if user != nil && user.DepartmentID != nil {
		filter.DepartmentID = user.DepartmentID
	}

	projects, err := s.projectRepo.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}

	return &projectv1.ListProjectsResponse{
		Projects:   projects,
		TotalCount: int32(len(projects)),
	}, nil

}

func (s *Service) GetUser(ctx context.Context, req *projectv1.GetUserRequest) (*projectv1.User, error) {
	if req.Id == "" {
		return nil, fmt.Errorf("%w: user id required", errs.ErrInvalidInput)
	}

	user, err := s.userRepo.FindByID(ctx, req.Id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errs.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	return &projectv1.User{
		Id:           user.ID,
		FullName:     user.FullName,
		Email:        user.Email,
		Role:         user.Role,
		DepartmentId: stringValue(user.DepartmentID),
	}, nil
}

func (s *Service) GetUserByEmail(ctx context.Context, req *projectv1.GetUserByEmailRequest) (*projectv1.User, error) {
	if req.Email == "" {
		return nil, fmt.Errorf("%w: email required", errs.ErrInvalidInput)
	}

	user, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errs.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	return &projectv1.User{
		Id:           user.ID,
		FullName:     user.FullName,
		Email:        user.Email,
		Role:         user.Role,
		DepartmentId: stringValue(user.DepartmentID),
	}, nil
}

func (s *Service) FindUsers(ctx context.Context, req *projectv1.FindUsersRequest) (*projectv1.FindUsersResponse, error) {
	q := strings.TrimSpace(req.Query)
	limit := int(req.Limit)
	if limit <= 0 {
		limit = 100
	}
	if q != "" && len(q) < 2 {
		return &projectv1.FindUsersResponse{}, nil
	}
	users, err := s.userRepo.Find(ctx, q, limit)
	if err != nil {
		return nil, fmt.Errorf("find users: %w", err)
	}
	out := make([]*projectv1.User, 0, len(users))
	for _, u := range users {
		out = append(out, &projectv1.User{
			Id:           u.ID,
			FullName:     u.FullName,
			Email:        u.Email,
			Role:         u.Role,
			DepartmentId: stringValue(u.DepartmentID),
		})
	}
	return &projectv1.FindUsersResponse{Users: out}, nil
}

func stringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
