package authz

import (
	projectv1 "building-services/gen/project/v1"
	"building-services/project-service/internal/user"
	"context"
	"log"
)

type PermissionChecker struct {
	userRepo   UserRepo
	memberRepo MemberRepo
}

func NewPermissionChecker(userRepo UserRepo, memberRepo MemberRepo) *PermissionChecker {
	return &PermissionChecker{
		userRepo:   userRepo,
		memberRepo: memberRepo,
	}
}

type UserRepo interface {
	FindByID(ctx context.Context, id string) (*user.User, error)
}

type MemberRepo interface {
	IsProjectMember(ctx context.Context, projectID, userID string) (*projectv1.ProjectMember, error)
	IsProjectInDepartment(ctx context.Context, projectID, departmentID string) (bool, error)
	IsManagerOfProject(ctx context.Context, userID, projectID string) (bool, error)
	GetProjectMembers(ctx context.Context, projectID string) ([]*projectv1.ProjectMember, error)
}

const (
	RoleDirector          = "ROLE_DIRECTOR"
	RoleGIP               = "ROLE_GIP"
	RoleDepartmentManager = "ROLE_DEPARTMENT_MANAGER"
	RoleProjectManager    = "ROLE_PROJECT_MANAGER"
	RoleWorker            = "ROLE_WORKER"
	RoleUnspecified       = "ROLE_UNSPECIFIED"
)

func (p *PermissionChecker) CanCreateProject(ctx context.Context, userID string) bool {
	user, err := p.userRepo.FindByID(ctx, userID)
	if err != nil {
		log.Printf("failed to find user %s: %v", userID, err)
		return false
	}

	allowedRoles := map[string]bool{
		RoleDirector:          true,
		RoleGIP:               true,
		RoleDepartmentManager: true,
		RoleProjectManager:    true,
		RoleWorker:            false,
		RoleUnspecified:       false,
	}
	return allowedRoles[user.Role]
}

func (p *PermissionChecker) CanGetProject(ctx context.Context, userID, projectID string) bool {
	user, err := p.userRepo.FindByID(ctx, userID)
	if err != nil {
		log.Printf("failed to find user %s: %v", userID, err)
		return false
	}

	if user.Role == RoleDirector || user.Role == RoleGIP {
		return true
	}

	if user.Role == RoleDepartmentManager {
		ok, err := p.memberRepo.IsProjectInDepartment(ctx, projectID, *user.DepartmentID)
		if err != nil {
			return false
		}
		return ok
	}

	if user.Role == RoleProjectManager {
		ok, err := p.memberRepo.IsManagerOfProject(ctx, userID, projectID)
		if err != nil {
			return false
		}
		return ok
	}
	member, err := p.memberRepo.IsProjectMember(ctx, projectID, userID)
	if err != nil {
		return false
	}

	if member != nil {
		return true
	}
	return false
}

func (p *PermissionChecker) CanUpdateProject(ctx context.Context, userID, projectID string) bool {
	user, err := p.userRepo.FindByID(ctx, userID)
	if err != nil {
		log.Printf("failed to find user %s: %v", userID, err)
		return false
	}

	if user.Role == RoleDirector || user.Role == RoleGIP {
		return true
	}

	if user.Role == RoleDepartmentManager {
		ok, err := p.memberRepo.IsProjectInDepartment(ctx, projectID, *user.DepartmentID)
		if err != nil {
			return false
		}
		return ok
	}

	if user.Role == RoleProjectManager {
		ok, err := p.memberRepo.IsManagerOfProject(ctx, userID, projectID)
		if err != nil {
			return false
		}
		return ok
	}

	return false
}

func (p *PermissionChecker) CanDeleteProject(ctx context.Context, userID string) bool {
	user, err := p.userRepo.FindByID(ctx, userID)
	if err != nil {
		log.Printf("failed to find user %s: %v", userID, err)
		return false
	}
	return user.Role == RoleDirector
}

func (p *PermissionChecker) CanChangeStatus(ctx context.Context, userID, projectID string) bool {
	user, err := p.userRepo.FindByID(ctx, userID)
	if err != nil {
		log.Printf("failed to find user %s: %v", userID, err)
		return false
	}

	// Директор и ГИП могут всё
	if user.Role == RoleDirector || user.Role == RoleGIP {
		return true
	}

	// Руководитель отдела - только проекты своего отдела
	// Проектный менеджер - только свои проекты

	if user.Role == RoleDepartmentManager {
		ok, err := p.memberRepo.IsProjectInDepartment(ctx, projectID, *user.DepartmentID)
		if err != nil {
			return false
		}
		return ok
	}

	if user.Role == RoleProjectManager {
		ok, err := p.memberRepo.IsManagerOfProject(ctx, userID, projectID)
		if err != nil {
			return false
		}
		return ok
	}

	return false
}
