package task

import (
	projectv1 "building-services/gen/project/v1"
	"context"
)

type Service struct {
	taskRepo    TaskRepo
	projectRepo ProjectRepo
	memberRepo  MemberRepo
}

func NewService(taskRepo Repository, projectRepo ProjectRepo, memberRepo MemberRepo) *Service {
	return &Service{
		taskRepo:    taskRepo,
		projectRepo: projectRepo,
		memberRepo:  memberRepo,
	}
}

type MemberRepo interface {
	IsMember(ctx context.Context, projectID, userID string) (bool, error)
	CanAssignTask(ctx context.Context, projectID, userID string) (bool, error)
}

type ProjectRepo interface {
	Exists(ctx context.Context, id string) (bool, error)
	FindByID(ctx context.Context, id string) (*projectv1.Project, error)
	GetManagerID(ctx context.Context, id string) (string, error)
}

type TaskRepo interface {
}
