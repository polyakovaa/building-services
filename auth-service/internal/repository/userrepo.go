package repository

import (
	"building-services/auth-service/internal/model"
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrEmailAlreadyExists = errors.New("email already exists")
	ErrDB                 = errors.New("DB error")
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{
		db: db,
	}
}

func (r *UserRepository) CreateUser(u *model.User) (*model.User, error) {
	query := `INSERT INTO users (full_name, role, email, password_hash) VALUES ($1, $2, $3, $4) RETURNING id`

	if err := r.db.QueryRow(
		query,
		u.FullName,
		u.Role,
		u.Email,
		u.PasswordHash,
	).Scan(&u.ID); err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23505" {
			return nil, ErrEmailAlreadyExists
		}
		return nil, fmt.Errorf("%w: %v", ErrDB, err)

	}
	return u, nil
}

func (r *UserRepository) FindByID(id string) (*model.User, error) {
	u := &model.User{}
	query := `SELECT id, role, full_name, email, password_hash FROM users WHERE id = $1`
	if err := r.db.QueryRow(query, id).Scan(
		&u.ID,
		&u.Role,
		&u.FullName,
		&u.Email,
		&u.PasswordHash,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("%w: %v", ErrDB, err)
	}
	return u, nil
}

func (r *UserRepository) FindAll(ctx context.Context) ([]*model.User, error) {
	query := `SELECT id, role, full_name, email FROM users`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to find users: %w", err)
	}
	defer rows.Close()
	var users []*model.User

	for rows.Next() {
		u := &model.User{}
		err := rows.Scan(&u.ID, &u.Role, &u.FullName, &u.Email)
		if err != nil {
			return nil, fmt.Errorf("failed to scan users: %w", err)
		}
		users = append(users, u)

	}
	return users, rows.Err()

}

func (r *UserRepository) UpdateRole(ctx context.Context, userID string, newRole string) error {
	query := `UPDATE users SET role = $1 WHERE id = $2`

	result, err := r.db.ExecContext(ctx, query, newRole, userID)
	if err != nil {
		return fmt.Errorf("failed to update role of user: %w", err)
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

func (r *UserRepository) FindByEmail(email string) (*model.User, error) {
	u := &model.User{}
	query := `SELECT id, role, full_name, email, password_hash FROM users WHERE email = $1`
	if err := r.db.QueryRow(query, email).Scan(
		&u.ID,
		&u.Role,
		&u.FullName,
		&u.Email,
		&u.PasswordHash,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("%w: %v", ErrDB, err)
	}
	return u, nil
}

func (r *UserRepository) UpdateProfile(ctx context.Context, userID string, fullName string, email string) (*model.User, error) {
	u := &model.User{}
	query := `
		UPDATE users
		SET
			full_name = COALESCE(NULLIF($1, ''), full_name),
			email = COALESCE(NULLIF($2, ''), email)
		WHERE id = $3
		RETURNING id, role, full_name, email, password_hash, created_at
	`

	if err := r.db.QueryRowContext(ctx, query, fullName, email, userID).Scan(
		&u.ID,
		&u.Role,
		&u.FullName,
		&u.Email,
		&u.PasswordHash,
		&u.CreatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrUserNotFound
		}
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23505" {
			return nil, ErrEmailAlreadyExists
		}
		return nil, fmt.Errorf("%w: %v", ErrDB, err)
	}

	return u, nil
}
