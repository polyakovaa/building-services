package project

import (
	projectv1 "building-services/gen/project/v1"
	"context"
	"database/sql"
	"fmt"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

type ProjectFilter struct {
	Status    projectv1.ProjectStatus
	ManagerID string
	UserID    string
	UserRole  string
}

func (r *Repository) Exists(ctx context.Context, id string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM projects WHERE id = $1)`
	var exists bool
	err := r.db.QueryRowContext(ctx, query, id).Scan(&exists)
	return exists, err
}

func (r *Repository) Create(ctx context.Context, project *projectv1.Project) error {
	query := `INSERT INTO projects (name, description, object_address, customer, start_date, end_date, created_by, status) 
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`
	var startDate, endDate *time.Time
	if project.StartDate != nil {
		t := project.StartDate.AsTime()
		startDate = &t
	}
	if project.EndDate != nil {
		t := project.EndDate.AsTime()
		endDate = &t
	}

	var id string
	err := r.db.QueryRowContext(ctx, query,
		project.Name,
		project.Description,
		project.ObjectAddress,
		project.Customer,
		startDate,
		endDate,
		project.CreatedBy,
		project.Status,
	).Scan(&id)

	if err != nil {
		return fmt.Errorf("failed to create project: %w", err)
	}

	project.Id = id

	return nil
}

func (r *Repository) FindByID(ctx context.Context, id string) (*projectv1.Project, error) {
	query := `SELECT id, name, description, object_address, customer, 
              status, start_date, end_date, created_by, created_at, updated_at 
              FROM projects WHERE id = $1`

	p := &projectv1.Project{}
	var startDate, endDate, createdAt, updatedAt sql.NullTime
	var status int32

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&p.Id,
		&p.Name,
		&p.Description,
		&p.ObjectAddress,
		&p.Customer,
		&status,
		&startDate,
		&endDate,
		&p.CreatedBy,
		&createdAt,
		&updatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to find project: %w", err)
	}
	p.Status = projectv1.ProjectStatus(status)

	if startDate.Valid {
		p.StartDate = timestamppb.New(startDate.Time)
	}
	if endDate.Valid {
		p.EndDate = timestamppb.New(endDate.Time)
	}
	if createdAt.Valid {
		p.CreatedAt = timestamppb.New(createdAt.Time)
	}
	if updatedAt.Valid {
		p.UpdatedAt = timestamppb.New(updatedAt.Time)
	}

	return p, nil

}

func (r *Repository) Update(ctx context.Context, project *projectv1.Project) error {
	query := `UPDATE projects SET name = $1, description =$2,
	 object_address = $3, customer = $4, start_date = $5, end_date = $6, status = $7 WHERE id=$8`

	var startDate, endDate *time.Time
	if project.StartDate != nil {
		t := project.StartDate.AsTime()
		startDate = &t
	}
	if project.EndDate != nil {
		t := project.EndDate.AsTime()
		endDate = &t
	}

	result, err := r.db.ExecContext(ctx, query,
		project.Name,
		project.Description,
		project.ObjectAddress,
		project.Customer,
		startDate,
		endDate,
		project.Status,
		project.Id,
	)

	if err != nil {
		return fmt.Errorf("failed to update project: %w", err)
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
	query := `DELETE FROM projects WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete project: %w", err)
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

func (r *Repository) UpdateStatus(ctx context.Context, id string, status projectv1.ProjectStatus) error {
	query := `UPDATE projects SET status = $1 WHERE id = $2`

	result, err := r.db.ExecContext(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("failed to update status of project: %w", err)
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

func (r *Repository) List(ctx context.Context, filter *ProjectFilter) ([]*projectv1.Project, error) {
	query := `SELECT id, name, description, object_address, customer, 
	status, start_date, end_date, created_by 
	FROM projects WHERE 1=1`

	args := []interface{}{}
	argIdx := 1
	if filter.Status != projectv1.ProjectStatus_PROJECT_STATUS_UNSPECIFIED {
		query += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, filter.Status)
		argIdx++
	}
	if filter.ManagerID != "" {
		query += fmt.Sprintf(` AND id IN (
            SELECT project_id FROM project_members 
            WHERE user_id = $%d
        )`, argIdx)
		args = append(args, filter.ManagerID)
		argIdx++
	}

	switch filter.UserRole {
	case "ROLE_DIRECTOR", "ROLE_GIP":
	case "ROLE_DEPARTMENT_MANAGER":
		query += fmt.Sprintf(` AND id IN (
            SELECT project_id FROM project_members pm
            JOIN users u ON u.id = pm.user_id
            WHERE u.department_id = $%d
        )`, argIdx)
		args = append(args, filter.UserID)
		argIdx++
	default:
		query += fmt.Sprintf(` AND id IN (
            SELECT project_id FROM project_members 
            WHERE user_id = $%d
        )`, argIdx)
		args = append(args, filter.UserID)
		argIdx++
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}
	defer rows.Close()
	var projects []*projectv1.Project
	for rows.Next() {
		p := &projectv1.Project{}
		var startDate, endDate sql.NullTime
		var status int32

		err := rows.Scan(
			&p.Id, &p.Name, &p.Description, &p.ObjectAddress,
			&p.Customer, &status, &startDate, &endDate, &p.CreatedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan project: %w", err)
		}

		p.Status = projectv1.ProjectStatus(status)
		if startDate.Valid {
			p.StartDate = timestamppb.New(startDate.Time)
		}
		if endDate.Valid {
			p.EndDate = timestamppb.New(endDate.Time)
		}

		projects = append(projects, p)
	}

	return projects, rows.Err()
}
