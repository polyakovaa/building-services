package service

import (
	"building-services/analytics-service/internal/repository"
	"building-services/analytics-service/internal/util"
	analyticsv1 "building-services/gen/analytics/v1"
	"math"
	"time"
)

type Repository interface {
	UpsertUser(user repository.User) error
	GetDepartmentWorkload(f repository.AnalyticsFilter, days int) ([]*analyticsv1.DepartmentWorkload, error)
	GetTaskCreatedAt(taskID string) (time.Time, error)
	GetTaskTrends(f repository.AnalyticsFilter, weeks int, groupBy string) ([]*analyticsv1.WeeklyTrend, error)
	GetDashboardStats(f repository.AnalyticsFilter) (activeProjects, totalTasks, overdueTasks int, completionRate, onTimeRate float64, err error)
	UpsertTaskAnalytics(taskID, projectID, departmentID, assignedUserID, createdBy string, createdAt time.Time, status int32, dueDate *time.Time) error
	UpdateTaskCompletion(taskID string, completedAt time.Time, isOverdue bool, cycleTimeDays, delayedDays int) error
	UpsertProject(projectID, projectName string, startDate, endDate *time.Time) error
	GetProjectTimeline(f repository.AnalyticsFilter) ([]*analyticsv1.ProjectTimelineControl, error)
	GetEmployeeProductivity(f repository.AnalyticsFilter) ([]*analyticsv1.EmployeeProductivity, error)
	PatchTaskLabor(taskID, activityTypeID string, plannedHours, actualHours float64) error
	GetLaborPlanFact(f repository.AnalyticsFilter, groupBy string) (*analyticsv1.LaborPlanFactResponse, error)
	GetDataFreshness() (time.Time, int32, error)
	UpsertDepartment(id, name string) error
	UpsertActivityType(id, name string, sortOrder int32) error
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetDashboard(req *analyticsv1.GetDashboardRequest) (*analyticsv1.DashboardResponse, error) {
	f := util.AnalyticsFilterFrom(req.DepartmentId, req.ProjectId, req.ProjectIds, req.FromDate, req.ToDate)

	activeProjects, totalTasks, overdueTasks, completionRate, onTimeRate, err := s.repo.GetDashboardStats(f)
	if err != nil {
		return nil, err
	}

	days := 30
	if f.HasDateRange() {
		from, errFrom := time.Parse("2006-01-02", f.FromDate)
		to, errTo := time.Parse("2006-01-02", f.ToDate)
		if errFrom == nil && errTo == nil {
			d := int(to.Sub(from).Hours()/24) + 1
			if d > 0 && d <= 366 {
				days = d
			}
		}
	}

	workloads, err := s.repo.GetDepartmentWorkload(f, days)
	if err != nil {
		return nil, err
	}
	weeks := 12
	groupBy := ""
	switch {
	case days <= 10:
		weeks = 1
		groupBy = "day"
	case days <= 40:
		weeks = 4
	}

	trends, err := s.repo.GetTaskTrends(f, weeks, groupBy)
	if err != nil {
		return nil, err
	}

	return &analyticsv1.DashboardResponse{
		ActiveProjects:     int32(activeProjects),
		TotalTasks:         int32(totalTasks),
		OverdueTasks:       int32(overdueTasks),
		CompletionRate:     completionRate,
		OnTimeRate:         onTimeRate,
		DepartmentWorkload: workloads,
		WeeklyTrend:        trends,
	}, nil
}

func (s *Service) GetDataFreshness(_ *analyticsv1.GetDataFreshnessRequest) (*analyticsv1.DataFreshnessResponse, error) {
	lastAt, tasksCount, err := s.repo.GetDataFreshness()
	if err != nil {
		return nil, err
	}
	resp := &analyticsv1.DataFreshnessResponse{TasksCount: tasksCount}
	if !lastAt.IsZero() {
		resp.LastEventAt = lastAt.UTC().Format(time.RFC3339)
	}
	return resp, nil
}

func (s *Service) GetDepartmentWorkload(req *analyticsv1.GetDepartmentWorkloadRequest) (*analyticsv1.DepartmentWorkloadResponse, error) {
	f := util.AnalyticsFilterFrom(req.DepartmentId, req.ProjectId, req.ProjectIds, req.FromDate, req.ToDate)
	days := int(req.Days)
	if days <= 0 {
		days = 30
	}
	workloads, err := s.repo.GetDepartmentWorkload(f, days)
	if err != nil {
		return nil, err
	}
	return &analyticsv1.DepartmentWorkloadResponse{Workloads: workloads}, nil
}

func (s *Service) GetTaskTrends(req *analyticsv1.GetTaskTrendsRequest) (*analyticsv1.TaskTrendsResponse, error) {
	f := util.AnalyticsFilterFrom(req.DepartmentId, req.ProjectId, req.ProjectIds, req.FromDate, req.ToDate)	
	weeks := int(req.Weeks)
	if weeks <= 0 {
		weeks = 8
	}
	trends, err := s.repo.GetTaskTrends(f, weeks, req.GetGroupBy())
	if err != nil {
		return nil, err
	}
	return &analyticsv1.TaskTrendsResponse{Trends: trends}, nil
}

func (s *Service) GetProjectTimeline(req *analyticsv1.GetProjectTimelineRequest) (*analyticsv1.ProjectTimelineResponse, error) {
	f := util.AnalyticsFilterFrom(req.DepartmentId, req.ProjectId, req.ProjectIds, req.FromDate, req.ToDate)
	projects, err := s.repo.GetProjectTimeline(f)
	if err != nil {
		return nil, err
	}
	return &analyticsv1.ProjectTimelineResponse{Projects: projects}, nil
}

func (s *Service) GetEmployeeProductivity(req *analyticsv1.GetEmployeeProductivityRequest) (*analyticsv1.EmployeeProductivityResponse, error) {
	f := util.AnalyticsFilterFrom(req.DepartmentId, req.ProjectId, req.ProjectIds, req.FromDate, req.ToDate)
	employees, err := s.repo.GetEmployeeProductivity(f)
	if err != nil {
		return nil, err
	}
	return &analyticsv1.EmployeeProductivityResponse{Employees: employees}, nil
}

func (s *Service) GetLaborPlanFact(req *analyticsv1.GetLaborPlanFactRequest) (*analyticsv1.LaborPlanFactResponse, error) {
	f := util.AnalyticsFilterFrom(req.DepartmentId, req.ProjectId, req.ProjectIds, req.FromDate, req.ToDate)
	return s.repo.GetLaborPlanFact(f, req.GroupBy)
}

func (s *Service) ProcessEvent(eventType string, event map[string]interface{}) error {
	switch eventType {
	case "task.created":
		return s.handleTaskCreated(event)
	case "task.assigned":
		return s.handleTaskAssigned(event)
	case "task.status_changed":
		return s.handleTaskStatusChanged(event)
	case "task.deadline_changed":
		return s.handleTaskDeadlineChanged(event)
	case "project.created":
		return s.upsertProjectFromEvent(event)
	case "project.updated":
		return s.upsertProjectFromEvent(event)
	case "project.status_changed":
		return s.handleProjectStatusChanged(event)
	case "task.updated":
		return s.handleTaskUpdated(event)
	case "department.created":
		return s.handleDepartmentCreated(event)
	case "activity_type.created":
		return s.handleActivityTypeCreated(event)
	default:
		return nil
	}
}

func (s *Service) handleDepartmentCreated(event map[string]interface{}) error {
	departmentID, _ := event["department_id"].(string)
	if departmentID == "" {
		departmentID, _ = event["id"].(string)
	}
	name, _ := event["name"].(string)
	return s.repo.UpsertDepartment(departmentID, name)
}

func (s *Service) handleActivityTypeCreated(event map[string]interface{}) error {
	id, _ := event["activity_type_id"].(string)
	if id == "" {
		id, _ = event["id"].(string)
	}
	name, _ := event["name"].(string)
	var sortOrder int32
	if v, ok := event["sort_order"].(float64); ok {
		sortOrder = int32(v)
	}
	return s.repo.UpsertActivityType(id, name, sortOrder)
}

func (s *Service) handleTaskCreated(event map[string]interface{}) error {
	taskID, _ := event["task_id"].(string)
	projectID, _ := event["project_id"].(string)
	if taskID == "" || projectID == "" {
		return nil
	}
	departmentID, _ := event["department_id"].(string)
	userID, _ := event["user_id"].(string)
	actorUserID, _ := event["actor_user_id"].(string)
	status, _ := event["status"].(float64)
	occurredAtStr, _ := event["occurred_at"].(string)
	dueDateStr, _ := event["deadline"].(string)
	occurredAt := util.ParseEventTime(occurredAtStr)
	dueDate := util.ParseOptionalTime(dueDateStr)

	if err := s.repo.UpsertTaskAnalytics(taskID, projectID, departmentID, userID, actorUserID, occurredAt, int32(status), dueDate); err != nil {
		return err
	}
	s.upsertAssigneeFromEvent(event)
	s.patchLaborFromEvent(event, taskID)
	return nil
}

func (s *Service) handleTaskStatusChanged(event map[string]interface{}) error {
	taskID, _ := event["task_id"].(string)
	if taskID == "" {
		return nil
	}
	projectID, _ := event["project_id"].(string)
	departmentID, _ := event["assignee_department_id"].(string)
	assigneeUserID, _ := event["assignee_user_id"].(string)
	actorUserID, _ := event["actor_user_id"].(string)
	toStatus, _ := event["to_status"].(float64)
	occurredAtStr, _ := event["occurred_at"].(string)
	dueDateStr, _ := event["deadline"].(string)
	occurredAt := util.ParseEventTime(occurredAtStr)
	dueDate := util.ParseOptionalTime(dueDateStr)

	if err := s.repo.UpsertTaskAnalytics(taskID, projectID, departmentID, assigneeUserID, actorUserID, occurredAt, int32(toStatus), dueDate); err != nil {
		return err
	}
	s.upsertAssigneeFromEvent(event)
	s.patchLaborFromEvent(event, taskID)
	if int32(toStatus) != 3 {
		return nil
	}

	taskCreatedAt, err := s.repo.GetTaskCreatedAt(taskID)
	if err != nil {
		taskCreatedAt = occurredAt
	}
	cycleTimeDays := int(math.Ceil(occurredAt.Sub(taskCreatedAt).Hours() / 24))
	isOverdue := false
	delayedDays := 0
	if dueDate != nil && occurredAt.After(*dueDate) {
		isOverdue = true
		delayedDays = int(occurredAt.Sub(*dueDate).Hours() / 24)
	}
	return s.repo.UpdateTaskCompletion(taskID, occurredAt, isOverdue, cycleTimeDays, delayedDays)
}

func (s *Service) handleTaskAssigned(event map[string]interface{}) error {
	taskID, _ := event["task_id"].(string)
	toUserID, _ := event["to_user_id"].(string)
	if taskID == "" || toUserID == "" {
		return nil
	}
	projectID, _ := event["project_id"].(string)
	toDepartmentID, _ := event["to_department_id"].(string)
	actorUserID, _ := event["actor_user_id"].(string)
	occurredAtStr, _ := event["occurred_at"].(string)
	occurredAt := util.ParseEventTime(occurredAtStr)

	if err := s.repo.UpsertTaskAnalytics(taskID, projectID, toDepartmentID, toUserID, actorUserID, occurredAt, 0, nil); err != nil {
		return err
	}
	s.upsertAssigneeFromEvent(event)
	return nil
}

func (s *Service) handleTaskDeadlineChanged(event map[string]interface{}) error {
	taskID, _ := event["task_id"].(string)
	if taskID == "" {
		return nil
	}
	projectID, _ := event["project_id"].(string)
	assigneeUserID, _ := event["assignee_user_id"].(string)
	assigneeDepartmentID, _ := event["assignee_department_id"].(string)
	actorUserID, _ := event["actor_user_id"].(string)
	occurredAtStr, _ := event["occurred_at"].(string)
	newDeadlineStr, _ := event["new_deadline"].(string)
	occurredAt := util.ParseEventTime(occurredAtStr)
	newDeadline := util.ParseOptionalTime(newDeadlineStr)

	if err := s.repo.UpsertTaskAnalytics(taskID, projectID, assigneeDepartmentID, assigneeUserID, actorUserID, occurredAt, 0, newDeadline); err != nil {
		return err
	}
	s.upsertAssigneeFromEvent(event)
	return nil
}


func (s *Service) upsertProjectFromEvent(event map[string]interface{}) error {
	projectID, _ := event["project_id"].(string)
	if projectID == "" {
		return nil
	}
	startDateStr, _ := event["start_date"].(string)
	endDateStr, _ := event["end_date"].(string)
	projectName, _ := event["project_name"].(string)
	return s.repo.UpsertProject(projectID, projectName, util.ParseOptionalTime(startDateStr), util.ParseOptionalTime(endDateStr))
}

func (s *Service) handleProjectStatusChanged(event map[string]interface{}) error {
	projectID, _ := event["project_id"].(string)
	if projectID == "" {
		return nil
	}
	occurredAtStr, _ := event["occurred_at"].(string)
	projectName, _ := event["project_name"].(string)
	toStatus, _ := event["to_status"].(float64)
	occurredAt := util.ParseEventTime(occurredAtStr)
	var endDate *time.Time
	if int32(toStatus) == 2 {
		endDate = &occurredAt
	}
	return s.repo.UpsertProject(projectID, projectName, nil, endDate)
}

func (s *Service) upsertAssigneeFromEvent(event map[string]interface{}) {
	if user, ok := util.AssigneeFromEvent(event); ok {
		_ = s.repo.UpsertUser(user)
	}
}

func (s *Service) patchLaborFromEvent(event map[string]interface{}, taskID string) {
	activityTypeID, plannedHours, actualHours := util.LaborFromEvent(event)
	if activityTypeID == "" && plannedHours <= 0 && actualHours <= 0 {
		return
	}
	_ = s.repo.PatchTaskLabor(taskID, activityTypeID, plannedHours, actualHours)
}

func (s *Service) handleTaskUpdated(event map[string]interface{}) error {
	taskID, _ := event["task_id"].(string)
	if taskID == "" {
		return nil
	}
	projectID, _ := event["project_id"].(string)
	assigneeUserID, _ := event["assignee_user_id"].(string)
	assigneeDepartmentID, _ := event["assignee_department_id"].(string)
	actorUserID, _ := event["actor_user_id"].(string)
	occurredAtStr, _ := event["occurred_at"].(string)
	occurredAt := util.ParseEventTime(occurredAtStr)
	if err := s.repo.UpsertTaskAnalytics(taskID, projectID, assigneeDepartmentID, assigneeUserID, actorUserID, occurredAt, 0, nil); err != nil {
		return err
	}
	s.upsertAssigneeFromEvent(event)
	s.patchLaborFromEvent(event, taskID)
	return nil
}
