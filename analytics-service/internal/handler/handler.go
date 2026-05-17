package handler

import (
	"building-services/analytics-service/internal/service"
	analyticsv1 "building-services/gen/analytics/v1"
	"context"
	"log"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Handler struct {
	analyticsv1.UnimplementedAnalyticsServiceServer
	service *service.Service
}

func NewHandler(service *service.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) GetDashboard(ctx context.Context, req *analyticsv1.GetDashboardRequest) (*analyticsv1.DashboardResponse, error) {
	start := time.Now()
	log.Printf("[REQUEST] GetDashboard: department_id=%s project_id=%s from=%s to=%s", req.DepartmentId, req.ProjectId, req.FromDate, req.ToDate)
	resp, err := h.service.GetDashboard(req)
	if err != nil {
		log.Printf("[ERROR] GetDashboard failed: %v", err)
		return nil, status.Error(codes.Internal, err.Error())
	}
	log.Printf("[SUCCESS] GetDashboard: duration=%v", time.Since(start))
	return resp, nil
}

func (h *Handler) GetDepartmentWorkload(ctx context.Context, req *analyticsv1.GetDepartmentWorkloadRequest) (*analyticsv1.DepartmentWorkloadResponse, error) {
	start := time.Now()
	log.Printf("[REQUEST] GetDepartmentWorkload: department_id=%s project_id=%s days=%d from=%s to=%s", req.DepartmentId, req.ProjectId, req.Days, req.FromDate, req.ToDate)
	resp, err := h.service.GetDepartmentWorkload(req)
	if err != nil {
		log.Printf("[ERROR] GetDepartmentWorkload failed: %v", err)
		return nil, status.Error(codes.Internal, err.Error())
	}
	log.Printf("[SUCCESS] GetDepartmentWorkload: count=%d, duration=%v", len(resp.Workloads), time.Since(start))
	return resp, nil
}

func (h *Handler) GetTaskTrends(ctx context.Context, req *analyticsv1.GetTaskTrendsRequest) (*analyticsv1.TaskTrendsResponse, error) {
	start := time.Now()
	log.Printf("[REQUEST] GetTaskTrends: department_id=%s project_id=%s weeks=%d", req.DepartmentId, req.ProjectId, req.Weeks)
	resp, err := h.service.GetTaskTrends(req)
	if err != nil {
		log.Printf("[ERROR] GetTaskTrends failed: %v", err)
		return nil, status.Error(codes.Internal, err.Error())
	}
	log.Printf("[SUCCESS] GetTaskTrends: count=%d, duration=%v", len(resp.Trends), time.Since(start))
	return resp, nil
}

func (h *Handler) GetProjectTimelineControl(ctx context.Context, req *analyticsv1.GetProjectTimelineRequest) (*analyticsv1.ProjectTimelineResponse, error) {
	start := time.Now()
	log.Printf("[REQUEST] GetProjectTimeline: project_id=%s department_id=%s project_ids=%s from=%s to=%s", req.ProjectId, req.DepartmentId, req.ProjectIds, req.FromDate, req.ToDate)
	resp, err := h.service.GetProjectTimeline(req)
	if err != nil {
		log.Printf("[ERROR] GetProjectTimeline failed: %v", err)
		return nil, status.Error(codes.Internal, err.Error())
	}
	log.Printf("[SUCCESS] GetProjectTimeline: count=%d, duration=%v", len(resp.Projects), time.Since(start))
	return resp, nil
}

func (h *Handler) GetEmployeeProductivity(ctx context.Context, req *analyticsv1.GetEmployeeProductivityRequest) (*analyticsv1.EmployeeProductivityResponse, error) {
	start := time.Now()
	log.Printf("[REQUEST] GetEmployeeProductivity: department_id=%s project_id=%s project_ids=%s from=%s to=%s", req.DepartmentId, req.ProjectId, req.ProjectIds, req.FromDate, req.ToDate)
	resp, err := h.service.GetEmployeeProductivity(req)
	if err != nil {
		log.Printf("[ERROR] GetEmployeeProductivity failed: %v", err)
		return nil, status.Error(codes.Internal, err.Error())
	}
	log.Printf("[SUCCESS] GetEmployeeProductivity: count=%d, duration=%v", len(resp.Employees), time.Since(start))
	return resp, nil
}
