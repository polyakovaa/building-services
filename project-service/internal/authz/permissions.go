package authz

import (
	projectv1 "building-services/gen/project/v1"
	"building-services/project-service/internal/user"
	"context"
	"database/sql"
	"errors"
	"log"
)

type PermissionChecker struct {
	userRepo       UserRepo
	memberRepo     MemberRepo
	taskRepo       TaskRepo
	attachmentRepo AttachmentRepo
	departmentRepo DepartmentRepo
}

func NewPermissionChecker(userRepo UserRepo, memberRepo MemberRepo,
	taskRepo TaskRepo, attachmentRepo AttachmentRepo,
	departmentRepo DepartmentRepo) *PermissionChecker {
	return &PermissionChecker{
		userRepo:       userRepo,
		memberRepo:     memberRepo,
		taskRepo:       taskRepo,
		attachmentRepo: attachmentRepo,
		departmentRepo: departmentRepo,
	}
}

type UserRepo interface {
	FindByID(ctx context.Context, id string) (*user.User, error)
}
type DepartmentRepo interface {
	FindByID(ctx context.Context, id string) (*projectv1.Department, error)
}

type TaskRepo interface {
	FindByID(ctx context.Context, id string) (*projectv1.Task, error)
	GetProjectID(ctx context.Context, id string) (string, error)
	IsAssignee(ctx context.Context, taskID string, userID string) (bool, error)
}

type MemberRepo interface {
	IsProjectMember(ctx context.Context, projectID, userID string) (*projectv1.ProjectMember, error)
	IsProjectInDepartment(ctx context.Context, projectID, departmentID string) (bool, error)
	IsManagerOfProject(ctx context.Context, userID, projectID string) (bool, error)
	GetProjectMembers(ctx context.Context, projectID string) ([]*projectv1.ProjectMember, error)
}
type AttachmentRepo interface {
	GetTaskID(ctx context.Context, attachmentID string) (string, error)
	GetUploadedBy(ctx context.Context, attachmentID string) (string, error)
}

const (
	ResourceProject    = "project"
	ResourceTask       = "task"
	ResourceDepartment  = "department"
	ResourceActivityType = "activity_type"
	ResourceUser        = "user"
	ResourceAttachment  = "attachment"
)

const (
	ActionCreate       = "create"
	ActionView         = "view"
	ActionEdit         = "edit"
	ActionDelete       = "delete"
	ActionChangeStatus = "change_status"
	ActionAssign       = "assign"
	ActionUpdateLabor  = "update_labor"
	ActionUpload       = "upload"
	ActionDownload     = "download"
	ActionAssignToDept = "assign_to_dept"
)

const (
	RoleDirector          = "ROLE_DIRECTOR"
	RoleGIP               = "ROLE_GIP"
	RoleDepartmentManager = "ROLE_DEPARTMENT_MANAGER"
	RoleProjectManager    = "ROLE_PROJECT_MANAGER"
	RoleWorker            = "ROLE_WORKER"
	RoleUnspecified       = "ROLE_UNSPECIFIED"
	RoleAdmin             = "ROLE_ADMIN"
)

func (c *PermissionChecker) Check(ctx context.Context, userID string, resourceType string, resourceID string, action string) (bool, error) {
	log.Printf("[DEBUG] Check: userID=%s, resourceType=%s, resourceID=%s, action=%s", userID, resourceType, resourceID, action)

	user, err := c.userRepo.FindByID(ctx, userID)
	if err != nil {
		log.Printf("[DEBUG] Check: failed to find user %s: %v", userID, err)
		return false, err
	}
	log.Printf("[DEBUG] Check: user %s has role %s", userID, user.Role)

	if user.Role == RoleAdmin || user.Role == RoleDirector {
		log.Printf("[DEBUG] Check: %s has admin/director role - access granted", userID)
		return true, nil
	}

	switch resourceType {
	case ResourceProject:
		if resourceID == "" && action == ActionView {
			return true, nil
		}
		result, err := c.checkProject(ctx, user, resourceID, action)
		log.Printf("[DEBUG] Check: project permission result for user %s: %v, err: %v", userID, result, err)
		return result, err
	case ResourceTask:
		result, err := c.checkTask(ctx, user, resourceID, action)
		log.Printf("[DEBUG] Check: task permission result for user %s: %v, err: %v", userID, result, err)
		return result, err
	case ResourceAttachment:
		return c.checkAttachment(ctx, user, resourceID, action)
	case ResourceDepartment:
		return c.checkDepartment(ctx, user, resourceID, action)
	case ResourceActivityType:
		return c.checkActivityType(ctx, user, action)
	}

	log.Printf("[DEBUG] Check: unsupported resource type %s - access denied", resourceType)
	return false, nil
}

func (c *PermissionChecker) checkProject(ctx context.Context, user *user.User, projectID string, action string) (bool, error) {
	switch action {
	case ActionCreate:
		return user.Role == RoleProjectManager ||
			user.Role == RoleGIP, nil

	case ActionView:
		if user.Role == RoleDepartmentManager {
			if user.DepartmentID == nil {
				return false, nil
			}
			return c.memberRepo.IsProjectInDepartment(ctx, projectID, *user.DepartmentID)
		}
		_, err := c.memberRepo.IsProjectMember(ctx, projectID, user.ID)
		if err != nil {
			return false, nil
		}
		return true, nil

	case ActionEdit, ActionChangeStatus:
		if user.Role == RoleGIP {
			member, err := c.memberRepo.IsProjectMember(ctx, projectID, user.ID)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return false, nil
				}
				return false, err
			}
			return member != nil, nil
		}
		if user.Role == RoleDepartmentManager {
			if user.DepartmentID == nil {
				return false, nil
			}
			return c.memberRepo.IsProjectInDepartment(ctx, projectID, *user.DepartmentID)
		}
		if user.Role == RoleProjectManager {
			return c.memberRepo.IsManagerOfProject(ctx, user.ID, projectID)
		}
		return false, nil

	case ActionDelete:
		return false, nil

	default:
		return false, nil
	}
}

func (c *PermissionChecker) checkTask(ctx context.Context, user *user.User, taskID string, action string) (bool, error) {
	log.Printf("[DEBUG] checkTask: user=%s, role=%s, taskID=%s, action=%s", user.ID, user.Role, taskID, action)

	switch action {
	case ActionCreate:
		projectID := taskID
		return c.checkProject(ctx, user, projectID, ActionEdit)

	case ActionView:
		projectID, err := c.taskRepo.GetProjectID(ctx, taskID)
		if err != nil {
			log.Printf("[DEBUG] checkTask: failed to get projectID for task %s: %v", taskID, err)
			return false, err
		}
		ok, err := c.checkProject(ctx, user, projectID, ActionView)
		if err != nil {
			return false, err
		}
		if ok {
			return true, nil
		}
		return c.taskRepo.IsAssignee(ctx, taskID, user.ID)

	case ActionEdit:
		projectID, err := c.taskRepo.GetProjectID(ctx, taskID)
		if err != nil {
			log.Printf("[DEBUG] checkTask: failed to get projectID for task %s: %v", taskID, err)
			return false, err
		}
		return c.checkProject(ctx, user, projectID, ActionEdit)

	case ActionDelete:
		projectID, err := c.taskRepo.GetProjectID(ctx, taskID)
		if err != nil {
			log.Printf("[DEBUG] checkTask: failed to get projectID for task %s: %v", taskID, err)
			return false, err
		}
		if user.Role == RoleAdmin || user.Role == RoleDirector || user.Role == RoleGIP {
			return true, nil
		}
		if user.Role == RoleDepartmentManager {
			return c.checkProject(ctx, user, projectID, ActionEdit)
		}
		if user.Role == RoleProjectManager {
			return c.checkProject(ctx, user, projectID, ActionEdit)
		}
		if user.Role == RoleWorker {
			task, err := c.taskRepo.FindByID(ctx, taskID)
			if err != nil {
				return false, err
			}
			return task.CreatedBy == user.ID, nil
		}
		return false, nil

	case ActionChangeStatus:
		projectID, err := c.taskRepo.GetProjectID(ctx, taskID)
		if err != nil {
			log.Printf("[DEBUG] checkTask: failed to get projectID for task %s: %v", taskID, err)
			return false, err
		}
		ok, err := c.checkProject(ctx, user, projectID, ActionEdit)
		if err != nil {
			return false, err
		}
		if ok {
			return true, nil
		}
		return c.taskRepo.IsAssignee(ctx, taskID, user.ID)

	case ActionUpdateLabor:
		projectID, err := c.taskRepo.GetProjectID(ctx, taskID)
		if err != nil {
			return false, err
		}
		ok, err := c.checkProject(ctx, user, projectID, ActionEdit)
		if err != nil {
			return false, err
		}
		if ok {
			return true, nil
		}
		return c.taskRepo.IsAssignee(ctx, taskID, user.ID)

	case ActionAssign:
		projectID, err := c.taskRepo.GetProjectID(ctx, taskID)
		if err != nil {
			log.Printf("[DEBUG] checkTask: failed to get projectID for task %s: %v", taskID, err)
			return false, err
		}
		return c.checkProject(ctx, user, projectID, ActionEdit)

	default:
		return false, nil
	}
}

func (c *PermissionChecker) checkAttachment(ctx context.Context, user *user.User, attachmentID string, action string) (bool, error) {
	taskID, err := c.attachmentRepo.GetTaskID(ctx, attachmentID)
	if err != nil {
		return false, err
	}

	switch action {
	case ActionUpload, ActionCreate:
		return c.checkTask(ctx, user, taskID, ActionEdit)

	case ActionView, ActionDownload:
		return c.checkTask(ctx, user, taskID, ActionView)

	case ActionDelete:
		uploadedBy, _ := c.attachmentRepo.GetUploadedBy(ctx, attachmentID)
		if user.ID == uploadedBy {
			return true, nil
		}
		return c.checkTask(ctx, user, taskID, ActionEdit)

	default:
		return false, nil
	}
}

func (c *PermissionChecker) checkActivityType(ctx context.Context, user *user.User, action string) (bool, error) {
	switch action {
	case ActionCreate, ActionEdit, ActionDelete:
		return user.Role == RoleDirector || user.Role == RoleGIP || user.Role == RoleAdmin, nil
	case ActionView:
		return true, nil
	default:
		return false, nil
	}
}

func (c *PermissionChecker) checkDepartment(ctx context.Context, user *user.User, departmentID string, action string) (bool, error) {
	switch action {
	case ActionCreate:
		return user.Role == RoleAdmin || user.Role == RoleDirector, nil

	case ActionView:
		if user.Role == RoleAdmin || user.Role == RoleDirector {
			return true, nil
		}
		if user.Role == RoleDepartmentManager {
			dept, err := c.departmentRepo.FindByID(ctx, departmentID)
			if err != nil {
				return false, err
			}
			return dept.HeadUserId == user.ID, nil
		}
		if user.Role == RoleProjectManager {
			return true, nil
		}
		return false, nil

	case ActionEdit:
		if user.Role == RoleAdmin || user.Role == RoleDirector {
			return true, nil
		}
		if user.Role == RoleDepartmentManager {
			dept, err := c.departmentRepo.FindByID(ctx, departmentID)
			if err != nil {
				return false, err
			}
			return dept.HeadUserId == user.ID, nil
		}
		return false, nil

	case ActionDelete:
		return user.Role == RoleAdmin, nil

	case ActionAssignToDept:
		if user.Role == RoleAdmin || user.Role == RoleDirector || user.Role == RoleProjectManager {
			return true, nil
		}
		if user.Role == RoleDepartmentManager {
			dept, err := c.departmentRepo.FindByID(ctx, departmentID)
			if err != nil {
				return false, err
			}
			return dept.HeadUserId == user.ID, nil
		}
		return false, nil

	default:
		return false, nil
	}
}
