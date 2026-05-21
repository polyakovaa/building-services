package activity

import (
	projectv1 "building-services/gen/project/v1"
	"context"
	"database/sql"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) List(ctx context.Context) ([]*projectv1.ActivityType, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name, sort_order
		FROM activity_types
		ORDER BY sort_order, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*projectv1.ActivityType
	for rows.Next() {
		var a projectv1.ActivityType
		if err := rows.Scan(&a.Id, &a.Name, &a.SortOrder); err != nil {
			return nil, err
		}
		out = append(out, &a)
	}
	return out, rows.Err()
}

func (r *Repository) Create(ctx context.Context, at *projectv1.ActivityType) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO activity_types (id, name, sort_order) VALUES ($1, $2, $3)`,
		at.Id, at.Name, at.SortOrder)
	return err
}

func (r *Repository) NextSortOrder(ctx context.Context) (int32, error) {
	var next int32
	err := r.db.QueryRowContext(ctx, `SELECT COALESCE(MAX(sort_order), 0) + 1 FROM activity_types`).Scan(&next)
	return next, err
}

func (r *Repository) Exists(ctx context.Context, id string) (bool, error) {
	if id == "" {
		return false, nil
	}
	var exists bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM activity_types WHERE id = $1)`, id).Scan(&exists)
	return exists, err
}
