package task

import (
	projectv1 "building-services/gen/project/v1"
	"database/sql"

	"google.golang.org/protobuf/types/known/timestamppb"
)

const taskSelectColumns = `id, project_id, title, description, status, priority, deadline, assigned_to, created_by, parent_task_id, created_at, updated_at, activity_type_id, planned_hours, actual_hours`

func scanTask(
	t *projectv1.Task,
	status, priority int32,
	deadline, createdAt, updatedAt sql.NullTime,
	assignedTo, createdBy, parentTaskId sql.NullString,
	activityTypeID sql.NullString,
	plannedHours, actualHours sql.NullFloat64,
) {
	t.Status = projectv1.TaskStatus(status)
	t.Priority = projectv1.TaskPriority(priority)
	if deadline.Valid {
		t.Deadline = timestamppb.New(deadline.Time)
	}
	if createdAt.Valid {
		t.CreatedAt = timestamppb.New(createdAt.Time)
	}
	if updatedAt.Valid {
		t.UpdatedAt = timestamppb.New(updatedAt.Time)
	}
	if assignedTo.Valid {
		t.AssignedTo = assignedTo.String
	}
	if createdBy.Valid {
		t.CreatedBy = createdBy.String
	}
	if parentTaskId.Valid {
		t.ParentTaskId = parentTaskId.String
	}
	if activityTypeID.Valid {
		t.ActivityTypeId = activityTypeID.String
	}
	if plannedHours.Valid {
		t.PlannedHours = plannedHours.Float64
	}
	if actualHours.Valid {
		t.ActualHours = actualHours.Float64
	}
}

func laborArgs(task *projectv1.Task) (activityTypeID interface{}, plannedHours interface{}, actualHours interface{}) {
	if task.ActivityTypeId != "" {
		activityTypeID = task.ActivityTypeId
	}
	if task.PlannedHours > 0 {
		plannedHours = task.PlannedHours
	}
	if task.ActualHours > 0 {
		actualHours = task.ActualHours
	}
	return activityTypeID, plannedHours, actualHours
}
