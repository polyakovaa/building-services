package service

import (
	"building-services/analytics-service/internal/repository"
	"math"

	analyticsv1 "building-services/gen/analytics/v1"

	"log"

	"time"
)

type Service struct {
	repo Repository
}

type Repository interface {
	SaveRawEvent(event repository.RawEvent) error
	GetDepartmentWorkload(departmentID string, days int) ([]*analyticsv1.DepartmentWorkload, error)
	UpsertDepartmentMetric(departmentID, date string, field string, value int) error
	GetTaskCreatedAt(taskID string) (time.Time, error)
	GetTaskTrends(departmentID string, weeks int) ([]*analyticsv1.WeeklyTrend, error)
	GetDashboardStats() (activeProjects, totalTasks, overdueTasks int, completionRate, onTimeRate float64, err error)
	UpsertTaskAnalytics(taskID, projectID, departmentID, assignedUserID, createdBy string, createdAt time.Time, status int32, dueDate *time.Time) error
	UpdateTaskCompletion(taskID string, completedAt time.Time, isOverdue bool, cycleTimeDays, delayedDays int) error
	UpsertProjectTimelineControl(projectID, projectName, departmentID string, startDate, endDate *time.Time) error
	UpdateProjectMetrics(projectID string, totalTasks, completedOnTime, overdueTasks int32, onTimeRate, avgDelayDays float64) error
	UpsertWeeklyTrends(week time.Time, departmentID string, tasksCreated, tasksCompleted, tasksOverdue int32, completionRate, onTimeRate float64) error
	UpsertEmployeeProductivity(userID, fullName, email, departmentID string, date time.Time, tasksCompleted, tasksOverdue int32, avgCycleTime, completionRate, onTimeRate float64) error
	GetProjectTimelineControl(projectID, departmentID string) ([]*analyticsv1.ProjectTimelineControl, error)
	GetEmployeeProductivity(departmentID, fromDate, toDate string) ([]*analyticsv1.EmployeeProductivity, error)
	GetProjectStats(projectID string) (totalTasks, completedOnTime, overdueTasks int32, onTimeRate, avgDelayDays float64, err error)
}

func NewService(repo Repository) *Service {

	return &Service{

		repo: repo,
	}

}

func (s *Service) GetDashboardStats() (*analyticsv1.DashboardResponse, error) {

	activeProjects, totalTasks, overdueTasks, completionRate, onTimeRate, err := s.repo.GetDashboardStats()

	if err != nil {

		return nil, err

	}

	workloads, err := s.repo.GetDepartmentWorkload("", 30)

	if err != nil {

		return nil, err

	}

	trends, err := s.repo.GetTaskTrends("", 8)

	if err != nil {

		return nil, err

	}

	return &analyticsv1.DashboardResponse{

		ActiveProjects: int32(activeProjects),

		TotalTasks: int32(totalTasks),

		OverdueTasks: int32(overdueTasks),

		CompletionRate: completionRate,

		OnTimeRate: onTimeRate,

		DepartmentWorkload: workloads,

		WeeklyTrend: trends,
	}, nil

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

		return s.handleProjectCreated(event)

	case "project.status_changed":

		return s.handleProjectStatusChanged(event)

	default:

		return nil

	}

}

func (s *Service) handleTaskCreated(event map[string]interface{}) error {

	taskID, _ := event["task_id"].(string)

	projectID, _ := event["project_id"].(string)

	departmentID, _ := event["department_id"].(string)

	userID, _ := event["user_id"].(string)

	actorUserID, _ := event["actor_user_id"].(string)

	occurredAtStr, _ := event["occurred_at"].(string)

	status, _ := event["status"].(float64)

	dueDateStr, _ := event["deadline"].(string)

	if taskID == "" || projectID == "" {

		return nil

	}

	occurredAt, err := time.Parse(time.RFC3339Nano, occurredAtStr)

	if err != nil {

		occurredAt = time.Now()

	}

	var dueDate *time.Time

	if dueDateStr != "" {

		if parsed, err := time.Parse(time.RFC3339Nano, dueDateStr); err == nil {

			dueDate = &parsed

		}

	}

	if err := s.repo.UpsertTaskAnalytics(taskID, projectID, departmentID, userID, actorUserID, occurredAt, int32(status), dueDate); err != nil {

		return err

	}

	date := occurredAt.Format("2006-01-02")

	if departmentID != "" {

		if err := s.repo.UpsertDepartmentMetric(departmentID, date, "total_tasks", 1); err != nil {

			return err

		}

		if err := s.repo.UpsertDepartmentMetric(departmentID, date, "wip_count", 1); err != nil {

			return err

		}

		week := s.getWeekStart(occurredAt)

		if err := s.repo.UpsertWeeklyTrends(week, departmentID, 1, 0, 0, 0, 0); err != nil {

			return err

		}

	}

	return nil

}

func (s *Service) handleTaskStatusChanged(event map[string]interface{}) error {

	taskID, _ := event["task_id"].(string)

	projectID, _ := event["project_id"].(string)

	departmentID, _ := event["assignee_department_id"].(string)

	assigneeUserID, _ := event["assignee_user_id"].(string)

	actorUserID, _ := event["actor_user_id"].(string)

	toStatus, _ := event["to_status"].(float64)

	dueDateStr, _ := event["deadline"].(string)

	occurredAtStr, _ := event["occurred_at"].(string)

	log.Printf("handleTaskStatusChanged: taskID=%s, projectID=%s, departmentID='%s', toStatus=%v, deadline='%s'",

		taskID, projectID, departmentID, toStatus, dueDateStr)

	if taskID == "" {

		return nil

	}

	occurredAt, err := time.Parse(time.RFC3339Nano, occurredAtStr)

	if err != nil {

		occurredAt = time.Now()

		log.Printf("handleTaskStatusChanged: Failed to parse occurredAt, using now: %s", occurredAt.Format(time.RFC3339))

	}

	var dueDate *time.Time

	if dueDateStr != "" {

		if parsed, err := time.Parse(time.RFC3339Nano, dueDateStr); err == nil {

			dueDate = &parsed

			log.Printf("handleTaskStatusChanged: Parsed deadline: %s", dueDate.Format(time.RFC3339))

		} else {

			log.Printf("handleTaskStatusChanged: Failed to parse deadline '%s': %v", dueDateStr, err)

		}

	} else {

		log.Printf("handleTaskStatusChanged: No deadline provided")

	}

	date := occurredAt.Format("2006-01-02")

	if err := s.repo.UpsertTaskAnalytics(taskID, projectID, departmentID, assigneeUserID, actorUserID, occurredAt, int32(toStatus), dueDate); err != nil {

		log.Printf("handleTaskStatusChanged: Failed to upsert task analytics: %v", err)

		return err

	}

	if int32(toStatus) == 3 {

		log.Printf("handleTaskStatusChanged: Task completed, calculating metrics")

		taskCreatedAt, err := s.repo.GetTaskCreatedAt(taskID)

		if err != nil {

			log.Printf("handleTaskStatusChanged: Failed to get task created time, using completion time: %v", err)

			taskCreatedAt = occurredAt

		}

		cycleTimeDays := int(math.Ceil(occurredAt.Sub(taskCreatedAt).Hours() / 24))

		log.Printf("handleTaskStatusChanged: cycleTimeDays=%d (created: %s, completed: %s)",

			cycleTimeDays, taskCreatedAt.Format(time.RFC3339), occurredAt.Format(time.RFC3339))

		// Check if task is overdue and calculate delay

		isOverdue := false

		delayedDays := 0

		// Parse due date if available

		if dueDate != nil {

			if occurredAt.After(*dueDate) {

				isOverdue = true

				delayedDays = int(occurredAt.Sub(*dueDate).Hours() / 24)

				log.Printf("handleTaskStatusChanged: Task is overdue! delayedDays=%d (deadline: %s, completed: %s)",

					delayedDays, dueDate.Format(time.RFC3339), occurredAt.Format(time.RFC3339))

			} else {

				log.Printf("handleTaskStatusChanged: Task completed on time (deadline: %s, completed: %s)",

					dueDate.Format(time.RFC3339), occurredAt.Format(time.RFC3339))

			}

		} else {

			log.Printf("handleTaskStatusChanged: No deadline, cannot determine if overdue")

		}

		if err := s.repo.UpdateTaskCompletion(taskID, occurredAt, isOverdue, cycleTimeDays, delayedDays); err != nil {

			log.Printf("handleTaskStatusChanged: Failed to update task completion: %v", err)

			return err

		}

		if departmentID != "" {

			log.Printf("handleTaskStatusChanged: Updating department metrics for departmentID=%s", departmentID)

			if err := s.repo.UpsertDepartmentMetric(departmentID, date, "completed_tasks", 1); err != nil {

				log.Printf("handleTaskStatusChanged: Failed to update completed_tasks: %v", err)

				return err

			}

			if err := s.repo.UpsertDepartmentMetric(departmentID, date, "wip_count", -1); err != nil {

				log.Printf("handleTaskStatusChanged: Failed to update wip_count: %v", err)

				return err

			}

			// Update weekly trends

			week := s.getWeekStart(occurredAt)

			if err := s.repo.UpsertWeeklyTrends(week, departmentID, 0, 1, 0, 0, 0); err != nil {

				log.Printf("handleTaskStatusChanged: Failed to update weekly trends: %v", err)

			}

			if err := s.updateProjectMetrics(projectID); err != nil {

				log.Printf("Failed to update project metrics: %v", err)

			}

		} else {

			log.Printf("handleTaskStatusChanged: No departmentID, skipping department metrics update")

		}

	}

	return nil

}

func (s *Service) handleTaskAssigned(event map[string]interface{}) error {

	taskID, _ := event["task_id"].(string)

	projectID, _ := event["project_id"].(string)

	toUserID, _ := event["to_user_id"].(string)

	toDepartmentID, _ := event["to_department_id"].(string)

	actorUserID, _ := event["actor_user_id"].(string)

	occurredAtStr, _ := event["occurred_at"].(string)

	if taskID == "" || toUserID == "" {

		return nil

	}

	occurredAt, err := time.Parse(time.RFC3339Nano, occurredAtStr)

	if err != nil {

		occurredAt = time.Now()

	}

	if err := s.repo.UpsertTaskAnalytics(taskID, projectID, toDepartmentID, toUserID, actorUserID, occurredAt, 0, nil); err != nil {

		return err

	}

	return nil

}

func (s *Service) handleTaskDeadlineChanged(event map[string]interface{}) error {

	taskID, _ := event["task_id"].(string)

	projectID, _ := event["project_id"].(string)

	assigneeUserID, _ := event["assignee_user_id"].(string)

	assigneeDepartmentID, _ := event["assignee_department_id"].(string)

	actorUserID, _ := event["actor_user_id"].(string)

	newDeadlineStr, _ := event["new_deadline"].(string)

	occurredAtStr, _ := event["occurred_at"].(string)

	if taskID == "" {

		return nil

	}

	occurredAt, err := time.Parse(time.RFC3339Nano, occurredAtStr)

	if err != nil {

		occurredAt = time.Now()

	}

	var newDeadline *time.Time

	if newDeadlineStr != "" {

		if parsed, err := time.Parse(time.RFC3339Nano, newDeadlineStr); err == nil {

			newDeadline = &parsed

		}

	}

	if err := s.repo.UpsertTaskAnalytics(taskID, projectID, assigneeDepartmentID, assigneeUserID, actorUserID, occurredAt, 0, newDeadline); err != nil {

		return err

	}

	return nil

}

func (s *Service) handleProjectCreated(event map[string]interface{}) error {

	projectID, _ := event["project_id"].(string)

	projectName, _ := event["project_name"].(string)

	startDateStr, _ := event["start_date"].(string)

	endDateStr, _ := event["end_date"].(string)

	if projectID == "" {

		return nil

	}

	var startDate, endDate *time.Time

	if startDateStr != "" {

		if parsed, err := time.Parse(time.RFC3339Nano, startDateStr); err == nil {

			startDate = &parsed

		}

	}

	if endDateStr != "" {

		if parsed, err := time.Parse(time.RFC3339Nano, endDateStr); err == nil {

			endDate = &parsed

		}

	}

	// Note: department_id is not available in project.created event, so we pass empty string

	// The repository method should handle empty department_id appropriately

	if err := s.repo.UpsertProjectTimelineControl(projectID, projectName, "", startDate, endDate); err != nil {

		return err

	}

	return nil

}

func (s *Service) handleProjectStatusChanged(event map[string]interface{}) error {

	projectID, _ := event["project_id"].(string)

	projectName, _ := event["project_name"].(string)

	toStatus, _ := event["to_status"].(float64)

	occurredAtStr, _ := event["occurred_at"].(string)

	if projectID == "" {

		return nil

	}

	occurredAt, err := time.Parse(time.RFC3339Nano, occurredAtStr)

	if err != nil {

		occurredAt = time.Now()

	}

	var endDate *time.Time

	if int32(toStatus) == 2 {

		endDate = &occurredAt

	}

	if err := s.repo.UpsertProjectTimelineControl(projectID, projectName, "", nil, endDate); err != nil {

		return err

	}

	return nil

}

func (s *Service) getWeekStart(t time.Time) time.Time {

	year, week := t.ISOWeek()

	weekStart := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)

	for weekStart.Weekday() != time.Monday {

		weekStart = weekStart.AddDate(0, 0, -1)

	}

	weekStart = weekStart.AddDate(0, 0, (int(week)-1)*7)

	return weekStart

}

func (s *Service) GetDepartmentWorkload(departmentID string, days int) ([]*analyticsv1.DepartmentWorkload, error) {

	workloads, err := s.repo.GetDepartmentWorkload(departmentID, days)

	if err != nil {

		return nil, err

	}

	for _, w := range workloads {

		// Calculate productivity rate: completed tasks / total tasks * 100

		totalTasks := w.Completed + w.Wip

		if totalTasks > 0 {

			w.Productivity = float64(w.Completed) / float64(totalTasks) * 100

		} else {

			w.Productivity = 0.0

		}

		// On-time rate is already calculated in database, but ensure it's valid

		if w.OnTimeRate < 0 || w.OnTimeRate > 100 {

			w.OnTimeRate = 0.0

		}

	}

	return workloads, nil

}

func (s *Service) GetTaskTrends(departmentID string, weeks int) ([]*analyticsv1.WeeklyTrend, error) {

	trends, err := s.repo.GetTaskTrends(departmentID, weeks)

	if err != nil {

		return nil, err

	}

	for _, t := range trends {

		// Calculate completion rate: completed tasks / created tasks * 100

		if t.Created > 0 {

			t.CompletionRate = float64(t.Completed) / float64(t.Created) * 100

		} else {

			t.CompletionRate = 0.0

		}

		// Calculate on-time rate: (completed - overdue) / completed * 100

		if t.Completed > 0 {

			onTimeTasks := t.Completed - t.Overdue

			if onTimeTasks >= 0 {

				t.OnTimeRate = float64(onTimeTasks) / float64(t.Completed) * 100

			} else {

				t.OnTimeRate = 0.0

			}

		} else {

			t.OnTimeRate = 0.0

		}

	}

	return trends, nil

}

func (s *Service) GetProjectTimelineControl(projectID, departmentID string) ([]*analyticsv1.ProjectTimelineControl, error) {

	projects, err := s.repo.GetProjectTimelineControl(projectID, departmentID)

	if err != nil {

		return nil, err

	}

	for _, p := range projects {

		// Calculate on-time rate

		if p.TotalTasks > 0 {

			p.OnTimeRate = float64(p.CompletedOnTime) / float64(p.TotalTasks) * 100

		} else {

			p.OnTimeRate = 0.0

		}

		// AvgDelayDays is already calculated in database, ensure it's valid

		if p.AvgDelayDays < 0 {

			p.AvgDelayDays = 0.0

		}

	}

	return projects, nil

}

func (s *Service) GetEmployeeProductivity(departmentID, fromDate, toDate string) ([]*analyticsv1.EmployeeProductivity, error) {

	employees, err := s.repo.GetEmployeeProductivity(departmentID, fromDate, toDate)

	if err != nil {

		return nil, err

	}

	for _, e := range employees {

		// Calculate completion rate: completed tasks / (completed + overdue) * 100

		totalTasks := e.TasksCompleted + e.TasksOverdue

		if totalTasks > 0 {

			e.CompletionRate = float64(e.TasksCompleted) / float64(totalTasks) * 100

		} else {

			e.CompletionRate = 0.0

		}

		// Calculate on-time rate: (completed - overdue) / completed * 100

		if e.TasksCompleted > 0 {

			onTimeTasks := e.TasksCompleted - e.TasksOverdue

			if onTimeTasks >= 0 {

				e.OnTimeRate = float64(onTimeTasks) / float64(e.TasksCompleted) * 100

			} else {

				e.OnTimeRate = 0.0

			}

		} else {

			e.OnTimeRate = 0.0

		}

	}

	return employees, nil

}

func (s *Service) updateProjectMetrics(projectID string) error {

	log.Printf("updateProjectMetrics: Calculating metrics for projectID=%s", projectID)

	totalTasks, completedOnTime, overdueTasks, onTimeRate, avgDelayDays, err := s.repo.GetProjectStats(projectID)

	if err != nil {

		log.Printf("updateProjectMetrics: Failed to get project stats: %v", err)

		return err

	}

	log.Printf("updateProjectMetrics: stats - totalTasks=%d, completedOnTime=%d, overdueTasks=%d, onTimeRate=%.2f, avgDelayDays=%.2f",

		totalTasks, completedOnTime, overdueTasks, onTimeRate, avgDelayDays)

	err = s.repo.UpdateProjectMetrics(projectID, totalTasks, completedOnTime, overdueTasks, onTimeRate, avgDelayDays)

	if err != nil {

		log.Printf("updateProjectMetrics: Failed to update project metrics: %v", err)

		return err

	}

	log.Printf("updateProjectMetrics: Successfully updated project metrics")

	return nil

}
