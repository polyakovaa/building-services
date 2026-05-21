package department

import (
	"context"
	"fmt"
	"log"
	"time"

	projectv1 "building-services/gen/project/v1"
	"building-services/project-service/internal/authz"
	"building-services/project-service/internal/errs"
	"building-services/project-service/internal/events"
	"building-services/project-service/internal/user"
	"building-services/project-service/internal/util"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Service struct {
	depRepo  DepRepo
	userRepo UserRepo
	authz    PermissionChecker
	events   events.Publisher
}

type UserRepo interface {
	FindByID(ctx context.Context, id string) (*user.User, error)
}
type PermissionChecker interface {
	Check(ctx context.Context, userID string, resourceType string, resourceID string, action string) (bool, error)
}

type DepRepo interface {
	Create(ctx context.Context, dept *projectv1.Department) error
	FindByID(ctx context.Context, id string) (*projectv1.Department, error)
	List(ctx context.Context) ([]*projectv1.Department, error)
	Update(ctx context.Context, dept *projectv1.Department) error
	Delete(ctx context.Context, id string) error
	AssignUser(ctx context.Context, userID, departmentID string) error
	RemoveUser(ctx context.Context, userID string) error
	GetDepartmentUsers(ctx context.Context, departmentID string) ([]*projectv1.User, error)
}

func NewService(repo DepRepo, userRepo UserRepo, permissionChecker PermissionChecker, eventPublisher events.Publisher) *Service {
	return &Service{
		depRepo:  repo,
		userRepo: userRepo,
		authz:    permissionChecker,
		events:   eventPublisher,
	}
}

func (s *Service) CreateDepartment(ctx context.Context, req *projectv1.CreateDepartmentRequest) (*projectv1.Department, error) {
	currentUserID, err := util.GetFromContext(ctx, "user_id")
	if err != nil {
		return nil, fmt.Errorf("failed to get user_id: %w", err)
	}

	ok, err := s.authz.Check(ctx, currentUserID, authz.ResourceDepartment, "", authz.ActionCreate)
	if err != nil || !ok {
		return nil, errs.ErrNoPermission
	}

	if req.Name == "" {
		return nil, fmt.Errorf("department name required: %w", errs.ErrInvalidInput)
	}

	dept := &projectv1.Department{
		Id:   uuid.New().String(),
		Name: req.Name,
	}

	if req.HeadUserId != "" {
		_, err := s.userRepo.FindByID(ctx, req.HeadUserId)
		if err != nil {
			return nil, fmt.Errorf("head user not found: %w", err)
		}
		dept.HeadUserId = req.HeadUserId
	}

	if err := s.depRepo.Create(ctx, dept); err != nil {
		return nil, err
	}

	if s.events != nil {
		event := map[string]interface{}{
			"event_type":    "department.created",
			"occurred_at":   time.Now().UTC().Format(time.RFC3339Nano),
			"actor_user_id": currentUserID,
			"department_id": dept.Id,
			"id":            dept.Id,
			"name":          dept.Name,
		}
		if err := s.events.Publish(ctx, "department.created", event); err != nil {
			log.Printf("Failed to publish department.created: %v", err)
		}
	}

	return dept, nil
}

func (s *Service) GetDepartment(ctx context.Context, req *projectv1.GetDepartmentRequest) (*projectv1.Department, error) {
	currentUserID, err := util.GetFromContext(ctx, "user_id")
	if err != nil {
		return nil, fmt.Errorf("failed to get user_id: %w", err)
	}

	if req.Id == "" {
		return nil, fmt.Errorf("department id required: %w", errs.ErrInvalidInput)
	}

	ok, err := s.authz.Check(ctx, currentUserID, authz.ResourceDepartment, req.Id, authz.ActionView)
	if err != nil || !ok {
		return nil, errs.ErrNoPermission
	}

	return s.depRepo.FindByID(ctx, req.Id)
}

func (s *Service) ListDepartments(ctx context.Context, req *projectv1.ListDepartmentsRequest) (*projectv1.ListDepartmentsResponse, error) {
	currentUserID, err := util.GetFromContext(ctx, "user_id")
	if err != nil {
		return nil, fmt.Errorf("failed to get user_id: %w", err)
	}

	user, err := s.userRepo.FindByID(ctx, currentUserID)
	if err != nil {
		return nil, err
	}

	var departments []*projectv1.Department
	if user.Role == authz.RoleDepartmentManager {
		allDepts, err := s.depRepo.List(ctx)
		if err != nil {
			return nil, err
		}
		for _, dept := range allDepts {
			if dept.HeadUserId == currentUserID || dept.HeadUserId == "" {
				departments = append(departments, dept)
			}
		}
	} else {
		departments, err = s.depRepo.List(ctx)
		if err != nil {
			return nil, err
		}
	}

	return &projectv1.ListDepartmentsResponse{
		Departments: departments,
		TotalCount:  int32(len(departments)),
	}, nil
}

func (s *Service) UpdateDepartment(ctx context.Context, req *projectv1.UpdateDepartmentRequest) (*projectv1.Department, error) {
	currentUserID, err := util.GetFromContext(ctx, "user_id")
	if err != nil {
		return nil, fmt.Errorf("failed to get user_id: %w", err)
	}

	if req.Id == "" {
		return nil, fmt.Errorf("department id required: %w", errs.ErrInvalidInput)
	}

	ok, err := s.authz.Check(ctx, currentUserID, authz.ResourceDepartment, req.Id, authz.ActionEdit)
	if err != nil || !ok {
		return nil, errs.ErrNoPermission
	}

	dept, err := s.depRepo.FindByID(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	if req.Name != "" {
		dept.Name = req.Name
	}

	if req.HeadUserId != "" {
		_, err := s.userRepo.FindByID(ctx, req.HeadUserId)
		if err != nil {
			return nil, fmt.Errorf("head user not found: %w", err)
		}
		dept.HeadUserId = req.HeadUserId
	}

	if err := s.depRepo.Update(ctx, dept); err != nil {
		return nil, err
	}

	return dept, nil
}

func (s *Service) DeleteDepartment(ctx context.Context, req *projectv1.DeleteDepartmentRequest) (*emptypb.Empty, error) {
	currentUserID, err := util.GetFromContext(ctx, "user_id")
	if err != nil {
		return nil, fmt.Errorf("failed to get user_id: %w", err)
	}

	if req.Id == "" {
		return nil, fmt.Errorf("department id required: %w", errs.ErrInvalidInput)
	}

	ok, err := s.authz.Check(ctx, currentUserID, authz.ResourceDepartment, req.Id, authz.ActionDelete)
	if err != nil || !ok {
		return nil, errs.ErrNoPermission
	}

	if err := s.depRepo.Delete(ctx, req.Id); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *Service) AssignUserToDepartment(ctx context.Context, req *projectv1.AssignUserToDepartmentRequest) (*emptypb.Empty, error) {
	currentUserID, err := util.GetFromContext(ctx, "user_id")
	if err != nil {
		return nil, fmt.Errorf("failed to get user_id: %w", err)
	}

	if req.UserId == "" {
		return nil, fmt.Errorf("user id required")
	}
	if req.DepartmentId == "" {
		return nil, fmt.Errorf("department id required: %w", errs.ErrInvalidInput)
	}
	ok, err := s.authz.Check(ctx, currentUserID, authz.ResourceDepartment, req.DepartmentId, authz.ActionAssignToDept)
	if err != nil || !ok {
		return nil, errs.ErrNoPermission
	}

	_, err = s.userRepo.FindByID(ctx, req.UserId)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}
	_, err = s.depRepo.FindByID(ctx, req.DepartmentId)
	if err != nil {
		return nil, fmt.Errorf("department not found: %w", err)
	}

	if err := s.depRepo.AssignUser(ctx, req.UserId, req.DepartmentId); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *Service) RemoveUserFromDepartment(ctx context.Context, req *projectv1.RemoveUserFromDepartmentRequest) (*emptypb.Empty, error) {
	currentUserID, err := util.GetFromContext(ctx, "user_id")
	if err != nil {
		return nil, fmt.Errorf("failed to get user_id: %w", err)
	}

	if req.UserId == "" {
		return nil, fmt.Errorf("user id required: %w", errs.ErrInvalidInput)
	}
	user, err := s.userRepo.FindByID(ctx, req.UserId)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}
	if user.DepartmentID == nil {
		return nil, fmt.Errorf("user not in any department")
	}

	ok, err := s.authz.Check(ctx, currentUserID, authz.ResourceDepartment, *user.DepartmentID, authz.ActionAssignToDept)
	if err != nil || !ok {
		return nil, errs.ErrNoPermission
	}

	if err := s.depRepo.RemoveUser(ctx, req.UserId); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *Service) GetDepartmentUsers(ctx context.Context, req *projectv1.GetDepartmentUsersRequest) (*projectv1.GetDepartmentUsersResponse, error) {
	currentUserID, err := util.GetFromContext(ctx, "user_id")
	if err != nil {
		return nil, fmt.Errorf("failed to get user_id: %w", err)
	}

	if req.DepartmentId == "" {
		return nil, fmt.Errorf("department id required: %w", errs.ErrInvalidInput)
	}

	ok, err := s.authz.Check(ctx, currentUserID, authz.ResourceDepartment, req.DepartmentId, authz.ActionView)
	if err != nil || !ok {
		return nil, errs.ErrNoPermission
	}

	users, err := s.depRepo.GetDepartmentUsers(ctx, req.DepartmentId)
	if err != nil {
		return nil, err
	}

	return &projectv1.GetDepartmentUsersResponse{
		Users: users,
	}, nil
}
