package user

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

type Repository interface {
	FindByID(ctx context.Context, id string) (*User, error)
	FindByEmail(ctx context.Context, email string) (*User, error)
	Find(ctx context.Context, query string, limit int) ([]*User, error)
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

func (r *postgresRepository) FindByEmail(ctx context.Context, email string) (*User, error) {
	query := `SELECT id, full_name, email, role, department_id
              FROM users WHERE email = $1`

	var user User
	var deptID sql.NullString

	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID, &user.FullName, &user.Email, &user.Role,
		&deptID,
	)
	if err != nil {
		return nil, err
	}

	if deptID.Valid {
		user.DepartmentID = &deptID.String
	}

	return &user, nil
}

func (r *postgresRepository) Find(ctx context.Context, query string, limit int) ([]*User, error) {
	q := strings.TrimSpace(query)
	if limit <= 0 || limit > 100 {
		limit = 100
	}

	var rows *sql.Rows
	var err error
	if q == "" {
		rows, err = r.db.QueryContext(ctx, `
			SELECT id, full_name, email, role, department_id
			FROM users
			ORDER BY full_name, email
			LIMIT $1`, limit)
	} else {
		pattern := "%" + q + "%"
		rows, err = r.db.QueryContext(ctx, `
			SELECT id, full_name, email, role, department_id
			FROM users
			WHERE full_name ILIKE $1 OR email ILIKE $1
			ORDER BY full_name, email
			LIMIT $2`, pattern, limit)
	}
	if err != nil {
		return nil, fmt.Errorf("find users: %w", err)
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		var u User
		var deptID sql.NullString
		if err := rows.Scan(&u.ID, &u.FullName, &u.Email, &u.Role, &deptID); err != nil {
			return nil, err
		}
		if deptID.Valid {
			u.DepartmentID = &deptID.String
		}
		users = append(users, &u)
	}
	return users, rows.Err()
}
