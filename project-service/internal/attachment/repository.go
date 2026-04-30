package attachment

import (
	"context"
	"database/sql"
	"time"

	projectv1 "building-services/gen/project/v1"

	"google.golang.org/protobuf/types/known/timestamppb"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, att *projectv1.Attachment) error {
	query := `INSERT INTO attachments (id, task_id, file_url, type, file_name, file_size, uploaded_by, uploaded_at, description)
              VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	_, err := r.db.ExecContext(ctx, query,
		att.Id, att.TaskId, att.FileUrl, att.Type, att.FileName,
		att.FileSize, att.UploadedBy, att.UploadedAt.AsTime(), att.Description,
	)
	return err
}

func (r *Repository) FindByID(ctx context.Context, id string) (*projectv1.Attachment, error) {
	query := `SELECT id, task_id, file_url, type, file_name, file_size, uploaded_by, uploaded_at, description
              FROM attachments WHERE id = $1`

	var att projectv1.Attachment
	var uploadedAt time.Time

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&att.Id, &att.TaskId, &att.FileUrl, &att.Type, &att.FileName,
		&att.FileSize, &att.UploadedBy, &uploadedAt, &att.Description,
	)
	if err != nil {
		return nil, err
	}

	att.UploadedAt = timestamppb.New(uploadedAt)
	return &att, nil
}

func (r *Repository) ListByTask(ctx context.Context, taskID string) ([]*projectv1.Attachment, error) {
	query := `SELECT id, task_id, file_url, type, file_name, file_size, uploaded_by, uploaded_at, description
              FROM attachments WHERE task_id = $1 ORDER BY uploaded_at DESC`

	rows, err := r.db.QueryContext(ctx, query, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var attachments []*projectv1.Attachment
	for rows.Next() {
		var att projectv1.Attachment
		var uploadedAt time.Time

		err := rows.Scan(
			&att.Id, &att.TaskId, &att.FileUrl, &att.Type, &att.FileName,
			&att.FileSize, &att.UploadedBy, &uploadedAt, &att.Description,
		)
		if err != nil {
			return nil, err
		}

		att.UploadedAt = timestamppb.New(uploadedAt)
		attachments = append(attachments, &att)
	}

	return attachments, nil
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM attachments WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *Repository) GetTaskID(ctx context.Context, attachmentID string) (string, error) {
	query := `SELECT task_id FROM attachments WHERE id = $1`
	var taskID string
	err := r.db.QueryRowContext(ctx, query, attachmentID).Scan(&taskID)
	return taskID, err
}

func (r *Repository) GetUploadedBy(ctx context.Context, attachmentID string) (string, error) {
	query := `SELECT uploaded_by FROM attachments WHERE id = $1`
	var userID string
	err := r.db.QueryRowContext(ctx, query, attachmentID).Scan(&userID)
	return userID, err
}
