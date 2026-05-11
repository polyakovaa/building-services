package handler

import (
	"net/http"
	"strconv"

	"building-services/api-gateway/internal/util"
	analyticsv1 "building-services/gen/analytics/v1"

	"github.com/gin-gonic/gin"
)

type AnalyticsHandler struct {
	client analyticsv1.AnalyticsServiceClient
}

func NewAnalyticsHandler(client analyticsv1.AnalyticsServiceClient) *AnalyticsHandler {
	return &AnalyticsHandler{client: client}
}

func (h *AnalyticsHandler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/analytics/dashboard", h.GetDashboard)
	r.GET("/analytics/workload", h.GetDepartmentWorkload)
	r.GET("/analytics/projects/timeline", h.GetProjectTimeline)
	r.GET("/analytics/trends", h.GetTaskTrends)
	r.GET("/analytics/productivity", h.GetEmployeeProductivity)
}

func (h *AnalyticsHandler) GetDashboard(c *gin.Context) {
	ctx, err := util.GetGRPCContext(c)
	if err != nil {
		return
	}

	resp, err := h.client.GetDashboard(ctx, &analyticsv1.GetDashboardRequest{})
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *AnalyticsHandler) GetDepartmentWorkload(c *gin.Context) {
	ctx, err := util.GetGRPCContext(c)
	if err != nil {
		return
	}

	departmentID := c.Query("department_id")
	days := c.DefaultQuery("days", "30")

	var daysInt int32 = 30
	if days != "" {
		if d, err := strconv.Atoi(days); err == nil {
			daysInt = int32(d)
		}
	}

	resp, err := h.client.GetDepartmentWorkload(ctx, &analyticsv1.GetDepartmentWorkloadRequest{
		DepartmentId: departmentID,
		Days:         daysInt,
	})
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *AnalyticsHandler) GetProjectTimeline(c *gin.Context) {
	ctx, err := util.GetGRPCContext(c)
	if err != nil {
		return
	}

	projectID := c.Query("project_id")
	departmentID := c.Query("department_id")

	resp, err := h.client.GetProjectTimelineControl(ctx, &analyticsv1.GetProjectTimelineRequest{
		ProjectId:    projectID,
		DepartmentId: departmentID,
	})
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *AnalyticsHandler) GetTaskTrends(c *gin.Context) {
	ctx, err := util.GetGRPCContext(c)
	if err != nil {
		return
	}

	departmentID := c.Query("department_id")
	weeks := c.DefaultQuery("weeks", "8")

	var weeksInt int32 = 8
	if weeks != "" {
		if w, err := strconv.Atoi(weeks); err == nil {
			weeksInt = int32(w)
		}
	}

	resp, err := h.client.GetTaskTrends(ctx, &analyticsv1.GetTaskTrendsRequest{
		DepartmentId: departmentID,
		Weeks:        weeksInt,
	})
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *AnalyticsHandler) GetEmployeeProductivity(c *gin.Context) {
	ctx, err := util.GetGRPCContext(c)
	if err != nil {
		return
	}

	departmentID := c.Query("department_id")
	fromDate := c.Query("from_date")
	toDate := c.Query("to_date")

	resp, err := h.client.GetEmployeeProductivity(ctx, &analyticsv1.GetEmployeeProductivityRequest{
		DepartmentId: departmentID,
		FromDate:     fromDate,
		ToDate:       toDate,
	})
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}
