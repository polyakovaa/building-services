package department

import (
	"context"
	"log"
	"time"

	projectv1 "building-services/gen/project/v1"
	"building-services/project-service/internal/errs"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Handler struct {
	projectv1.UnimplementedDepartmentServiceServer
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) CreateDepartment(ctx context.Context, req *projectv1.CreateDepartmentRequest) (*projectv1.Department, error) {
	start := time.Now()
	log.Printf("[REQUEST] CreateDepartment: name=%s", req.Name)

	dept, err := h.service.CreateDepartment(ctx, req)
	if err != nil {
		log.Printf("[ERROR] CreateDepartment failed: %v", err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	log.Printf("[SUCCESS] CreateDepartment: id=%s, duration=%v", dept.Id, time.Since(start))
	return dept, nil
}

func (h *Handler) GetDepartment(ctx context.Context, req *projectv1.GetDepartmentRequest) (*projectv1.Department, error) {

	start := time.Now()

	log.Printf("[REQUEST] GetDepartment: id: %s", req.Id)
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is nil")
	}
	dept, err := h.service.GetDepartment(ctx, req)

	if err != nil {
		if grpcErr := errs.Handle(err, "GetDepartment"); grpcErr != nil {
			return nil, grpcErr
		}
	}

	log.Printf("[SUCCESS] GetDepartment: id=%s, duration=%v", dept.Id, time.Since(start))
	return dept, nil
}

func (h *Handler) ListDepartments(ctx context.Context, req *projectv1.ListDepartmentsRequest) (*projectv1.ListDepartmentsResponse, error) {

	start := time.Now()
	log.Printf("[REQUEST] ListDepartments")

	resp, err := h.service.ListDepartments(ctx, req)
	if err != nil {
		if grpcErr := errs.Handle(err, "ListProjects"); grpcErr != nil {
			return nil, grpcErr
		}
	}

	log.Printf("[SUCCESS] ListProjects: count=%d, duration=%v", len(resp.Departments), time.Since(start))
	return resp, nil
}

func (h *Handler) UpdateDepartment(ctx context.Context, req *projectv1.UpdateDepartmentRequest) (*projectv1.Department, error) {
	start := time.Now()
	log.Printf("[REQUEST] UpdateDepartment: id: %s", req.Id)

	resp, err := h.service.UpdateDepartment(ctx, req)
	if err != nil {
		if grpcErr := errs.Handle(err, "UpdateDepartment"); grpcErr != nil {
			return nil, grpcErr
		}
	}

	log.Printf("[SUCCESS] UpdateDepartment: duration=%v", time.Since(start))
	return resp, nil
}

func (h *Handler) DeleteDepartment(ctx context.Context, req *projectv1.DeleteDepartmentRequest) (*emptypb.Empty, error) {
	start := time.Now()
	log.Printf("[REQUEST] DeleteDepartment: id: %s", req.Id)

	resp, err := h.service.DeleteDepartment(ctx, req)
	if err != nil {
		if grpcErr := errs.Handle(err, "DeleteDepartment"); grpcErr != nil {
			return nil, grpcErr
		}
	}

	log.Printf("[SUCCESS] DeleteDepartment: duration=%v", time.Since(start))
	return resp, nil
}

func (h *Handler) AssignUserToDepartment(ctx context.Context, req *projectv1.AssignUserToDepartmentRequest) (*emptypb.Empty, error) {
	start := time.Now()
	log.Printf("[REQUEST] AssignUserToDepartment: user_id: %s department_id %s", req.UserId, req.DepartmentId)

	resp, err := h.service.AssignUserToDepartment(ctx, req)
	if err != nil {
		if grpcErr := errs.Handle(err, "AssignUserToDepartment"); grpcErr != nil {
			return nil, grpcErr
		}
	}

	log.Printf("[SUCCESS] AssignUserToDepartment: duration=%v", time.Since(start))
	return resp, nil
}

func (h *Handler) RemoveUserFromDepartment(ctx context.Context, req *projectv1.RemoveUserFromDepartmentRequest) (*emptypb.Empty, error) {
	start := time.Now()
	log.Printf("[REQUEST] RemoveUserFromDepartment: user_id: %s department_id %s", req.UserId, req.DepartmentId)

	resp, err := h.service.RemoveUserFromDepartment(ctx, req)
	if err != nil {
		if grpcErr := errs.Handle(err, "RemoveUserFromDepartment"); grpcErr != nil {
			return nil, grpcErr
		}
	}

	log.Printf("[SUCCESS] RemoveUserFromDepartment: duration=%v", time.Since(start))
	return resp, nil
}

func (h *Handler) GetDepartmentUsers(ctx context.Context, req *projectv1.GetDepartmentUsersRequest) (*projectv1.GetDepartmentUsersResponse, error) {
	start := time.Now()
	log.Printf("[REQUEST] GetDepartmentUsers:  department_id %s", req.DepartmentId)

	users, err := h.service.GetDepartmentUsers(ctx, req)
	if err != nil {
		if grpcErr := errs.Handle(err, "GetDepartmentUsers"); grpcErr != nil {
			return nil, grpcErr
		}
	}
	log.Printf("[SUCCESS] GetDepartmentUsers: duration=%v", time.Since(start))
	return users, nil

}
