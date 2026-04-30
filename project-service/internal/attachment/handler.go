package attachment

import (
	projectv1 "building-services/gen/project/v1"
	"building-services/project-service/internal/errs"
	"context"
	"log"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Handler struct {
	projectv1.UnimplementedAttachmentServiceServer
	service *Service
}

func NewHandler(s *Service) *Handler {
	return &Handler{
		service: s,
	}
}

func (h *Handler) AddAttachment(ctx context.Context, req *projectv1.AddAttachmentRequest) (*projectv1.Attachment, error) {
	start := time.Now()

	log.Printf("[REQUEST] AddAttachment: task_id=%s", req.TaskId)
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is nil")
	}

	task, err := h.service.AddAttachment(ctx, req)

	if err != nil {
		if grpcErr := errs.Handle(err, "AddAttachment"); grpcErr != nil {
			return nil, grpcErr
		}
	}

	log.Printf("[SUCCESS] AddAttachment: id=%s, duration=%v", task.Id, time.Since(start))
	return task, nil

}

func (h *Handler) GetAttachment(ctx context.Context, req *projectv1.GetAttachmentRequest) (*projectv1.Attachment, error) {
	start := time.Now()

	log.Printf("[REQUEST] GetAttachment: id: %s", req.Id)
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is nil")
	}

	log.Printf("Received request: %+v", req)

	task, err := h.service.GetAttachment(ctx, req)
	if err != nil {
		if grpcErr := errs.Handle(err, "GetAttachment"); grpcErr != nil {
			return nil, grpcErr
		}
	}

	log.Printf("[SUCCESS] GetAttachment: id=%s, duration=%v", task.Id, time.Since(start))
	return task, nil

}

func (h *Handler) ListAttachments(ctx context.Context, req *projectv1.ListAttachmentsRequest) (*projectv1.ListAttachmentsResponse, error) {
	start := time.Now()
	log.Printf("[REQUEST] ListAttachments: task_id=%v", req.TaskId)

	resp, err := h.service.ListAttachments(ctx, req)
	if err != nil {
		if grpcErr := errs.Handle(err, "ListAttachments"); grpcErr != nil {
			return nil, grpcErr
		}
	}

	log.Printf("[SUCCESS] ListAttachments: count=%d, duration=%v", len(resp.Attachments), time.Since(start))
	return resp, nil
}

func (h *Handler) DeleteAttachment(ctx context.Context, req *projectv1.DeleteAttachmentRequest) (*emptypb.Empty, error) {
	start := time.Now()
	log.Printf("[REQUEST] DeleteAttachment: id: %s", req.Id)

	resp, err := h.service.DeleteAttachment(ctx, req)
	if err != nil {
		if grpcErr := errs.Handle(err, "DeleteAttachment"); grpcErr != nil {
			return nil, grpcErr
		}
	}

	log.Printf("[SUCCESS] DeleteAttachment: duration=%v", time.Since(start))
	return resp, nil
}
