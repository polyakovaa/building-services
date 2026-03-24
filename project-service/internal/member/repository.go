package member

import (
	projectv1 "building-services/gen/project/v1"
	"context"
	"database/sql"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) IsProjectMember(ctx context.Context, projectID, userID string) (*projectv1.ProjectMember, error) {
	query := `SELECT FROM project_members 
        WHERE project_id = $1 AND user_id = $2`
	var member projectv1.ProjectMember
	err := r.db.QueryRowContext(ctx, query, projectID, userID).Scan(&member)
	return &member, err

}

func (r *Repository) Add(ctx context.Context, member *projectv1.ProjectMember) error {
	query := `INSERT INTO project_member(project_id, user_id, department_id, joined_at) VALUES($1, $2, $3,$4)`
	var joinedAt *time.Time
	if member.JoinedAt != nil {
		t := member.JoinedAt.AsTime()
		joinedAt = &t
	}

	_, err := r.db.ExecContext(ctx, query,
		member.UserId,
		member.DepartmentId,
		member.ProjectId,
		joinedAt,
	)

	return err
}

func (r *Repository) IsProjectInDepartment(ctx context.Context, projectID, departmentID string) (bool, error) {
	query := `SELECT EXISTS(
        SELECT 1 FROM project_members pm
        JOIN users u ON u.id= pm.user_id
        WHERE pm.project_id = $1 AND u.department_id = $2
    )`

	var exists bool
	err := r.db.QueryRowContext(ctx, query, projectID, departmentID).Scan(&exists)
	return exists, err
}

func (r *Repository) IsManagerOfProject(ctx context.Context, projectID, userID string) (bool, error) {
	query := `SELECT EXISTS(
        SELECT 1 FROM project_members
        WHERE project_id = $1 AND user_id = $2
    )`

	var exists bool
	err := r.db.QueryRowContext(ctx, query, projectID, userID).Scan(&exists)
	return exists, err
}

func (r *Repository) GetProjectMembers(ctx context.Context, projectID string) ([]*projectv1.ProjectMember, error) {
	query := `SELECT project_id, user_id, department_id, joined_at
	 FROM project_members WHERE project_id= $1`

	var members []*projectv1.ProjectMember

	rows, err := r.db.QueryContext(ctx, query, projectID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var member projectv1.ProjectMember
		var joinedAt sql.NullTime

		err := rows.Scan(
			&member.ProjectId,
			&member.UserId,
			&member.DepartmentId,
			&joinedAt,
		)
		if err != nil {
			return nil, err
		}

		if joinedAt.Valid {
			member.JoinedAt = timestamppb.New(joinedAt.Time)
		}

		members = append(members, &member)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return members, nil

}

func (r *Repository) FindByID(ctx context.Context, userID string) (*projectv1.ProjectMember, error) {
	query := `SELECT project_id, user_Id, department_id, joined_at
	FROM project_members WHERE user_id = $1`

	var member projectv1.ProjectMember
	var joinedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&member.ProjectId,
		&member.UserId,
		&member.DepartmentId,
		&joinedAt,
	)

	if err != nil {
		return nil, err
	}

	if joinedAt.Valid {
		member.JoinedAt = timestamppb.New(joinedAt.Time)
	}

	return &member, nil
}

func (r *Repository) Update(ctx context.Context, member *projectv1.ProjectMember) error {
	query := `UPDATE project_members 
	SET department_id = $1, joined_at = $2
	WHERE project_id = $3 AND user_id = $4`

	var joinedAt *time.Time
	if member.JoinedAt != nil {
		t := member.JoinedAt.AsTime()
		joinedAt = &t
	}

	result, err := r.db.ExecContext(ctx, query,
		member.DepartmentId,
		joinedAt,
		member.ProjectId,
		member.UserId,
	)
	if err != nil {
		return err
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

func (r *Repository) Remove(ctx context.Context, projectID, userID string) error {
	query := `DELETE FROM project_members WHERE project_id = $1 AND user_id = $2`

	result, err := r.db.ExecContext(ctx, query, projectID, userID)
	if err != nil {
		return err
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
