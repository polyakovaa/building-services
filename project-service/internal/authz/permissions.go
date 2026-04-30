package authz

import (
	projectv1 "building-services/gen/project/v1"
	"building-services/project-service/internal/user"
	"context"
)

type PermissionChecker struct {
	userRepo       UserRepo
	memberRepo     MemberRepo
	taskRepo       TaskRepo
	attachmentRepo AttachmentRepo
}

func NewPermissionChecker(userRepo UserRepo, memberRepo MemberRepo, taskRepo TaskRepo, attachmentRepo AttachmentRepo) *PermissionChecker {
	return &PermissionChecker{
		userRepo:       userRepo,
		memberRepo:     memberRepo,
		taskRepo:       taskRepo,
		attachmentRepo: attachmentRepo,
	}
}

type UserRepo interface {
	FindByID(ctx context.Context, id string) (*user.User, error)
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
	ResourceAttachment = "attachment"
)

const (
	ActionCreate       = "create"
	ActionView         = "view"
	ActionEdit         = "edit"
	ActionDelete       = "delete"
	ActionChangeStatus = "change_status"
	ActionAssign       = "assign"
	ActionUpload       = "upload"
	ActionDownload     = "download"
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
	user, err := c.userRepo.FindByID(ctx, userID)
	if err != nil {
		return false, err
	}

	if user.Role == RoleAdmin {
		return true, nil
	}

	if user.Role == RoleDirector || user.Role == RoleGIP {
		return true, nil
	}

	switch resourceType {
	case ResourceProject:
		if resourceID == "" && action == ActionView {
			return true, nil
		}
		return c.checkProject(ctx, user, resourceID, action)
	case ResourceTask:
		return c.checkTask(ctx, user, resourceID, action)
	case ResourceAttachment:
		return c.checkAttachment(ctx, user, resourceID, action)
	}

	return false, nil
}

func (c *PermissionChecker) checkProject(ctx context.Context, user *user.User, projectID string, action string) (bool, error) {
	switch action {
	case ActionCreate:
		return user.Role == RoleProjectManager ||
			user.Role == RoleDepartmentManager, nil

	case ActionView:
		if user.Role == RoleDepartmentManager {
			if user.DepartmentID == nil {
				return false, nil
			}
			return c.memberRepo.IsProjectInDepartment(ctx, projectID, *user.DepartmentID)
		}
		_, err := c.memberRepo.IsProjectMember(ctx, projectID, user.ID)
		if err != nil {
			return false, err
		}
		return true, nil

	case ActionEdit, ActionChangeStatus:
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
	projectID, err := c.taskRepo.GetProjectID(ctx, taskID)
	if err != nil {
		return false, err
	}

	switch action {
	case ActionCreate:
		// Создавать задачу могут те же, кто может редактировать проект
		return c.checkProject(ctx, user, projectID, ActionEdit)

	case ActionView:
		// Если может видеть проект — видит и задачи
		ok, err := c.checkProject(ctx, user, projectID, ActionView)
		if err != nil {
			return false, err
		}
		if ok {
			return true, nil
		}
		// Или если исполнитель задачи
		return c.taskRepo.IsAssignee(ctx, taskID, user.ID)

	case ActionEdit:
		return c.checkProject(ctx, user, projectID, ActionEdit)

	case ActionDelete:
		return false, nil // только директор

	case ActionChangeStatus:
		// Менеджеры могут менять статус любых задач в проекте
		ok, err := c.checkProject(ctx, user, projectID, ActionEdit)
		if err != nil {
			return false, err
		}
		if ok {
			return true, nil
		}
		// Worker может менять статус только своих задач
		return c.taskRepo.IsAssignee(ctx, taskID, user.ID)

	case ActionAssign:
		return c.checkProject(ctx, user, projectID, ActionEdit)

	default:
		return false, nil
	}
}

// authz/permissions.go
func (c *PermissionChecker) checkAttachment(ctx context.Context, user *user.User, attachmentID string, action string) (bool, error) {
	// Получаем task_id из вложения
	taskID, err := c.attachmentRepo.GetTaskID(ctx, attachmentID)
	if err != nil {
		return false, err
	}

	// Права на вложение наследуются от прав на задачу
	switch action {
	case ActionUpload, ActionCreate:
		// Загружать файлы могут те, кто может редактировать задачу
		return c.checkTask(ctx, user, taskID, ActionEdit)

	case ActionView, ActionDownload:
		// Смотреть файлы могут те, кто может видеть задачу
		return c.checkTask(ctx, user, taskID, ActionView)

	case ActionDelete:
		// только автор или менеджер
		uploadedBy, _ := c.attachmentRepo.GetUploadedBy(ctx, attachmentID)
		if user.ID == uploadedBy {
			return true, nil
		}
		return c.checkTask(ctx, user, taskID, ActionEdit)

	default:
		return false, nil
	}
}
