package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	notificationv1 "building-services/gen/notification/v1"

	"google.golang.org/protobuf/types/known/timestamppb"
)

var ErrNotFound = errors.New("notification not found")

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

type RawEvent struct {
	EventType  string
	EventKey   string
	OccurredAt time.Time
	Payload    []byte
}

type Notification struct {
	ID              string
	RecipientUserID string
	Type            string
	Priority        int32
	Title           string
	Message         string
	ProjectID       string
	TaskID          string
	ActorUserID     string
	ActionURL       string
	SourceEventType string
	SourceEventKey  string
	Payload         json.RawMessage
	ReadAt          *time.Time
	CreatedAt       time.Time
}

type CreateNotificationParams struct {
	RecipientUserID string
	Type            string
	Priority        int32
	Title           string
	Message         string
	ProjectID       string
	TaskID          string
	ActorUserID     string
	ActionURL       string
	SourceEventType string
	SourceEventKey  string
	Payload         []byte
}

type NotificationTask struct {
	TaskID         string
	ProjectID      string
	AssigneeUserID string
	TaskTitle      string
	ProjectName    string
	Deadline       *time.Time
	Status         int32
	CompletedAt    *time.Time
}

func (r *Repository) SaveRawEvent(ctx context.Context, event RawEvent) (bool, error) {
	query := `INSERT INTO notification_events (event_type, event_key, occurred_at, payload)
		VALUES ($1, $2, $3, $4) ON CONFLICT (event_key) DO NOTHING`
	result, err := r.db.ExecContext(ctx, query, event.EventType, event.EventKey, event.OccurredAt, event.Payload)
	if err != nil {
		return false, err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return rows > 0, nil
}

func (r *Repository) MarkEventProcessed(ctx context.Context, eventKey string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE notification_events SET processed_at = CURRENT_TIMESTAMP WHERE event_key = $1`, eventKey)
	return err
}

func (r *Repository) CreateNotification(ctx context.Context, params CreateNotificationParams) error {
	query := `INSERT INTO notifications (
			recipient_user_id, type, priority, title, message, project_id, task_id, actor_user_id,
			action_url, source_event_type, source_event_key, payload
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (recipient_user_id, source_event_key) DO NOTHING`
	_, err := r.db.ExecContext(ctx, query,
		params.RecipientUserID,
		params.Type,
		params.Priority,
		params.Title,
		params.Message,
		nullIfEmpty(params.ProjectID),
		nullIfEmpty(params.TaskID),
		nullIfEmpty(params.ActorUserID),
		params.ActionURL,
		params.SourceEventType,
		params.SourceEventKey,
		params.Payload,
	)
	return err
}

func (r *Repository) UpsertNotificationTask(ctx context.Context, task NotificationTask) error {
	query := `INSERT INTO notification_tasks (
			task_id, project_id, assignee_user_id, task_title, project_name, deadline, status, completed_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, CURRENT_TIMESTAMP)
		ON CONFLICT (task_id) DO UPDATE SET
			project_id = COALESCE(EXCLUDED.project_id, notification_tasks.project_id),
			assignee_user_id = COALESCE(EXCLUDED.assignee_user_id, notification_tasks.assignee_user_id),
			task_title = COALESCE(NULLIF(EXCLUDED.task_title, ''), notification_tasks.task_title),
			project_name = COALESCE(NULLIF(EXCLUDED.project_name, ''), notification_tasks.project_name),
			deadline = COALESCE(EXCLUDED.deadline, notification_tasks.deadline),
			status = EXCLUDED.status,
			completed_at = COALESCE(EXCLUDED.completed_at, notification_tasks.completed_at),
			updated_at = CURRENT_TIMESTAMP`
	_, err := r.db.ExecContext(ctx, query,
		task.TaskID,
		nullIfEmpty(task.ProjectID),
		nullIfEmpty(task.AssigneeUserID),
		task.TaskTitle,
		task.ProjectName,
		task.Deadline,
		task.Status,
		task.CompletedAt,
	)
	return err
}

func (r *Repository) UpdateProjectName(ctx context.Context, projectID, projectName string) error {
	if projectID == "" || projectName == "" {
		return nil
	}
	_, err := r.db.ExecContext(ctx,
		`UPDATE notification_tasks SET project_name = $2, updated_at = CURRENT_TIMESTAMP WHERE project_id = $1`,
		projectID, projectName)
	return err
}

func (r *Repository) UpdateTaskAssignee(ctx context.Context, taskID, assigneeUserID string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE notification_tasks SET assignee_user_id = $2, updated_at = CURRENT_TIMESTAMP WHERE task_id = $1`, taskID, nullIfEmpty(assigneeUserID))
	return err
}

func (r *Repository) UpdateTaskDeadline(ctx context.Context, taskID string, deadline *time.Time, assigneeUserID, taskTitle, projectName string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE notification_tasks
		SET deadline = $2,
		assignee_user_id = COALESCE($3, assignee_user_id),
		task_title = COALESCE(NULLIF($4, ''), task_title),
		project_name = COALESCE(NULLIF($5, ''), project_name),
		updated_at = CURRENT_TIMESTAMP
		WHERE task_id = $1`, taskID, deadline, nullIfEmpty(assigneeUserID), taskTitle, projectName)
	return err
}

func (r *Repository) UpdateTaskStatus(ctx context.Context, taskID string, status int32, completedAt *time.Time, taskTitle, projectName string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE notification_tasks
		SET status = $2,
		completed_at = COALESCE($3, completed_at),
		task_title = COALESCE(NULLIF($4, ''), task_title),
		project_name = COALESCE(NULLIF($5, ''), project_name),
		updated_at = CURRENT_TIMESTAMP
		WHERE task_id = $1`, taskID, status, completedAt, taskTitle, projectName)
	return err
}

func (r *Repository) ListUpcomingDeadlineTasks(ctx context.Context, now, until time.Time) ([]NotificationTask, error) {
	query := `SELECT task_id::text, COALESCE(project_id::text, ''), COALESCE(assignee_user_id::text, ''),
			COALESCE(task_title, ''), COALESCE(project_name, ''), deadline, status, completed_at
		FROM notification_tasks
		WHERE assignee_user_id IS NOT NULL
		AND deadline IS NOT NULL
		AND (deadline AT TIME ZONE 'UTC')::date >= ($1 AT TIME ZONE 'UTC')::date
		AND (deadline AT TIME ZONE 'UTC')::date <= ($2 AT TIME ZONE 'UTC')::date
		AND status <> 3`
	return r.listDeadlineTasks(ctx, query, now, until)
}

func (r *Repository) ListOverdueTasks(ctx context.Context, now time.Time) ([]NotificationTask, error) {
	query := `SELECT task_id::text, COALESCE(project_id::text, ''), COALESCE(assignee_user_id::text, ''),
			COALESCE(task_title, ''), COALESCE(project_name, ''), deadline, status, completed_at
		FROM notification_tasks
		WHERE assignee_user_id IS NOT NULL
		AND deadline IS NOT NULL
		AND (deadline AT TIME ZONE 'UTC')::date < ($1 AT TIME ZONE 'UTC')::date
		AND status <> 3`
	return r.listDeadlineTasks(ctx, query, now)
}

func (r *Repository) listDeadlineTasks(ctx context.Context, query string, args ...interface{}) ([]NotificationTask, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []NotificationTask
	for rows.Next() {
		var task NotificationTask
		var deadline sql.NullTime
		var completedAt sql.NullTime
		if err := rows.Scan(
			&task.TaskID,
			&task.ProjectID,
			&task.AssigneeUserID,
			&task.TaskTitle,
			&task.ProjectName,
			&deadline,
			&task.Status,
			&completedAt,
		); err != nil {
			return nil, err
		}
		if deadline.Valid {
			task.Deadline = &deadline.Time
		}
		if completedAt.Valid {
			task.CompletedAt = &completedAt.Time
		}
		tasks = append(tasks, task)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return tasks, nil
}

func (r *Repository) ListNotifications(ctx context.Context, userID string, pageSize int, pageToken string, unreadOnly bool) ([]Notification, string, error) {
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	args := []interface{}{userID}
	query := `SELECT id::text, recipient_user_id::text, type, priority, title, message,
			COALESCE(project_id::text, ''), COALESCE(task_id::text, ''), COALESCE(actor_user_id::text, ''),
			action_url, source_event_type, source_event_key, payload, read_at, created_at
		FROM notifications WHERE recipient_user_id = $1`
	argIdx := 2
	if unreadOnly {
		query += ` AND read_at IS NULL`
	}
	if pageToken != "" {
		query += fmt.Sprintf(" AND created_at < $%d", argIdx)
		args = append(args, pageToken)
		argIdx++
	}
	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d", argIdx)
	args = append(args, pageSize+1)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	notifications := make([]Notification, 0, pageSize)
	for rows.Next() {
		n, err := scanNotification(rows)
		if err != nil {
			return nil, "", err
		}
		notifications = append(notifications, n)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}

	nextToken := ""
	if len(notifications) > pageSize {
		nextToken = notifications[pageSize-1].CreatedAt.Format(time.RFC3339Nano)
		notifications = notifications[:pageSize]
	}
	return notifications, nextToken, nil
}

func (r *Repository) GetUnreadCount(ctx context.Context, userID string) (int32, error) {
	var count int32
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM notifications WHERE recipient_user_id = $1 AND read_at IS NULL`, userID).Scan(&count)
	return count, err
}

func (r *Repository) MarkAsRead(ctx context.Context, userID, notificationID string) (Notification, error) {
	query := `UPDATE notifications
		SET read_at = COALESCE(read_at, CURRENT_TIMESTAMP)
		WHERE id = $1 AND recipient_user_id = $2
		RETURNING id::text, recipient_user_id::text, type, priority, title, message,
			COALESCE(project_id::text, ''), COALESCE(task_id::text, ''), COALESCE(actor_user_id::text, ''),
			action_url, source_event_type, source_event_key, payload, read_at, created_at`
	row := r.db.QueryRowContext(ctx, query, notificationID, userID)
	n, err := scanNotification(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Notification{}, ErrNotFound
	}
	return n, err
}

func (r *Repository) MarkAllAsRead(ctx context.Context, userID string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE notifications SET read_at = COALESCE(read_at, CURRENT_TIMESTAMP) WHERE recipient_user_id = $1 AND read_at IS NULL`, userID)
	return err
}

func ToProto(n Notification) *notificationv1.Notification {
	out := &notificationv1.Notification{
		Id:              n.ID,
		RecipientUserId: n.RecipientUserID,
		Type:            n.Type,
		Priority:        notificationv1.NotificationPriority(n.Priority),
		Title:           n.Title,
		Message:         n.Message,
		ProjectId:       n.ProjectID,
		TaskId:          n.TaskID,
		ActorUserId:     n.ActorUserID,
		ActionUrl:       n.ActionURL,
		SourceEventType: n.SourceEventType,
		SourceEventKey:  n.SourceEventKey,
		CreatedAt:       timestamppb.New(n.CreatedAt),
	}
	if n.ReadAt != nil {
		out.ReadAt = timestamppb.New(*n.ReadAt)
	}
	return out
}
