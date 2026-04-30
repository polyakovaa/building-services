package attachment

import (
	projectv1 "building-services/gen/project/v1"
	"building-services/project-service/internal/authz"
	"building-services/project-service/internal/errs"
	"building-services/project-service/internal/util"
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Service struct {
	attachmentRepo AttachmentRepo
	taskRepo       TaskRepo
	authz          PermissionChecker
}

func NewService(taskRepo TaskRepo,
	attachmentRepo AttachmentRepo,
	authz PermissionChecker) *Service {
	return &Service{
		attachmentRepo: attachmentRepo,
		taskRepo:       taskRepo,
		authz:          authz,
	}
}

type AttachmentRepo interface {
	Create(ctx context.Context, att *projectv1.Attachment) error
	FindByID(ctx context.Context, id string) (*projectv1.Attachment, error)
	ListByTask(ctx context.Context, taskID string) ([]*projectv1.Attachment, error)
	Delete(ctx context.Context, id string) error
}

type TaskRepo interface {
	FindByID(ctx context.Context, id string) (*projectv1.Task, error)
}
type PermissionChecker interface {
	Check(ctx context.Context, userID string, resourceType string, resourceID string, action string) (bool, error)
}

func (s *Service) AddAttachment(ctx context.Context, req *projectv1.AddAttachmentRequest) (*projectv1.Attachment, error) {
	userID, err := util.GetFromContext(ctx, "user_id")
	if err != nil {
		return nil, fmt.Errorf("failed to get user_id: %w", err)
	}

	_, err = s.taskRepo.FindByID(ctx, req.TaskId)
	if err != nil {
		return nil, errs.ErrTaskNotFound
	}

	ok, err := s.authz.Check(ctx, userID, authz.ResourceAttachment, req.TaskId, authz.ActionUpload)
	if err != nil || !ok {
		return nil, errs.ErrNoPermission
	}

	attachment := &projectv1.Attachment{
		Id:          uuid.New().String(),
		TaskId:      req.TaskId,
		FileUrl:     req.FileUrl,
		Type:        req.Type,
		FileName:    req.FileName,
		FileSize:    req.FileSize,
		UploadedBy:  userID,
		Description: req.Description,
	}

	if err := s.attachmentRepo.Create(ctx, attachment); err != nil {
		return nil, fmt.Errorf("failed to create attachment: %w", err)
	}

	return attachment, nil
}

func (s *Service) DeleteAttachment(ctx context.Context, req *projectv1.DeleteAttachmentRequest) (*emptypb.Empty, error) {
	userID, err := util.GetFromContext(ctx, "user_id")
	if err != nil {
		return nil, err
	}
	ok, err := s.authz.Check(ctx, userID, authz.ResourceAttachment, req.Id, authz.ActionDelete)
	if err != nil || !ok {
		return nil, errs.ErrNoPermission
	}

	if req.Id == "" {
		return nil, fmt.Errorf("%w: attachment id required", errs.ErrInvalidInput)
	}

	if err := s.attachmentRepo.Delete(ctx, req.Id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &emptypb.Empty{}, nil
		}
		return nil, fmt.Errorf("failed to delete attacjment: %w", err)
	}

	return &emptypb.Empty{}, nil
}

func (s *Service) GetAttachment(ctx context.Context, req *projectv1.GetAttachmentRequest) (*projectv1.Attachment, error) {
	userID, err := util.GetFromContext(ctx, "user_id")
	if err != nil {
		return nil, fmt.Errorf("failed to get user_id: %w", err)
	}

	ok, err := s.authz.Check(ctx, userID, authz.ResourceAttachment, req.Id, authz.ActionView)
	if err != nil || !ok {
		return nil, errs.ErrNoPermission
	}

	if req.Id == "" {
		return nil, fmt.Errorf("%w: attachment id required", errs.ErrInvalidInput)
	}

	att, err := s.attachmentRepo.FindByID(ctx, req.Id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errs.ErrProjectNotFound
		}
		return nil, fmt.Errorf("failed to get attachment: %w", err)
	}

	return att, nil
}

func (s *Service) ListAttachments(ctx context.Context, req *projectv1.ListAttachmentsRequest) (*projectv1.ListAttachmentsResponse, error) {
	userID, err := util.GetFromContext(ctx, "user_id")
	if err != nil {
		return nil, fmt.Errorf("failed to get user_id: %w", err)
	}
	ok, err := s.authz.Check(ctx, userID, authz.ResourceTask, req.TaskId, authz.ActionView)
	if err != nil || !ok {
		return nil, errs.ErrNoPermission
	}

	_, err = s.taskRepo.FindByID(ctx, req.TaskId)
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	att, err := s.attachmentRepo.ListByTask(ctx, req.TaskId)
	if err != nil {
		return nil, fmt.Errorf("failed to list attacments: %w", err)
	}

	return &projectv1.ListAttachmentsResponse{
		Attachments: att,
	}, nil

}
