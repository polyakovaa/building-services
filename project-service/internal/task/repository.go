package task

import (
	projectv1 "building-services/gen/project/v1"
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"

)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

type TaskFilter struct {
	ProjectID    string
	Priority     *projectv1.TaskPriority
	Status       *projectv1.TaskStatus
	AssignedTo   *string
	ParentTaskID *string
}

func (r *Repository) Create(ctx context.Context, task *projectv1.Task) error {
	query := `INSERT INTO tasks (project_id, title, description, status, priority, deadline, assigned_to, created_by, parent_task_id, activity_type_id, planned_hours, actual_hours)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12) RETURNING id`
	var deadline *time.Time
	if task.Deadline != nil {
		t := task.Deadline.AsTime()
		deadline = &t
	}
	var parentTaskID interface{} = nil
	if task.ParentTaskId != "" {
		parentTaskID = task.ParentTaskId
	}

	var assignedTo interface{} = nil
	if task.AssignedTo != "" {
		assignedTo = task.AssignedTo
	}

	activityTypeID, plannedHours, actualHours := laborArgs(task)
	var id string
	err := r.db.QueryRowContext(ctx, query,
		task.ProjectId,
		task.Title,
		task.Description,
		task.Status,
		task.Priority,
		deadline,
		assignedTo,
		task.CreatedBy,
		parentTaskID,
		activityTypeID,
		plannedHours,
		actualHours).Scan(&id)

	if err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}
	task.Id = id
	return nil

}

func (r *Repository) FindByID(ctx context.Context, id string) (*projectv1.Task, error) {
	query := `SELECT ` + taskSelectColumns + ` FROM tasks WHERE id = $1`

	t := &projectv1.Task{}
	var deadline, createdAt, updatedAt sql.NullTime
	var status, priority int32
	var assignedTo, createdBy, parentTaskId, activityTypeID sql.NullString
	var plannedHours, actualHours sql.NullFloat64

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&t.Id,
		&t.ProjectId,
		&t.Title,
		&t.Description,
		&status,
		&priority,
		&deadline,
		&assignedTo,
		&createdBy,
		&parentTaskId,
		&createdAt,
		&updatedAt,
		&activityTypeID,
		&plannedHours,
		&actualHours,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to find task: %w", err)
	}
	scanTask(t, status, priority, deadline, createdAt, updatedAt, assignedTo, createdBy, parentTaskId, activityTypeID, plannedHours, actualHours)
	return t, nil

}

func (r *Repository) Update(ctx context.Context, task *projectv1.Task) error {
	query := `UPDATE tasks SET 
        title = $1,
        description = $2,
        priority = $3,
        deadline = $4,
        assigned_to = $5,
        parent_task_id = $6,
        activity_type_id = $7,
        planned_hours = $8,
        updated_at = CURRENT_TIMESTAMP
        WHERE id = $9`

	var deadline *time.Time
	if task.Deadline != nil {
		t := task.Deadline.AsTime()
		deadline = &t
	}
	var assignedTo interface{} = nil
	if task.AssignedTo != "" {
		assignedTo = task.AssignedTo
	}
	var parentTaskID interface{} = nil
	if task.ParentTaskId != "" {
		parentTaskID = task.ParentTaskId
	}

	activityTypeID, plannedHours, _ := laborArgs(task)
	result, err := r.db.ExecContext(ctx, query,
		task.Title,
		task.Description,
		task.Priority,
		deadline,
		assignedTo,
		parentTaskID,
		activityTypeID,
		plannedHours,
		task.Id,
	)

	if err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM tasks WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}

	return nil

}

func (r *Repository) UpdateLabor(ctx context.Context, id string, activityTypeID string, plannedHours, actualHours float64) error {
	activityArg, plannedArg, actualArg := laborArgs(&projectv1.Task{
		ActivityTypeId: activityTypeID,
		PlannedHours:   plannedHours,
		ActualHours:    actualHours,
	})
	result, err := r.db.ExecContext(ctx, `UPDATE tasks SET
		activity_type_id = $1,
		planned_hours = $2,
		actual_hours = $3,
		updated_at = CURRENT_TIMESTAMP
		WHERE id = $4`, activityArg, plannedArg, actualArg, id)
	if err != nil {
		return fmt.Errorf("failed to update task labor: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *Repository) UpdateStatus(ctx context.Context, id string, status projectv1.TaskStatus, actualHours float64) error {
	var result sql.Result
	var err error
	if actualHours > 0 {
		result, err = r.db.ExecContext(ctx,
			`UPDATE tasks SET status = $1, actual_hours = $2, updated_at = CURRENT_TIMESTAMP WHERE id = $3`,
			status, actualHours, id)
	} else {
		result, err = r.db.ExecContext(ctx,
			`UPDATE tasks SET status = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2`,
			status, id)
	}
	if err != nil {
		return fmt.Errorf("failed to update status of task: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func (r *Repository) List(ctx context.Context, filter *TaskFilter) ([]*projectv1.Task, error) {
	query := `SELECT ` + taskSelectColumns + ` FROM tasks WHERE 1=1`

	args := []interface{}{}
	argIdx := 1
	if filter.ProjectID != "" {
		query += fmt.Sprintf(" AND project_id = $%d", argIdx)
		args = append(args, filter.ProjectID)
		argIdx++
	}

	if filter.Status != nil {
		query += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, *filter.Status)
		argIdx++
	}
	if filter.AssignedTo != nil && *filter.AssignedTo != "" {
		query += fmt.Sprintf(" AND assigned_to = $%d", argIdx)
		args = append(args, *filter.AssignedTo)
		argIdx++
	}

	if filter.ParentTaskID != nil {
		if *filter.ParentTaskID == "" {
			query += " AND (parent_task_id IS NULL OR parent_task_id = '')"
		} else {
			query += " AND parent_task_id = $" + strconv.Itoa(argIdx)
			args = append(args, *filter.ParentTaskID)
			argIdx++
		}
	}
	if filter.Priority != nil {
		query += fmt.Sprintf(" AND priority = $%d", argIdx)
		args = append(args, *filter.Priority)
		argIdx++
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}
	defer rows.Close()

	var tasks []*projectv1.Task

	for rows.Next() {
		t := &projectv1.Task{}
		var deadline, createdAt, updatedAt sql.NullTime
		var status, priority int32
		var assignedTo, createdBy, parentTaskId, activityTypeID sql.NullString
		var plannedHours, actualHours sql.NullFloat64

		err := rows.Scan(&t.Id, &t.ProjectId, &t.Title, &t.Description,
			&status, &priority, &deadline, &assignedTo, &createdBy,
			&parentTaskId, &createdAt, &updatedAt, &activityTypeID, &plannedHours, &actualHours)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task: %w", err)
		}
		scanTask(t, status, priority, deadline, createdAt, updatedAt, assignedTo, createdBy, parentTaskId, activityTypeID, plannedHours, actualHours)
		tasks = append(tasks, t)
	}

	return tasks, rows.Err()

}

func (r *Repository) Assign(ctx context.Context, id string, assignedId string) (*projectv1.Task, error) {
	query := `UPDATE tasks SET assigned_to = $1 WHERE id = $2`
	var assignee interface{} = nil
	if assignedId != "" {
		assignee = assignedId
	}

	res, err := r.db.ExecContext(ctx, query, assignee, id)
	if err != nil {
		return nil, fmt.Errorf("failed to assign: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}
	if rows == 0 {
		return nil, sql.ErrNoRows
	}
	return r.FindByID(ctx, id)
}

func (r *Repository) GetProjectID(ctx context.Context, taskID string) (string, error) {
	query := `SELECT project_id FROM tasks WHERE id = $1`
	var projectID string
	err := r.db.QueryRowContext(ctx, query, taskID).Scan(&projectID)
	return projectID, err
}

func (r *Repository) IsAssignee(ctx context.Context, taskID, userID string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM tasks WHERE id = $1 AND assigned_to = $2)`
	var exists bool
	err := r.db.QueryRowContext(ctx, query, taskID, userID).Scan(&exists)
	return exists, err
}
