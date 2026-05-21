package department

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

func (r *Repository) Create(ctx context.Context, dept *projectv1.Department) error {
	query := `INSERT INTO departments (id, name, head_user_id) VALUES ($1, $2, $3)`
	_, err := r.db.ExecContext(ctx, query, dept.Id, dept.Name, dept.HeadUserId)
	return err
}

func (r *Repository) FindByID(ctx context.Context, id string) (*projectv1.Department, error) {
	query := `SELECT id, name, head_user_id, created_at FROM departments WHERE id = $1`

	dept := &projectv1.Department{}
	var headUserID sql.NullString
	var createdAt time.Time

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&dept.Id, &dept.Name, &headUserID, &createdAt)
	if err != nil {
		return nil, err
	}

	if headUserID.Valid {
		dept.HeadUserId = headUserID.String
	}
	dept.CreatedAt = timestamppb.New(createdAt)

	return dept, nil
}

func (r *Repository) List(ctx context.Context) ([]*projectv1.Department, error) {
	query := `SELECT id, name, head_user_id, created_at FROM departments ORDER BY name`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var depts []*projectv1.Department
	for rows.Next() {
		dept := &projectv1.Department{}
		var headUserID sql.NullString
		var createdAt time.Time

		err := rows.Scan(&dept.Id, &dept.Name, &headUserID, &createdAt)
		if err != nil {
			return nil, err
		}

		if headUserID.Valid {
			dept.HeadUserId = headUserID.String
		}
		dept.CreatedAt = timestamppb.New(createdAt)

		depts = append(depts, dept)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return depts, nil
}

func (r *Repository) Update(ctx context.Context, dept *projectv1.Department) error {
	query := `UPDATE departments SET name = $1, head_user_id = $2 WHERE id = $3`
	result, err := r.db.ExecContext(ctx, query, dept.Name, dept.HeadUserId, dept.Id)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, "UPDATE users SET department_id = NULL WHERE department_id = $1", id)
	if err != nil {
		return err
	}

	result, err := r.db.ExecContext(ctx, "DELETE FROM departments WHERE id = $1", id)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *Repository) AssignUser(ctx context.Context, userID, departmentID string) error {
	query := `UPDATE users SET department_id = $1 WHERE id = $2`
	result, err := r.db.ExecContext(ctx, query, departmentID, userID)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *Repository) RemoveUser(ctx context.Context, userID string) error {
	query := `UPDATE users SET department_id = NULL WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, userID)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *Repository) GetDepartmentUsers(ctx context.Context, departmentID string) ([]*projectv1.User, error) {
	query := `SELECT id, full_name, email, role FROM users WHERE department_id = $1`

	rows, err := r.db.QueryContext(ctx, query, departmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*projectv1.User
	for rows.Next() {
		u := &projectv1.User{}
		err := rows.Scan(&u.Id, &u.FullName, &u.Email, &u.Role)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}

	return users, nil
}
