package repository

import (
	"database/sql"
	"fmt"
	"strings"

	analyticsv1 "building-services/gen/analytics/v1"
)

const laborMetricsSelect = `
				COALESCE(SUM(ta.planned_hours), 0) AS planned,
				COALESCE(SUM(ta.actual_hours), 0) AS actual,
				COUNT(*)::int AS task_count,
				COUNT(*) FILTER (WHERE ta.planned_hours > 0)::int AS tasks_with_plan,
				COUNT(*) FILTER (WHERE ta.planned_hours > 0 AND ta.actual_hours > 0)::int AS tasks_comparable,
				COUNT(*) FILTER (WHERE ta.planned_hours > 0 AND ta.actual_hours > ta.planned_hours)::int AS overrun_tasks,
				COALESCE(ROUND(100.0 * (
					COALESCE(SUM(ta.actual_hours), 0) - COALESCE(SUM(ta.planned_hours), 0)
				) / NULLIF(SUM(ta.planned_hours) FILTER (WHERE ta.planned_hours > 0), 0), 2), 0) AS deviation_percent,
				COALESCE(AVG(ta.actual_hours) FILTER (WHERE ta.completed_at IS NOT NULL AND ta.actual_hours > 0), 0) AS avg_actual_completed`

func (r *Repository) PatchTaskLabor(taskID, activityTypeID string, plannedHours, actualHours float64) error {
	if taskID == "" {
		return nil
	}
	const q = `UPDATE task_analytics SET
		activity_type_id = COALESCE(NULLIF($2, '')::uuid, activity_type_id),
		planned_hours = CASE WHEN $3 > 0 THEN $3 ELSE planned_hours END,
		actual_hours = CASE WHEN $4 > 0 THEN $4 ELSE actual_hours END,
		updated_at = CURRENT_TIMESTAMP
		WHERE task_id = $1`
	_, err := r.db.Exec(q, taskID, activityTypeID, plannedHours, actualHours)
	return err
}

func (r *Repository) GetLaborPlanFact(f AnalyticsFilter, groupBy string) (*analyticsv1.LaborPlanFactResponse, error) {
	taskScopeSQL, args, nextArg := f.appendTaskScope("ta", 1)
	activeInPeriodSQL := ""
	if f.HasDateRange() {
		activeInPeriodSQL, args, _ = f.appendPeriodTaskSet("ta", args, nextArg)
	}
	taskWhereSQL := fmt.Sprintf("%s%s", taskScopeSQL, activeInPeriodSQL)

	rows, err := r.queryLaborGroups(groupBy, taskWhereSQL, args)
	if err != nil {
		return nil, err
	}

	totals, err := r.queryLaborTotals(taskWhereSQL, args)
	if err != nil {
		return nil, err
	}

	resp := &analyticsv1.LaborPlanFactResponse{
		Rows:                   rows,
		TotalPlannedHours:      totals.planned,
		TotalActualHours:       totals.actual,
		TotalDeviationPercent:  totals.deviationPercent,
		TasksWithPlan:          totals.tasksWithPlan,
		TasksComparable:        totals.tasksComparable,
		OverrunTasks:           totals.overrunTasks,
		AvgActualPerCompleted:  totals.avgActualCompleted,
	}
	return resp, nil
}

type laborTotals struct {
	planned            float64
	actual             float64
	deviationPercent   float64
	tasksWithPlan      int32
	tasksComparable    int32
	overrunTasks       int32
	avgActualCompleted float64
}

func (r *Repository) queryLaborTotals(taskWhereSQL string, args []interface{}) (laborTotals, error) {
	q := fmt.Sprintf(`SELECT
			COALESCE(SUM(ta.planned_hours), 0),
			COALESCE(SUM(ta.actual_hours), 0),
			COUNT(*) FILTER (WHERE ta.planned_hours > 0)::int,
			COUNT(*) FILTER (WHERE ta.planned_hours > 0 AND ta.actual_hours > 0)::int,
			COUNT(*) FILTER (WHERE ta.planned_hours > 0 AND ta.actual_hours > ta.planned_hours)::int,
			COALESCE(ROUND(100.0 * (
				COALESCE(SUM(ta.actual_hours), 0) - COALESCE(SUM(ta.planned_hours), 0)
			) / NULLIF(SUM(ta.planned_hours) FILTER (WHERE ta.planned_hours > 0), 0), 2), 0),
			COALESCE(AVG(ta.actual_hours) FILTER (WHERE ta.completed_at IS NOT NULL AND ta.actual_hours > 0), 0)
		FROM task_analytics ta
		%s
		WHERE %s`, usersJoinSQL, taskWhereSQL)

	var t laborTotals
	err := r.db.QueryRow(q, args...).Scan(
		&t.planned,
		&t.actual,
		&t.tasksWithPlan,
		&t.tasksComparable,
		&t.overrunTasks,
		&t.deviationPercent,
		&t.avgActualCompleted,
	)
	return t, err
}

func (r *Repository) queryLaborGroups(groupBy, taskWhereSQL string, args []interface{}) ([]*analyticsv1.LaborPlanFactRow, error) {
	groupBy = strings.ToLower(strings.TrimSpace(groupBy))
	var q string
	switch groupBy {
	case "department":
		deptName := `COALESCE(NULLIF(d.name, ''), 'Без отдела')`
		q = fmt.Sprintf(`
			SELECT
				COALESCE(MIN(%s::text), '') AS group_id,
				%s AS group_name,
				%s
			FROM task_analytics ta
			%s
			LEFT JOIN departments d ON d.id = %s
			WHERE %s
			GROUP BY %s
			HAVING COUNT(*) FILTER (WHERE ta.planned_hours > 0 OR ta.actual_hours > 0) > 0
			ORDER BY group_name`, assigneeDepartmentSQL, deptName, laborMetricsSelect, usersJoinSQL, assigneeDepartmentSQL, taskWhereSQL, deptName)
	case "activity":
		q = fmt.Sprintf(`
			SELECT
				COALESCE(MIN(at.id::text), '') AS group_id,
				COALESCE(NULLIF(MIN(at.name), ''), 'Без вида работ') AS group_name,
				%s
			FROM task_analytics ta
			%s
			LEFT JOIN activity_types at ON at.id = ta.activity_type_id
			WHERE %s
			GROUP BY COALESCE(at.id::text, ''), COALESCE(at.name, 'Без вида работ')
			HAVING COUNT(*) FILTER (WHERE ta.planned_hours > 0 OR ta.actual_hours > 0) > 0
			ORDER BY group_name`, laborMetricsSelect, usersJoinSQL, taskWhereSQL)
	case "project":
		q = fmt.Sprintf(`
			SELECT
				COALESCE(MIN(p.project_id::text), '') AS group_id,
				COALESCE(NULLIF(MIN(p.project_name), ''), 'Без проекта') AS group_name,
				%s
			FROM task_analytics ta
			%s
			LEFT JOIN projects p ON p.project_id = ta.project_id
			WHERE %s
			GROUP BY COALESCE(p.project_id::text, ''), COALESCE(p.project_name, 'Без проекта')
			HAVING COUNT(*) FILTER (WHERE ta.planned_hours > 0 OR ta.actual_hours > 0) > 0
			ORDER BY group_name`, laborMetricsSelect, usersJoinSQL, taskWhereSQL)
	default:
		q = fmt.Sprintf(`
			SELECT
				'' AS group_id,
				'Итого' AS group_name,
				%s
			FROM task_analytics ta
			%s
			WHERE %s`, laborMetricsSelect, usersJoinSQL, taskWhereSQL)
	}

	rows, err := r.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*analyticsv1.LaborPlanFactRow
	for rows.Next() {
		row, err := scanLaborRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func scanLaborRow(rows *sql.Rows) (*analyticsv1.LaborPlanFactRow, error) {
	var row analyticsv1.LaborPlanFactRow
	var planned, actual, deviation, avgActual sql.NullFloat64
	if err := rows.Scan(
		&row.GroupId,
		&row.GroupName,
		&planned,
		&actual,
		&row.TaskCount,
		&row.TasksWithPlan,
		&row.TasksComparable,
		&row.OverrunTasks,
		&deviation,
		&avgActual,
	); err != nil {
		return nil, err
	}
	if planned.Valid {
		row.PlannedHours = planned.Float64
	}
	if actual.Valid {
		row.ActualHours = actual.Float64
	}
	if deviation.Valid {
		row.DeviationPercent = deviation.Float64
	}
	if avgActual.Valid {
		row.AvgActualPerCompleted = avgActual.Float64
	}
	return &row, nil
}
