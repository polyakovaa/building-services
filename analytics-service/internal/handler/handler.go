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
	log.Printf("[REQUEST] GetDashboard")

	stats, err := h.service.GetDashboardStats()
	if err != nil {
		log.Printf("[ERROR] GetDashboard failed: %v", err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	log.Printf("[SUCCESS] GetDashboard: duration=%v", time.Since(start))
	return stats, nil
}

func (h *Handler) GetDepartmentWorkload(ctx context.Context, req *analyticsv1.GetDepartmentWorkloadRequest) (*analyticsv1.DepartmentWorkloadResponse, error) {
	start := time.Now()
	log.Printf("[REQUEST] GetDepartmentWorkload: department_id=%s", req.DepartmentId)

	workloads, err := h.service.GetDepartmentWorkload(req.DepartmentId, int(req.Days))
	if err != nil {
		log.Printf("[ERROR] GetDepartmentWorkload failed: %v", err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	log.Printf("[SUCCESS] GetDepartmentWorkload: count=%d, duration=%v", len(workloads), time.Since(start))
	return &analyticsv1.DepartmentWorkloadResponse{Workloads: workloads}, nil
}

func (h *Handler) GetTaskTrends(ctx context.Context, req *analyticsv1.GetTaskTrendsRequest) (*analyticsv1.TaskTrendsResponse, error) {
	start := time.Now()
	log.Printf("[REQUEST] GetTaskTrends: department_id=%s, weeks=%d", req.DepartmentId, req.Weeks)

	trends, err := h.service.GetTaskTrends(req.DepartmentId, int(req.Weeks))
	if err != nil {
		log.Printf("[ERROR] GetTaskTrends failed: %v", err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	log.Printf("[SUCCESS] GetTaskTrends: count=%d, duration=%v", len(trends), time.Since(start))
	return &analyticsv1.TaskTrendsResponse{Trends: trends}, nil
}

func (h *Handler) GetProjectTimelineControl(ctx context.Context, req *analyticsv1.GetProjectTimelineRequest) (*analyticsv1.ProjectTimelineResponse, error) {
	start := time.Now()
	log.Printf("[REQUEST] GetProjectTimelineControl: project_id=%s, department_id=%s", req.ProjectId, req.DepartmentId)

	projects, err := h.service.GetProjectTimelineControl(req.ProjectId, req.DepartmentId)
	if err != nil {
		log.Printf("[ERROR] GetProjectTimelineControl failed: %v", err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	log.Printf("[SUCCESS] GetProjectTimelineControl: count=%d, duration=%v", len(projects), time.Since(start))
	return &analyticsv1.ProjectTimelineResponse{Projects: projects}, nil
}

func (h *Handler) GetEmployeeProductivity(ctx context.Context, req *analyticsv1.GetEmployeeProductivityRequest) (*analyticsv1.EmployeeProductivityResponse, error) {
	start := time.Now()
	log.Printf("[REQUEST] GetEmployeeProductivity: department_id=%s, from_date=%s, to_date=%s", req.DepartmentId, req.FromDate, req.ToDate)

	employees, err := h.service.GetEmployeeProductivity(req.DepartmentId, req.FromDate, req.ToDate)
	if err != nil {
		log.Printf("[ERROR] GetEmployeeProductivity failed: %v", err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	log.Printf("[SUCCESS] GetEmployeeProductivity: count=%d, duration=%v", len(employees), time.Since(start))
	return &analyticsv1.EmployeeProductivityResponse{Employees: employees}, nil
}
