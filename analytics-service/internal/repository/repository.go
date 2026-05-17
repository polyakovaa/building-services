package repository

import (
	"database/sql"
	"time"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

type User struct {
	ID           string
	Email        string
	FullName     string
	Role         string
	DepartmentID string
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
	const q = `INSERT INTO events_raw (event_type, project_id, task_id, user_id, department_id, actor_user_id, occurred_at, payload)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err := r.db.Exec(q,
		event.EventType,
		nullIfEmpty(event.ProjectID),
		nullIfEmpty(event.TaskID),
		nullIfEmpty(event.UserID),
		nullIfEmpty(event.DepartmentID),
		nullIfEmpty(event.ActorUserID),
		event.OccurredAt,
		event.Payload,
	)
	return err
}

func (r *Repository) UpsertUser(user User) error {
	var dept interface{}
	if user.DepartmentID != "" {
		dept = user.DepartmentID
	}
	const q = `INSERT INTO users (id, email, full_name, role, department_id, updated_at)
		VALUES ($1, NULLIF($2, ''), NULLIF($3, ''), NULLIF($4, ''), $5, CURRENT_TIMESTAMP)
		ON CONFLICT (id) DO UPDATE SET
			email = COALESCE(NULLIF(EXCLUDED.email, ''), users.email),
			full_name = COALESCE(NULLIF(EXCLUDED.full_name, ''), users.full_name),
			role = COALESCE(NULLIF(EXCLUDED.role, ''), users.role),
			department_id = COALESCE(EXCLUDED.department_id, users.department_id),
			updated_at = CURRENT_TIMESTAMP`
	_, err := r.db.Exec(q, user.ID, user.Email, user.FullName, user.Role, dept)
	return err
}

func (r *Repository) UpdateTaskDepartmentsForUser(userID, departmentID string) error {
	if userID == "" || departmentID == "" {
		return nil
	}
	_, err := r.db.Exec(
		`UPDATE task_analytics SET department_id = $1, updated_at = CURRENT_TIMESTAMP WHERE assigned_user_id = $2`,
		departmentID, userID,
	)
	return err
}

func (r *Repository) GetTaskCreatedAt(taskID string) (time.Time, error) {
	var createdAt time.Time
	err := r.db.QueryRow(`SELECT created_at FROM task_analytics WHERE task_id = $1 LIMIT 1`, taskID).Scan(&createdAt)
	if err == nil {
		return createdAt, nil
	}
	err = r.db.QueryRow(
		`SELECT occurred_at FROM events_raw WHERE task_id = $1 AND event_type = 'task.created' ORDER BY occurred_at ASC LIMIT 1`,
		taskID,
	).Scan(&createdAt)
	return createdAt, err
}

func (r *Repository) UpsertTaskAnalytics(taskID, projectID, departmentID, assignedUserID, createdBy string, createdAt time.Time, status int32, dueDate *time.Time) error {
	var deptID interface{}
	if departmentID != "" {
		deptID = departmentID
	}
	var assigneeID interface{}
	if assignedUserID != "" {
		assigneeID = assignedUserID
	}
	var createdByVal interface{}
	if createdBy != "" {
		createdByVal = createdBy
	}
	var assignAt interface{}
	if assignedUserID != "" {
		assignAt = createdAt
	}
	const q = `INSERT INTO task_analytics (task_id, project_id, department_id, assigned_user_id, created_at, assigned_at, status, due_date, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (task_id) DO UPDATE SET
			project_id = EXCLUDED.project_id,
			department_id = CASE
				WHEN EXCLUDED.department_id IS NOT NULL THEN EXCLUDED.department_id
				ELSE task_analytics.department_id
			END,
			assigned_user_id = COALESCE(EXCLUDED.assigned_user_id, task_analytics.assigned_user_id),
			status = CASE WHEN EXCLUDED.status = 0 THEN task_analytics.status ELSE EXCLUDED.status END,
			due_date = COALESCE(EXCLUDED.due_date, task_analytics.due_date),
			assigned_at = CASE
				WHEN EXCLUDED.assigned_user_id IS NOT NULL AND (
					task_analytics.assigned_user_id IS NULL OR EXCLUDED.assigned_user_id IS DISTINCT FROM task_analytics.assigned_user_id)
				THEN COALESCE(EXCLUDED.assigned_at, EXCLUDED.created_at)
				ELSE task_analytics.assigned_at
			END,
			updated_at = CURRENT_TIMESTAMP`
	_, err := r.db.Exec(q, taskID, projectID, deptID, assigneeID, createdAt, assignAt, status, dueDate, createdByVal)
	return err
}

func (r *Repository) UpdateTaskCompletion(taskID string, completedAt time.Time, isOverdue bool, cycleTimeDays, delayedDays int) error {
	const q = `UPDATE task_analytics
		SET completed_at = $1, is_overdue = $2, cycle_time_days = $3, delayed_days = $4, updated_at = CURRENT_TIMESTAMP, status = 3
		WHERE task_id = $5`
	_, err := r.db.Exec(q, completedAt, isOverdue, cycleTimeDays, delayedDays, taskID)
	return err
}

func (r *Repository) UpsertProject(projectID, projectName string, startDate, endDate *time.Time) error {
	const q = `INSERT INTO projects (project_id, project_name, start_date, end_date)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (project_id) DO UPDATE SET
			project_name = EXCLUDED.project_name,
			start_date = COALESCE(EXCLUDED.start_date, projects.start_date),
			end_date = COALESCE(EXCLUDED.end_date, projects.end_date),
			updated_at = CURRENT_TIMESTAMP`
	_, err := r.db.Exec(q, projectID, projectName, startDate, endDate)
	return err
}

func nullIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
