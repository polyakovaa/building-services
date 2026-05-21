package member

import (
	projectv1 "building-services/gen/project/v1"
	"context"
	"database/sql"
	"log"
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
	log.Printf("[DEBUG] IsProjectMember: checking user %s in project %s", userID, projectID)
	query := `SELECT project_id, user_id, department_id, joined_at FROM project_members WHERE project_id = $1 AND user_id = $2`
	var member projectv1.ProjectMember
	var joinedAt time.Time
	var deptID sql.NullString
	err := r.db.QueryRowContext(ctx, query, projectID, userID).Scan(
		&member.ProjectId,
		&member.UserId,
		&deptID,
		&joinedAt,
	)
	if deptID.Valid {
		member.DepartmentId = deptID.String
	}
	if err != nil {
		log.Printf("[DEBUG] IsProjectMember: query failed for user %s in project %s: %v", userID, projectID, err)
		return nil, err
	}

	member.JoinedAt = timestamppb.New(joinedAt)
	log.Printf("[DEBUG] IsProjectMember: found member for user %s in project %s: %+v", userID, projectID, member)
	
	return &member, nil
}

func (r *Repository) CanAssignTask(ctx context.Context, projectID, userID string) (bool, error) {
	query := `SELECT EXISTS(
        SELECT 1 FROM project_members pm
        JOIN users u ON u.id = pm.user_id
        WHERE pm.project_id = $1 
        AND pm.user_id = $2
        AND u.role IN ('ROLE_PROJECT_MANAGER', 'ROLE_GIP', 'ROLE_DIRECTOR')
    )`

	var exists bool
	err := r.db.QueryRowContext(ctx, query, projectID, userID).Scan(&exists)
	return exists, err
}

func (r *Repository) Add(ctx context.Context, member *projectv1.ProjectMember) error {
	if member.DepartmentId == "" {
		query := `INSERT INTO project_members (project_id, user_id, joined_at) VALUES ($1, $2, $3)`
		var joinedAt *time.Time
		if member.JoinedAt != nil {
			t := member.JoinedAt.AsTime()
			joinedAt = &t
		}
		_, err := r.db.ExecContext(ctx, query, member.ProjectId, member.UserId, joinedAt)
		return err
	}

	query := `INSERT INTO project_members (project_id, user_id, department_id, joined_at) VALUES ($1, $2, $3, $4)`
	var joinedAt *time.Time
	if member.JoinedAt != nil {
		t := member.JoinedAt.AsTime()
		joinedAt = &t
	}
	_, err := r.db.ExecContext(ctx, query,
		member.ProjectId,
		member.UserId,
		member.DepartmentId,
		joinedAt,
	)
	return err
}

func (r *Repository) IsProjectInDepartment(ctx context.Context, projectID, departmentID string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM project_members pm JOIN users u ON u.id= pm.user_id WHERE pm.project_id = $1 AND u.department_id = $2)`
	var exists bool
	err := r.db.QueryRowContext(ctx, query, projectID, departmentID).Scan(&exists)
	return exists, err
}

func (r *Repository) IsManagerOfProject(ctx context.Context, userID, projectID string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM project_members WHERE project_id = $1 AND user_id = $2)`

	var exists bool
	err := r.db.QueryRowContext(ctx, query, projectID, userID).Scan(&exists)
	return exists, err
}

func (r *Repository) GetProjectMembers(ctx context.Context, projectID string) ([]*projectv1.ProjectMember, error) {
	query := `
		SELECT pm.project_id, pm.user_id, COALESCE(pm.department_id, u.department_id) AS department_id, pm.joined_at,
		       COALESCE(NULLIF(d.name, ''), NULLIF(ud.name, ''), '') AS department_name,
		       COALESCE(NULLIF(u.full_name, ''), '') AS user_full_name,
		       COALESCE(NULLIF(u.email, ''), '') AS user_email
		FROM project_members pm
		LEFT JOIN users u ON u.id = pm.user_id
		LEFT JOIN departments d ON d.id = COALESCE(pm.department_id, u.department_id)
		LEFT JOIN departments ud ON ud.id = u.department_id
		WHERE pm.project_id = $1
		ORDER BY u.full_name NULLS LAST, pm.joined_at`

	var members []*projectv1.ProjectMember
	rows, err := r.db.QueryContext(ctx, query, projectID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var member projectv1.ProjectMember
		var joinedAt sql.NullTime
		var deptID sql.NullString
		var deptName, fullName, email string

		err := rows.Scan(
			&member.ProjectId,
			&member.UserId,
			&deptID,
			&joinedAt,
			&deptName,
			&fullName,
			&email,
		)
		if err != nil {
			return nil, err
		}
		if deptID.Valid {
			member.DepartmentId = deptID.String
		}
		member.DepartmentName = deptName
		member.UserFullName = fullName
		member.UserEmail = email

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
	query := `SELECT project_id, user_Id, department_id, joined_at FROM project_members WHERE user_id = $1`
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
	query := `UPDATE project_members SET department_id = $1, joined_at = $2 WHERE project_id = $3 AND user_id = $4`
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

func (r *Repository) SyncDepartmentFromUser(ctx context.Context, projectID, userID string) error {
	query := `
		UPDATE project_members pm
		SET department_id = u.department_id
		FROM users u
		WHERE pm.project_id = $1
		  AND pm.user_id = $2
		  AND pm.department_id IS NULL
		  AND u.department_id IS NOT NULL`
	_, err := r.db.ExecContext(ctx, query, projectID, userID)
	return err
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
