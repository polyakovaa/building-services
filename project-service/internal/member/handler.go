package member

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
	projectv1.UnimplementedProjectMemberServiceServer
	service *Service
}

func NewHandler(s *Service) *Handler {
	return &Handler{
		service: s,
	}
}

func (h *Handler) AddMember(ctx context.Context, req *projectv1.AddMemberRequest) (*projectv1.ProjectMember, error) {
	start := time.Now()

	log.Printf("[REQUEST] AddMember: project_id=%s, user_id=%s", req.ProjectId, req.UserId)
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is nil")
	}

	member, err := h.service.AddMember(ctx, req)

	if err != nil {
		if grpcErr := errs.Handle(err, "AddMember"); grpcErr != nil {
			return nil, grpcErr
		}
	}

	log.Printf("[SUCCESS] CreateProject: id=%s, duration=%v", member.UserId, time.Since(start))
	return member, nil
}

func (h *Handler) GetMember(ctx context.Context, req *projectv1.GetMemberRequest) (*projectv1.ProjectMember, error) {
	start := time.Now()

	log.Printf("[REQUEST] GetMember: id: %s", req.UserId)
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is nil")
	}

	log.Printf("Received request: %+v", req)

	member, err := h.service.GetMember(ctx, req)
	if err != nil {
		if grpcErr := errs.Handle(err, "GetMember"); grpcErr != nil {
			return nil, grpcErr
		}
	}

	log.Printf("[SUCCESS] GetMember: id=%s, duration=%v", member.UserId, time.Since(start))
	return member, nil

}

func (h *Handler) ListMembers(ctx context.Context, req *projectv1.ListMembersRequest) (*projectv1.ListMembersResponse, error) {
	start := time.Now()
	log.Printf("[REQUEST] ListMembers:  project_id=%s", req.ProjectId)

	resp, err := h.service.ListMembers(ctx, req)
	if err != nil {
		if grpcErr := errs.Handle(err, "ListMembers"); grpcErr != nil {
			return nil, grpcErr
		}
	}

	log.Printf("[SUCCESS] ListMembers: count=%d, duration=%v", len(resp.Members), time.Since(start))
	return resp, nil
}

func (h *Handler) UpdateMember(ctx context.Context, req *projectv1.UpdateMemberRequest) (*projectv1.ProjectMember, error) {
	start := time.Now()
	log.Printf("[REQUEST] UpdateMember: id: %s", req.UserId)

	resp, err := h.service.UpdateMember(ctx, req)
	if err != nil {
		if grpcErr := errs.Handle(err, "UpdateMember"); grpcErr != nil {
			return nil, grpcErr
		}
	}

	log.Printf("[SUCCESS] UpdateMember: duration=%v", time.Since(start))
	return resp, nil
}

func (h *Handler) RemoveMember(ctx context.Context, req *projectv1.RemoveMemberRequest) (*emptypb.Empty, error) {
	start := time.Now()
	log.Printf("[REQUEST] RemoveMember: id: %s", req.UserId)

	resp, err := h.service.RemoveMember(ctx, req)
	if err != nil {
		if grpcErr := errs.Handle(err, "RemoveMember"); grpcErr != nil {
			return nil, grpcErr
		}
	}

	log.Printf("[SUCCESS] RemoveMember: duration=%v", time.Since(start))
	return resp, nil
}
