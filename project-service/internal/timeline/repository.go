package timeline

import (
	"context"
	"database/sql"
	"fmt"

	projectv1 "building-services/gen/project/v1"

	"google.golang.org/protobuf/types/known/timestamppb"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateEmpty(ctx context.Context, projectID string) error {
	query := `INSERT INTO project_timeline (project_id) VALUES ($1)`
	_, err := r.db.ExecContext(ctx, query, projectID)
	if err != nil {
		return fmt.Errorf("failed to create timeline: %w", err)
	}
	return nil
}

func (r *Repository) Get(ctx context.Context, projectID string) (*projectv1.ProjectTimeline, error) {
	query := `SELECT project_id, contract_date, work_start_date, work_end_date, 
              handover_date, comments_date, comments_fixed_date, acceptance_date, 
              final_payment_date, updated_at, updated_by
              FROM project_timeline WHERE project_id = $1`

	timeline := &projectv1.ProjectTimeline{}
	var (
		contractDate, workStartDate, workEndDate, handoverDate,
		commentsDate, commentsFixedDate, acceptanceDate, finalPaymentDate,
		updatedAt sql.NullTime
		updatedBy sql.NullString
	)

	err := r.db.QueryRowContext(ctx, query, projectID).Scan(
		&timeline.ProjectId,
		&contractDate, &workStartDate, &workEndDate,
		&handoverDate, &commentsDate, &commentsFixedDate,
		&acceptanceDate, &finalPaymentDate,
		&updatedAt, &updatedBy,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return &projectv1.ProjectTimeline{ProjectId: projectID}, nil
		}
		return nil, fmt.Errorf("failed to get timeline: %w", err)
	}

	setIfValid := func(dest **timestamppb.Timestamp, src sql.NullTime) {
		if src.Valid {
			*dest = timestamppb.New(src.Time)
		}
	}

	setIfValid(&timeline.ContractDate, contractDate)
	setIfValid(&timeline.WorkStartDate, workStartDate)
	setIfValid(&timeline.WorkEndDate, workEndDate)
	setIfValid(&timeline.HandoverDate, handoverDate)
	setIfValid(&timeline.CommentsDate, commentsDate)
	setIfValid(&timeline.CommentsFixedDate, commentsFixedDate)
	setIfValid(&timeline.AcceptanceDate, acceptanceDate)
	setIfValid(&timeline.FinalPaymentDate, finalPaymentDate)

	if updatedAt.Valid {
		timeline.UpdatedAt = timestamppb.New(updatedAt.Time)
	}
	if updatedBy.Valid {
		timeline.UpdatedBy = updatedBy.String
	}

	return timeline, nil
}

func (r *Repository) Upsert(ctx context.Context, timeline *projectv1.ProjectTimeline) error {
	query := `
        INSERT INTO project_timeline (
            project_id, contract_date, work_start_date, work_end_date,
            handover_date, comments_date, comments_fixed_date, acceptance_date,
            final_payment_date, updated_at, updated_by
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, CURRENT_TIMESTAMP, $10)
        ON CONFLICT (project_id) DO UPDATE SET
            contract_date = EXCLUDED.contract_date,
            work_start_date = EXCLUDED.work_start_date,
            work_end_date = EXCLUDED.work_end_date,
            handover_date = EXCLUDED.handover_date,
            comments_date = EXCLUDED.comments_date,
            comments_fixed_date = EXCLUDED.comments_fixed_date,
            acceptance_date = EXCLUDED.acceptance_date,
            final_payment_date = EXCLUDED.final_payment_date,
            updated_at = CURRENT_TIMESTAMP,
            updated_by = EXCLUDED.updated_by
    `

	toPtr := func(t *timestamppb.Timestamp) interface{} {
		if t == nil {
			return nil
		}
		return t.AsTime()
	}

	_, err := r.db.ExecContext(ctx, query,
		timeline.ProjectId,
		toPtr(timeline.ContractDate),
		toPtr(timeline.WorkStartDate),
		toPtr(timeline.WorkEndDate),
		toPtr(timeline.HandoverDate),
		toPtr(timeline.CommentsDate),
		toPtr(timeline.CommentsFixedDate),
		toPtr(timeline.AcceptanceDate),
		toPtr(timeline.FinalPaymentDate),
		timeline.UpdatedBy,
	)
	if err != nil {
		return fmt.Errorf("failed to upsert timeline: %w", err)
	}
	return nil
}
