package repository

import (
	analyticsv1 "building-services/gen/analytics/v1"
	"database/sql"
	"fmt"
	"log"
	"time"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

type RawEvent struct {
	ID           string
	EventType    string
	ProjectID    string
	TaskID       string
	UserID       string
	DepartmentID string
	ActorUserID  string
	OccurredAt   time.Time
	Payload      []byte
}

func (r *Repository) SaveRawEvent(event RawEvent) error {
	query := `INSERT INTO events_raw ( event_type, project_id, task_id, user_id, department_id, actor_user_id, occurred_at, payload) 
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	// Debug logging
	log.Printf("SaveRawEvent: EventType=%s, ProjectID='%s', TaskID='%s', UserID='%s', DepartmentID='%s', ActorUserID='%s'",
		event.EventType, event.ProjectID, event.TaskID, event.UserID, event.DepartmentID, event.ActorUserID)

	_, err := r.db.Exec(query,
		event.EventType,
		nullIfEmpty(event.ProjectID),
		nullIfEmpty(event.TaskID),
		nullIfEmpty(event.UserID),
		nullIfEmpty(event.DepartmentID),
		nullIfEmpty(event.ActorUserID),
		event.OccurredAt, event.Payload,
	)
	return err
}

func nullIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func (r *Repository) UpsertDepartmentMetric(departmentID, date string, field string, value int) error {
	query := fmt.Sprintf(`INSERT INTO department_metrics (department_id, date, %s) VALUES ($1, $2, $3)
	          ON CONFLICT (department_id, date) 
	          DO UPDATE SET %s = department_metrics.%s + $3, updated_at = CURRENT_TIMESTAMP`, field, field, field)
	_, err := r.db.Exec(query, departmentID, date, value)
	return err
}

func (r *Repository) GetTaskCreatedAt(taskID string) (time.Time, error) {
	var createdAt time.Time

	// First try to get from task_analytics
	query := `SELECT created_at FROM task_analytics 
	          WHERE task_id = $1 LIMIT 1`
	err := r.db.QueryRow(query, taskID).Scan(&createdAt)
	if err == nil {
		log.Printf("GetTaskCreatedAt: Found in task_analytics, taskID=%s, createdAt=%s", taskID, createdAt.Format(time.RFC3339))
		return createdAt, nil
	}

	log.Printf("GetTaskCreatedAt: Not found in task_analytics, taskID=%s, error=%v", taskID, err)

	// Fallback to events_raw if not found in task_analytics
	query = `SELECT occurred_at FROM events_raw 
	          WHERE task_id = $1 AND event_type = 'task.created' ORDER BY occurred_at ASC LIMIT 1`
	err = r.db.QueryRow(query, taskID).Scan(&createdAt)
	if err == nil {
		log.Printf("GetTaskCreatedAt: Found in events_raw, taskID=%s, createdAt=%s", taskID, createdAt.Format(time.RFC3339))
	} else {
		log.Printf("GetTaskCreatedAt: Not found in events_raw either, taskID=%s, error=%v", taskID, err)
	}

	return createdAt, err
}

func (r *Repository) GetDashboardStats() (activeProjects, totalTasks, overdueTasks int, completionRate, onTimeRate float64, err error) {
	// Get active projects from project timeline control
	err = r.db.QueryRow(`SELECT COUNT(*) FROM project_timeline_control WHERE end_date IS NULL OR end_date > CURRENT_DATE`).Scan(&activeProjects)
	if err != nil {
		return
	}

	// Get total tasks from task analytics
	err = r.db.QueryRow(`SELECT COUNT(*) FROM task_analytics`).Scan(&totalTasks)
	if err != nil {
		return
	}

	// Get overdue tasks
	err = r.db.QueryRow(`SELECT COUNT(*) FROM task_analytics WHERE is_overdue = TRUE`).Scan(&overdueTasks)
	if err != nil {
		return
	}

	// Calculate completion rate
	if totalTasks > 0 {
		var completed int
		err = r.db.QueryRow(`SELECT COUNT(*) FROM task_analytics WHERE completed_at IS NOT NULL`).Scan(&completed)
		if err == nil {
			completionRate = float64(completed) / float64(totalTasks) * 100
		}
	}

	// Calculate on-time rate
	err = r.db.QueryRow(`
        SELECT COALESCE(AVG(on_time_rate), 0) FROM department_metrics 
        WHERE date = CURRENT_DATE
    `).Scan(&onTimeRate)

	return
}

func (r *Repository) GetDepartmentWorkload(departmentID string, days int) ([]*analyticsv1.DepartmentWorkload, error) {
	var rows *sql.Rows
	var err error

	if departmentID != "" {
		rows, err = r.db.Query(`
            SELECT 
                dm.department_id,
                dm.department_name,
                COALESCE(dm.wip_count, 0) as wip,
                COALESCE(dm.completed_tasks, 0) as completed,
                COALESCE(dm.overdue_tasks, 0) as overdue,
                COALESCE(dm.avg_cycle_time, 0) as avg_cycle_time,
                COALESCE(dm.productivity_rate, 0) as productivity,
                COALESCE(dm.on_time_rate, 0) as on_time_rate
            FROM department_metrics dm
            WHERE dm.department_id = $1 AND dm.date >= CURRENT_DATE - INTERVAL '1 day' * $2
            ORDER BY dm.date DESC
            LIMIT 1
        `, departmentID, days)
	} else {
		rows, err = r.db.Query(`
            SELECT DISTINCT ON (dm.department_id)
                dm.department_id,
                dm.department_name,
                COALESCE(dm.wip_count, 0) as wip,
                COALESCE(dm.completed_tasks, 0) as completed,
                COALESCE(dm.overdue_tasks, 0) as overdue,
                COALESCE(dm.avg_cycle_time, 0) as avg_cycle_time,
                COALESCE(dm.productivity_rate, 0) as productivity,
                COALESCE(dm.on_time_rate, 0) as on_time_rate
            FROM department_metrics dm
            WHERE dm.date >= CURRENT_DATE - INTERVAL '1 day' * $1
            ORDER BY dm.department_id, dm.date DESC
        `, days)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var workloads []*analyticsv1.DepartmentWorkload
	for rows.Next() {
		w := &analyticsv1.DepartmentWorkload{}
		err := rows.Scan(&w.DepartmentId, &w.DepartmentName, &w.Wip, &w.Completed, &w.Overdue, &w.AvgCycleTime, &w.Productivity, &w.OnTimeRate)
		if err != nil {
			return nil, err
		}
		workloads = append(workloads, w)
	}

	return workloads, nil
}

func (r *Repository) GetTaskTrends(departmentID string, weeks int) ([]*analyticsv1.WeeklyTrend, error) {
	var rows *sql.Rows
	var err error

	if departmentID != "" {
		rows, err = r.db.Query(`
            SELECT 
                wt.week,
                COALESCE(wt.tasks_created, 0) as created,
                COALESCE(wt.tasks_completed, 0) as completed,
                COALESCE(wt.tasks_overdue, 0) as overdue,
                COALESCE(wt.completion_rate, 0) as completion_rate,
                COALESCE(wt.on_time_rate, 0) as on_time_rate
            FROM weekly_trends wt
            WHERE wt.department_id = $1 AND wt.week >= CURRENT_DATE - INTERVAL '1 week' * $2
            ORDER BY wt.week DESC
        `, departmentID, weeks)
	} else {
		rows, err = r.db.Query(`
            SELECT 
                wt.week,
                COALESCE(SUM(wt.tasks_created), 0) as created,
                COALESCE(SUM(wt.tasks_completed), 0) as completed,
                COALESCE(SUM(wt.tasks_overdue), 0) as overdue,
                COALESCE(AVG(wt.completion_rate), 0) as completion_rate,
                COALESCE(AVG(wt.on_time_rate), 0) as on_time_rate
            FROM weekly_trends wt
            WHERE wt.week >= CURRENT_DATE - INTERVAL '1 week' * $1
            GROUP BY wt.week
            ORDER BY wt.week DESC
        `, weeks)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trends []*analyticsv1.WeeklyTrend
	for rows.Next() {
		t := &analyticsv1.WeeklyTrend{}
		var week time.Time
		err := rows.Scan(&week, &t.Created, &t.Completed, &t.Overdue, &t.CompletionRate, &t.OnTimeRate)
		if err != nil {
			return nil, err
		}
		t.Week = week.Format("2006-01-02")
		trends = append(trends, t)
	}

	return trends, nil
}

func (r *Repository) UpsertTaskAnalytics(taskID, projectID, departmentID, assignedUserID, createdBy string, createdAt time.Time, status int32, dueDate *time.Time) error {
	var deptID interface{} = nil
	if departmentID != "" {
		deptID = departmentID
	}

	var assigneeID interface{} = nil
	if assignedUserID != "" {
		assigneeID = assignedUserID
	}

	query := `INSERT INTO task_analytics (task_id, project_id, department_id, assigned_user_id, created_at, status, due_date, created_by) 
              VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
              ON CONFLICT (task_id) 
              DO UPDATE SET project_id = EXCLUDED.project_id, department_id = EXCLUDED.department_id, 
                         assigned_user_id = EXCLUDED.assigned_user_id, status = EXCLUDED.status, 
                         due_date = EXCLUDED.due_date, updated_at = CURRENT_TIMESTAMP`

	_, err := r.db.Exec(query, taskID, projectID, deptID, assigneeID, createdAt, status, dueDate, createdBy)
	return err
}

func (r *Repository) UpdateTaskCompletion(taskID string, completedAt time.Time, isOverdue bool, cycleTimeDays, delayedDays int) error {
	log.Printf("UpdateTaskCompletion: taskID=%s, completedAt=%s, isOverdue=%t, cycleTimeDays=%d, delayedDays=%d",
		taskID, completedAt.Format(time.RFC3339), isOverdue, cycleTimeDays, delayedDays)
	query := `UPDATE task_analytics 
	          SET completed_at = $1, is_overdue = $2, cycle_time_days = $3, delayed_days = $4, updated_at = CURRENT_TIMESTAMP
	          WHERE task_id = $5`
	result, err := r.db.Exec(query, completedAt, isOverdue, cycleTimeDays, delayedDays, taskID)
	if err != nil {
		log.Printf("UpdateTaskCompletion: Failed to update: %v", err)
		return err
	}
	rowsAffected, _ := result.RowsAffected()
	log.Printf("UpdateTaskCompletion: Successfully updated, rowsAffected=%d", rowsAffected)
	return nil
}

func (r *Repository) UpsertProjectTimelineControl(projectID, projectName, departmentID string, startDate, endDate *time.Time) error {
	query := `INSERT INTO project_timeline_control (project_id, project_name, department_id, start_date, end_date) 
	          VALUES ($1, $2, $3, $4, $5)
	          ON CONFLICT (project_id) 
	          DO UPDATE SET project_name = EXCLUDED.project_name, department_id = EXCLUDED.department_id,
	                     start_date = EXCLUDED.start_date, end_date = EXCLUDED.end_date, updated_at = CURRENT_TIMESTAMP`

	// Handle empty department_id as NULL
	var deptID interface{} = nullIfEmpty(departmentID)

	_, err := r.db.Exec(query, projectID, projectName, deptID, startDate, endDate)
	return err
}

func (r *Repository) UpdateProjectMetrics(projectID string, totalTasks, completedOnTime, overdueTasks int32, onTimeRate, avgDelayDays float64) error {
	query := `UPDATE project_timeline_control 
	          SET total_tasks = $1, completed_on_time = $2, overdue_tasks = $3, 
	              on_time_rate = $4, avg_delay_days = $5, updated_at = CURRENT_TIMESTAMP
	          WHERE project_id = $6`
	_, err := r.db.Exec(query, totalTasks, completedOnTime, overdueTasks, onTimeRate, avgDelayDays, projectID)
	return err
}

func (r *Repository) UpsertWeeklyTrends(week time.Time, departmentID string, tasksCreated, tasksCompleted, tasksOverdue int32, completionRate, onTimeRate float64) error {
	log.Printf("UpsertWeeklyTrends: week=%s, departmentID=%s, tasksCreated=%d, tasksCompleted=%d, tasksOverdue=%d, completionRate=%.2f, onTimeRate=%.2f",
		week.Format("2006-01-02"), departmentID, tasksCreated, tasksCompleted, tasksOverdue, completionRate, onTimeRate)
	query := `INSERT INTO weekly_trends (week, department_id, tasks_created, tasks_completed, tasks_overdue, completion_rate, on_time_rate) 
	          VALUES ($1, $2, $3, $4, $5, $6, $7)
	          ON CONFLICT (week, department_id) 
	          DO UPDATE SET tasks_created = weekly_trends.tasks_created + EXCLUDED.tasks_created, 
	                     tasks_completed = weekly_trends.tasks_completed + EXCLUDED.tasks_completed,
	                     tasks_overdue = weekly_trends.tasks_overdue + EXCLUDED.tasks_overdue, 
	                     completion_rate = EXCLUDED.completion_rate,
	                     on_time_rate = EXCLUDED.on_time_rate`
	result, err := r.db.Exec(query, week, departmentID, tasksCreated, tasksCompleted, tasksOverdue, completionRate, onTimeRate)
	if err != nil {
		log.Printf("UpsertWeeklyTrends: Failed to upsert: %v", err)
		return err
	}
	rowsAffected, _ := result.RowsAffected()
	log.Printf("UpsertWeeklyTrends: Successfully uperted, rowsAffected=%d", rowsAffected)
	return nil
}

func (r *Repository) UpsertEmployeeProductivity(userID, fullName, email, departmentID string, date time.Time, tasksCompleted, tasksOverdue int32, avgCycleTime, completionRate, onTimeRate float64) error {
	query := `INSERT INTO employee_productivity (user_id, full_name, email, department_id, date, tasks_completed, tasks_overdue, avg_cycle_time, completion_rate, on_time_rate) 
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	          ON CONFLICT (user_id, date) 
	          DO UPDATE SET full_name = EXCLUDED.full_name, email = EXCLUDED.email, department_id = EXCLUDED.department_id,
	                     tasks_completed = EXCLUDED.tasks_completed, tasks_overdue = EXCLUDED.tasks_overdue,
	                     avg_cycle_time = EXCLUDED.avg_cycle_time, completion_rate = EXCLUDED.completion_rate,
	                     on_time_rate = EXCLUDED.on_time_rate, updated_at = CURRENT_TIMESTAMP`
	_, err := r.db.Exec(query, userID, fullName, email, departmentID, date, tasksCompleted, tasksOverdue, avgCycleTime, completionRate, onTimeRate)
	return err
}

func (r *Repository) GetProjectTimelineControl(projectID, departmentID string) ([]*analyticsv1.ProjectTimelineControl, error) {
	var rows *sql.Rows
	var err error

	// Базовый запрос с реальными данными
	query := `
        SELECT 
            p.project_id,
            p.project_name,
            COALESCE(t.total_tasks, 0) as total_tasks,
            COALESCE(t.completed_on_time, 0) as completed_on_time,
            COALESCE(t.overdue_tasks, 0) as overdue_tasks,
            COALESCE(t.on_time_rate, 0) as on_time_rate,
            COALESCE(t.avg_delay_days, 0) as avg_delay_days
        FROM project_timeline_control p
        LEFT JOIN (
            SELECT 
                project_id,
                COUNT(*) as total_tasks,
                COUNT(*) FILTER (WHERE completed_at <= due_date) as completed_on_time,
                COUNT(*) FILTER (WHERE is_overdue = true) as overdue_tasks,
                ROUND(100.0 * COUNT(*) FILTER (WHERE completed_at <= due_date) / NULLIF(COUNT(*), 0), 2) as on_time_rate,
                COALESCE(AVG(delayed_days), 0) as avg_delay_days
            FROM task_analytics
            GROUP BY project_id
        ) t ON p.project_id = t.project_id
        WHERE 1=1
    `

	args := []interface{}{}
	argIdx := 1

	if projectID != "" {
		query += fmt.Sprintf(" AND p.project_id = $%d", argIdx)
		args = append(args, projectID)
		argIdx++
	}

	if departmentID != "" {
		query += fmt.Sprintf(" AND p.department_id = $%d", argIdx)
		args = append(args, departmentID)
		argIdx++
	}

	rows, err = r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []*analyticsv1.ProjectTimelineControl
	for rows.Next() {
		p := &analyticsv1.ProjectTimelineControl{}
		err := rows.Scan(&p.ProjectId, &p.ProjectName, &p.TotalTasks, &p.CompletedOnTime, &p.OverdueTasks, &p.OnTimeRate, &p.AvgDelayDays)
		if err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}

	return projects, nil
}

func (r *Repository) GetEmployeeProductivity(departmentID, fromDate, toDate string) ([]*analyticsv1.EmployeeProductivity, error) {
	var rows *sql.Rows
	var err error

	if departmentID != "" && fromDate != "" && toDate != "" {
		// Get productivity for specific department within date range
		query := `
			SELECT user_id, full_name, email,
			       COALESCE(tasks_completed, 0) as tasks_completed,
			       COALESCE(tasks_overdue, 0) as tasks_overdue,
			       COALESCE(avg_cycle_time, 0) as avg_cycle_time,
			       COALESCE(completion_rate, 0) as completion_rate,
			       COALESCE(on_time_rate, 0) as on_time_rate
			FROM employee_productivity
			WHERE department_id = $1 AND date >= $2 AND date <= $3
			ORDER BY date DESC`
		rows, err = r.db.Query(query, departmentID, fromDate, toDate)
	} else if departmentID != "" {
		// Get productivity for specific department (last 30 days)
		query := `
			SELECT user_id, full_name, email,
			       COALESCE(tasks_completed, 0) as tasks_completed,
			       COALESCE(tasks_overdue, 0) as tasks_overdue,
			       COALESCE(avg_cycle_time, 0) as avg_cycle_time,
			       COALESCE(completion_rate, 0) as completion_rate,
			       COALESCE(on_time_rate, 0) as on_time_rate
			FROM employee_productivity
			WHERE department_id = $1 AND date >= CURRENT_DATE - INTERVAL '30 days'
			ORDER BY date DESC`
		rows, err = r.db.Query(query, departmentID)
	} else {
		// Get productivity for all departments (last 30 days)
		query := `
			SELECT user_id, full_name, email,
			       COALESCE(tasks_completed, 0) as tasks_completed,
			       COALESCE(tasks_overdue, 0) as tasks_overdue,
			       COALESCE(avg_cycle_time, 0) as avg_cycle_time,
			       COALESCE(completion_rate, 0) as completion_rate,
			       COALESCE(on_time_rate, 0) as on_time_rate
			FROM employee_productivity
			WHERE date >= CURRENT_DATE - INTERVAL '30 days'
			ORDER BY date DESC`
		rows, err = r.db.Query(query)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var employees []*analyticsv1.EmployeeProductivity
	for rows.Next() {
		var userID, fullName, email string
		var tasksCompleted, tasksOverdue int32
		var avgCycleTime, completionRate, onTimeRate float64

		err := rows.Scan(&userID, &fullName, &email, &tasksCompleted, &tasksOverdue, &avgCycleTime, &completionRate, &onTimeRate)
		if err != nil {
			return nil, err
		}

		e := &analyticsv1.EmployeeProductivity{
			UserId:         userID,
			FullName:       fullName,
			Email:          email,
			TasksCompleted: tasksCompleted,
			TasksOverdue:   tasksOverdue,
			AvgCycleTime:   avgCycleTime,
			CompletionRate: completionRate,
			OnTimeRate:     onTimeRate,
		}
		employees = append(employees, e)
	}

	return employees, nil
}

func (r *Repository) GetProjectStats(projectID string) (totalTasks, completedOnTime, overdueTasks int32, onTimeRate, avgDelayDays float64, err error) {
	query := `
        SELECT 
            COUNT(*) as total_tasks,
            COUNT(*) FILTER (WHERE completed_at <= due_date) as completed_on_time,
            COUNT(*) FILTER (WHERE is_overdue = true) as overdue_tasks,
            ROUND(100.0 * COUNT(*) FILTER (WHERE completed_at <= due_date) / NULLIF(COUNT(*), 0), 2) as on_time_rate,
            COALESCE(AVG(delayed_days), 0) as avg_delay_days
        FROM task_analytics
        WHERE project_id = $1 AND completed_at IS NOT NULL
    `
	err = r.db.QueryRow(query, projectID).Scan(&totalTasks, &completedOnTime, &overdueTasks, &onTimeRate, &avgDelayDays)
	return
}
