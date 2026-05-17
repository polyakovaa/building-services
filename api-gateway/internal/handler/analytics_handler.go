package handler

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"building-services/api-gateway/internal/util"
	analyticsv1 "building-services/gen/analytics/v1"
	projectv1 "building-services/gen/project/v1"

	"github.com/gin-gonic/gin"
)

type AnalyticsHandler struct {
	client   analyticsv1.AnalyticsServiceClient
	projects projectv1.ProjectServiceClient
}

func NewAnalyticsHandler(client analyticsv1.AnalyticsServiceClient, projects projectv1.ProjectServiceClient) *AnalyticsHandler {
	return &AnalyticsHandler{client: client, projects: projects}
}

func sliceContains(hay []string, needle string) bool {
	for _, s := range hay {
		if s == needle {
			return true
		}
	}
	return false
}

type analyticsScope struct {
	projectCSV   string
	unrestricted bool
	projectList  []string
	departmentID string
}

func (h *AnalyticsHandler) memberProjectScope(c *gin.Context, grpcCtx context.Context) (*analyticsScope, error) {
	roleVal, ok := c.Get("user_role")
	if !ok {
		return nil, fmt.Errorf("missing user_role")
	}
	role, _ := roleVal.(string)
	scope := &analyticsScope{}
	if role == "ROLE_DEPARTMENT_MANAGER" {
		userIDVal, ok := c.Get("user_id")
		if !ok {
			return nil, fmt.Errorf("missing user_id")
		}
		userID, _ := userIDVal.(string)
		user, err := h.projects.GetUser(grpcCtx, &projectv1.GetUserRequest{Id: userID})
		if err != nil {
			return nil, err
		}
		scope.departmentID = user.GetDepartmentId()
	}
	if role == "ROLE_DIRECTOR" {
		scope.unrestricted = true
		return scope, nil
	}
	resp, err := h.projects.ListProjects(grpcCtx, &projectv1.ListProjectsRequest{})
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(resp.Projects))
	for _, p := range resp.Projects {
		if p.GetId() != "" {
			out = append(out, p.GetId())
		}
	}
	for i := 0; i < len(out); i++ {
		if i > 0 {
			scope.projectCSV += ","
		}
		scope.projectCSV += out[i]
	}
	scope.projectList = out
	return scope, nil
}

func (s *analyticsScope) scopedDepartmentID(requested string) string {
	if s != nil && s.departmentID != "" {
		return s.departmentID
	}
	return requested
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

	scope, err := h.memberProjectScope(c, ctx)
	if err != nil {
		handleError(c, err)
		return
	}

	if !scope.unrestricted && len(scope.projectList) == 0 {
		c.JSON(http.StatusOK, &analyticsv1.DashboardResponse{})
		return
	}

	qProj := c.Query("project_id")
	if !scope.unrestricted && qProj != "" && !sliceContains(scope.projectList, qProj) {
		c.JSON(http.StatusForbidden, gin.H{"error": "project not in scope"})
		return
	}

	req := &analyticsv1.GetDashboardRequest{
		DepartmentId: scope.scopedDepartmentID(c.Query("department_id")),
		FromDate:     c.Query("from_date"),
		ToDate:       c.Query("to_date"),
	}
	if scope.unrestricted {
		req.ProjectId = qProj
	} else if qProj != "" {
		req.ProjectId = qProj
	} else {
		req.ProjectIds = scope.projectCSV
	}

	resp, err := h.client.GetDashboard(ctx, req)
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

	scope, err := h.memberProjectScope(c, ctx)
	if err != nil {
		handleError(c, err)
		return
	}

	if !scope.unrestricted && len(scope.projectList) == 0 {
		c.JSON(http.StatusOK, &analyticsv1.DepartmentWorkloadResponse{})
		return
	}

	qProj := c.Query("project_id")
	if !scope.unrestricted && qProj != "" && !sliceContains(scope.projectList, qProj) {
		c.JSON(http.StatusForbidden, gin.H{"error": "project not in scope"})
		return
	}

	departmentID := scope.scopedDepartmentID(c.Query("department_id"))
	projectID := qProj
	days := c.DefaultQuery("days", "30")

	var daysInt int32 = 30
	if days != "" {
		if d, err := strconv.Atoi(days); err == nil {
			daysInt = int32(d)
		}
	}

	req := &analyticsv1.GetDepartmentWorkloadRequest{
		DepartmentId: departmentID,
		Days:         daysInt,
		ProjectId:    projectID,
		FromDate:     c.Query("from_date"),
		ToDate:       c.Query("to_date"),
	}
	if !scope.unrestricted && projectID == "" {
		req.ProjectIds = scope.projectCSV
	}
	if scope.unrestricted && projectID == "" && scope.projectCSV == "" {
		req.SummaryScope = analyticsv1.AnalyticsSummaryScope_ANALYTICS_SUMMARY_SCOPE_ORGANIZATION
	}
	resp, err := h.client.GetDepartmentWorkload(ctx, req)
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

	scope, err := h.memberProjectScope(c, ctx)
	if err != nil {
		handleError(c, err)
		return
	}

	if !scope.unrestricted && len(scope.projectList) == 0 {
		c.JSON(http.StatusOK, &analyticsv1.ProjectTimelineResponse{Projects: nil})
		return
	}

	projectID := c.Query("project_id")
	departmentID := scope.scopedDepartmentID(c.Query("department_id"))

	if !scope.unrestricted && projectID != "" && !sliceContains(scope.projectList, projectID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "project not in scope"})
		return
	}

	req := &analyticsv1.GetProjectTimelineRequest{
		ProjectId:    projectID,
		DepartmentId: departmentID,
		FromDate:     c.Query("from_date"),
		ToDate:       c.Query("to_date"),
	}
	if !scope.unrestricted && projectID == "" {
		req.ProjectIds = scope.projectCSV
	}

	resp, err := h.client.GetProjectTimelineControl(ctx, req)
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

	scope, err := h.memberProjectScope(c, ctx)
	if err != nil {
		handleError(c, err)
		return
	}
	if !scope.unrestricted && len(scope.projectList) == 0 {
		c.JSON(http.StatusOK, &analyticsv1.TaskTrendsResponse{})
		return
	}
	qProj := c.Query("project_id")
	if !scope.unrestricted && qProj != "" && !sliceContains(scope.projectList, qProj) {
		c.JSON(http.StatusForbidden, gin.H{"error": "project not in scope"})
		return
	}
	departmentID := scope.scopedDepartmentID(c.Query("department_id"))
	projectID := qProj
	weeks := c.DefaultQuery("weeks", "8")
	fromDate := c.Query("from_date")
	toDate := c.Query("to_date")
	groupBy := c.Query("group_by")

	var weeksInt int32 = 8
	if weeks != "" {
		if w, err := strconv.Atoi(weeks); err == nil {
			weeksInt = int32(w)
		}
	}
	if groupBy == "day" {
		weeksInt = 1
	} else if fromDate != "" && toDate != "" {
		if from, err1 := time.Parse("2006-01-02", fromDate); err1 == nil {
			if to, err2 := time.Parse("2006-01-02", toDate); err2 == nil {
				days := int(to.Sub(from).Hours()/24) + 1
				if days <= 10 {
					weeksInt = 1
				}
			}
		}
	}

	req := &analyticsv1.GetTaskTrendsRequest{
		DepartmentId: departmentID,
		Weeks:        weeksInt,
		ProjectId:    projectID,
		FromDate:     fromDate,
		ToDate:       toDate,
		GroupBy:      groupBy,
	}
	if !scope.unrestricted && projectID == "" {
		req.ProjectIds = scope.projectCSV
	}
	if scope.unrestricted && projectID == "" && scope.projectCSV == "" {
		req.SummaryScope = analyticsv1.AnalyticsSummaryScope_ANALYTICS_SUMMARY_SCOPE_ORGANIZATION
	}

	resp, err := h.client.GetTaskTrends(ctx, req)
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

	scope, err := h.memberProjectScope(c, ctx)
	if err != nil {
		handleError(c, err)
		return
	}

	if !scope.unrestricted && len(scope.projectList) == 0 {
		c.JSON(http.StatusOK, &analyticsv1.EmployeeProductivityResponse{})
		return
	}

	qProj := c.Query("project_id")
	if !scope.unrestricted && qProj != "" && !sliceContains(scope.projectList, qProj) {
		c.JSON(http.StatusForbidden, gin.H{"error": "project not in scope"})
		return
	}

	departmentID := scope.scopedDepartmentID(c.Query("department_id"))
	fromDate := c.Query("from_date")
	toDate := c.Query("to_date")

	req := &analyticsv1.GetEmployeeProductivityRequest{
		DepartmentId: departmentID,
		FromDate:     fromDate,
		ToDate:       toDate,
		ProjectId:    qProj,
	}
	if !scope.unrestricted && qProj == "" {
		req.ProjectIds = scope.projectCSV
	}

	resp, err := h.client.GetEmployeeProductivity(ctx, req)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}
