package user

import (
	"context"
	"database/sql"
	"fmt"
)

type Repository interface {
	FindByID(ctx context.Context, id string) (*User, error)
	Upsert(ctx context.Context, user *User) error
}

type postgresRepository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *postgresRepository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) FindByID(ctx context.Context, id string) (*User, error) {
	query := `SELECT id, full_name, email, role, department_id 
              FROM users WHERE id = $1`

	var user User

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID, &user.FullName, &user.Email, &user.Role, &user.DepartmentID,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found: %w", err)
		}
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	return &user, nil
}

func (r *postgresRepository) Upsert(ctx context.Context, user *User) error {
	query := ` INSERT INTO users (id, full_name, email, role, department_id, updated_at)
			VALUES ($1, $2, $3, $4, $5, CURRENT_TIMESTAMP)
			ON CONFLICT (id) DO UPDATE SET
				full_name = EXCLUDED.full_name,
				email = EXCLUDED.email,
				role = EXCLUDED.role,
				department_id = EXCLUDED.department_id,
				updated_at = CURRENT_TIMESTAMP`

	var deptID interface{} = nil

	if user.DepartmentID != nil && *user.DepartmentID != "" {
		deptID = *user.DepartmentID
	}

	_, err := r.db.ExecContext(ctx, query,
		user.ID,
		user.FullName,
		user.Email,
		user.Role,
		deptID)
	return err
}
