package repository

import (
	"database/sql"
	"fmt"
	"time"

	analyticsv1 "building-services/gen/analytics/v1"
)

func (r *Repository) GetDashboardStats(f AnalyticsFilter) (activeProjects, totalTasks, overdueTasks int, completionRate, onTimeRate float64, err error) {
	projectScopeSQL, projArgs, _ := f.appendProjectID("p", 1)
	err = r.db.QueryRow(fmt.Sprintf(
		`SELECT COUNT(*) FROM projects p WHERE (p.end_date IS NULL OR p.end_date > CURRENT_DATE) AND %s`, projectScopeSQL),
		projArgs...).Scan(&activeProjects)
	if err != nil {
		return
	}

	taskScopeSQL, baseArgs, nextArg := f.appendTaskScope("ta", 1)
	activeInPeriodSQL := ""
	args := baseArgs
	if f.HasDateRange() {
		activeInPeriodSQL, args, nextArg = f.appendPeriodTaskSet("ta", args, nextArg)
	}
	taskWhereSQL := fmt.Sprintf("%s%s", taskScopeSQL, activeInPeriodSQL)

	qTotal := fmt.Sprintf(`SELECT COUNT(*) FROM task_analytics ta %s WHERE %s`, usersJoinSQL, taskWhereSQL)
	if err = r.db.QueryRow(qTotal, args...).Scan(&totalTasks); err != nil {
		return
	}

	var completedCount int
	if f.HasDateRange() {
		completedInPeriodSQL, compArgs, _ := f.appendCompletedInPeriod("ta", args, nextArg)
		qComp := fmt.Sprintf(`SELECT COUNT(*) FROM task_analytics ta %s WHERE %s%s`, usersJoinSQL, taskWhereSQL, completedInPeriodSQL)
		if err = r.db.QueryRow(qComp, compArgs...).Scan(&completedCount); err != nil {
			return
		}
	} else {
		qComp := fmt.Sprintf(`SELECT COUNT(*) FROM task_analytics ta %s WHERE %s AND ta.completed_at IS NOT NULL`, usersJoinSQL, taskWhereSQL)
		if err = r.db.QueryRow(qComp, args...).Scan(&completedCount); err != nil {
			return
		}
	}
	if totalTasks > 0 {
		completionRate = float64(completedCount) / float64(totalTasks) * 100
	}

	qOver := fmt.Sprintf(`SELECT COUNT(*) FROM task_analytics ta %s WHERE %s AND (%s)`, usersJoinSQL, taskWhereSQL, taskOverdueSQL)
	if err = r.db.QueryRow(qOver, args...).Scan(&overdueTasks); err != nil {
		return
	}

	qOT := fmt.Sprintf(`
		SELECT COUNT(*) FILTER (WHERE %s), COUNT(*) FILTER (WHERE %s)
		FROM task_analytics ta %s WHERE %s`, completedOnTimeSQL, taskOverdueSQL, usersJoinSQL, taskWhereSQL)
	var onTimeCount, overdueForRate int
	if err = r.db.QueryRow(qOT, args...).Scan(&onTimeCount, &overdueForRate); err != nil {
		return
	}
	if onTimeCount+overdueForRate > 0 {
		onTimeRate = float64(onTimeCount) / float64(onTimeCount+overdueForRate) * 100
	}
	return
}

func (r *Repository) GetDepartmentWorkload(f AnalyticsFilter, days int) ([]*analyticsv1.DepartmentWorkload, error) {
	taskScopeSQL, args, nextArg := f.appendTaskScope("ta", 1)

	if f.HasDateRange() {
		activeInPeriodSQL, args, nextArg := f.appendPeriodTaskSet("ta", args, nextArg)
		taskWhereSQL := fmt.Sprintf("%s%s", taskScopeSQL, activeInPeriodSQL)
		completedInPeriodSQL, args, _ := f.appendCompletedInPeriod("ta", args, nextArg)
		completedPred := "TRUE" + completedInPeriodSQL

		deptName := `COALESCE(NULLIF(d.name, ''), 'Без отдела')`
		q := fmt.Sprintf(`
			SELECT
				COALESCE(MIN(%s::text), '') AS department_id,
				%s AS department_name,
				CAST(COUNT(*) FILTER (WHERE %s) AS INTEGER) AS wip,
				CAST(COUNT(*) FILTER (WHERE %s) AS INTEGER) AS completed,
				CAST(COUNT(*) FILTER (WHERE %s) AS INTEGER) AS overdue,
				COALESCE(AVG(ta.cycle_time_days) FILTER (WHERE %s), 0) AS avg_cycle_time,
				COALESCE(ROUND(100.0 * CAST(COUNT(*) FILTER (WHERE %s) AS NUMERIC)
					/ NULLIF(CAST(COUNT(*) FILTER (WHERE %s) AS INTEGER) + CAST(COUNT(*) FILTER (WHERE %s) AS INTEGER), 0), 2), 0) AS productivity,
				COALESCE(ROUND(100.0 * CAST(COUNT(*) FILTER (WHERE %s) AS NUMERIC)
					/ NULLIF(CAST(COUNT(*) FILTER (WHERE %s) AS INTEGER) + CAST(COUNT(*) FILTER (WHERE %s) AS INTEGER), 0), 2), 0) AS on_time_rate
			FROM task_analytics ta
			%s
			LEFT JOIN departments d ON d.id = %s
			WHERE %s
			GROUP BY %s
			ORDER BY department_name`,
			assigneeDepartmentSQL, deptName,
			wipPredSQL, completedPred, taskOverdueSQL, completedPred,
			completedPred, wipPredSQL, completedPred,
			completedOnTimeSQL, taskOverdueSQL, completedOnTimeSQL,
			usersJoinSQL, assigneeDepartmentSQL, taskWhereSQL, deptName)

		rows, err := r.db.Query(q, args...)
		return scanDepartmentWorkloadRows(rows, err)
	}

	completedPred := fmt.Sprintf(`ta.completed_at IS NOT NULL AND ta.completed_at::date >= CURRENT_DATE - ($%d * INTERVAL '1 day') AND ta.completed_at::date <= CURRENT_DATE`, nextArg)
	overduePred := fmt.Sprintf(`(
		(ta.completed_at IS NOT NULL AND ta.completed_at::date >= CURRENT_DATE - ($%d * INTERVAL '1 day') AND ta.is_overdue = TRUE)
		OR (ta.completed_at IS NULL AND ta.due_date IS NOT NULL AND ta.due_date < CURRENT_TIMESTAMP)
	)`, nextArg)
	args = append(args, days)

	deptName := `COALESCE(NULLIF(d.name, ''), 'Без отдела')`
	q := fmt.Sprintf(`
		SELECT
			COALESCE(MIN(%s::text), '') AS department_id,
			%s AS department_name,
			CAST(COUNT(*) FILTER (WHERE %s) AS INTEGER) AS wip,
			CAST(COUNT(*) FILTER (WHERE %s) AS INTEGER) AS completed,
			CAST(COUNT(*) FILTER (WHERE %s) AS INTEGER) AS overdue,
			COALESCE(AVG(ta.cycle_time_days) FILTER (WHERE %s), 0) AS avg_cycle_time,
			COALESCE(ROUND(100.0 * CAST(COUNT(*) FILTER (WHERE %s) AS NUMERIC)
				/ NULLIF(CAST(COUNT(*) FILTER (WHERE %s) AS INTEGER) + CAST(COUNT(*) FILTER (WHERE %s) AS INTEGER), 0), 2), 0) AS productivity,
			COALESCE(ROUND(100.0 * CAST(COUNT(*) FILTER (WHERE %s) AS NUMERIC)
				/ NULLIF(CAST(COUNT(*) FILTER (WHERE %s) AS INTEGER) + CAST(COUNT(*) FILTER (WHERE %s) AS INTEGER), 0), 2), 0) AS on_time_rate
		FROM task_analytics ta
		%s
		LEFT JOIN departments d ON d.id = %s
		WHERE %s
		GROUP BY %s
		ORDER BY department_name`,
		assigneeDepartmentSQL, deptName,
		wipPredSQL, completedPred, overduePred, completedPred,
		completedPred, wipPredSQL, completedPred,
		completedOnTimeSQL, overduePred, completedOnTimeSQL,
		usersJoinSQL, assigneeDepartmentSQL, taskScopeSQL, deptName)

	rows, err := r.db.Query(q, args...)
	return scanDepartmentWorkloadRows(rows, err)
}

func (r *Repository) GetTaskTrends(f AnalyticsFilter, weeks int, groupBy string) ([]*analyticsv1.WeeklyTrend, error) {
	bucket := "week"
	var bucketsCTE string
	queryArgs := []interface{}{}

	if f.HasDateRange() {
		queryArgs = append(queryArgs, f.FromDate, f.ToDate)
		if groupBy == "day" {
			bucket = "day"
			bucketsCTE = `SELECT gs::date AS bucket
				FROM bounds
				CROSS JOIN generate_series(bounds.start_date, bounds.end_date, INTERVAL '1 day') gs`
		} else {
			bucketsCTE = `SELECT DISTINCT date_trunc('week', gs::timestamp)::date AS bucket
				FROM bounds
				CROSS JOIN generate_series(bounds.start_date, bounds.end_date, INTERVAL '1 day') gs`
		}
	} else {
		if weeks < 1 {
			weeks = 1
		}
		if weeks == 1 || groupBy == "day" {
			bucket = "day"
			bucketsCTE = `SELECT generate_series(
				(CURRENT_DATE - INTERVAL '6 days')::date, CURRENT_DATE::date, INTERVAL '1 day')::date AS bucket`
		} else {
			queryArgs = append(queryArgs, weeks)
			bucketsCTE = fmt.Sprintf(`SELECT DISTINCT date_trunc('week', gs::timestamp)::date AS bucket
				FROM generate_series(
					date_trunc('week', CURRENT_DATE - (($%d::int - 1) * INTERVAL '1 week'))::date,
					date_trunc('week', CURRENT_DATE)::date,
					INTERVAL '1 day') gs`, len(queryArgs))
		}
	}

	taskScopeSQL, scopeArgs, _ := f.appendTaskScope("ta", len(queryArgs)+1)
	queryArgs = append(queryArgs, scopeArgs...)
	scopeFilter := " AND " + taskScopeSQL

	var boundsSelect string
	if f.HasDateRange() {
		boundsSelect = `SELECT $1::date AS start_date, $2::date AS end_date`
	} else if weeks == 1 || groupBy == "day" {
		boundsSelect = `SELECT (CURRENT_DATE - INTERVAL '6 days')::date AS start_date, CURRENT_DATE::date AS end_date`
	} else {
		boundsSelect = fmt.Sprintf(`SELECT date_trunc('week', CURRENT_DATE - (($%d::int - 1) * INTERVAL '1 week'))::date AS start_date,
			date_trunc('week', CURRENT_DATE)::date AS end_date`, len(queryArgs)-len(scopeArgs))
	}

	query := fmt.Sprintf(`
		WITH bounds AS (%s),
		buckets AS (%s),
		events AS (
			SELECT date_trunc('%s', ta.created_at)::date AS bucket,
				   1 AS created_cnt, 0 AS completed_cnt, 0 AS overdue_cnt
			FROM task_analytics ta
			%s, bounds
			WHERE ta.created_at::date >= bounds.start_date
			  AND ta.created_at::date <= bounds.end_date
			  %s
			UNION ALL
			SELECT date_trunc('%s', ta.completed_at)::date AS bucket,
				   0, 1, CASE WHEN ta.is_overdue THEN 1 ELSE 0 END
			FROM task_analytics ta
			%s, bounds
			WHERE ta.completed_at IS NOT NULL
			  AND ta.completed_at::date >= bounds.start_date
			  AND ta.completed_at::date <= bounds.end_date
			  %s
		)
		SELECT b.bucket,
			   COALESCE(SUM(e.created_cnt), 0)::int,
			   COALESCE(SUM(e.completed_cnt), 0)::int,
			   COALESCE(SUM(e.overdue_cnt), 0)::int,
			   CASE WHEN COALESCE(SUM(e.created_cnt), 0) > 0
				THEN ROUND(100.0 * COALESCE(SUM(e.completed_cnt), 0)::numeric / SUM(e.created_cnt), 2)::double precision
				ELSE 0::double precision END,
			   CASE WHEN COALESCE(SUM(e.completed_cnt), 0) > 0
				THEN ROUND(100.0 * (COALESCE(SUM(e.completed_cnt), 0) - COALESCE(SUM(e.overdue_cnt), 0))::numeric / SUM(e.completed_cnt), 2)::double precision
				ELSE 0::double precision END
		FROM buckets b
		LEFT JOIN events e ON e.bucket = b.bucket
		GROUP BY b.bucket
		ORDER BY b.bucket ASC`, boundsSelect, bucketsCTE, bucket, usersJoinSQL, scopeFilter, bucket, usersJoinSQL, scopeFilter)

	return scanWeeklyTrendRows(r.db.Query(query, queryArgs...))
}

func (r *Repository) GetProjectTimeline(f AnalyticsFilter) ([]*analyticsv1.ProjectTimelineControl, error) {
	taskScopeSQL, args, nextArg := f.appendTaskScope("ta", 1)
	activeInPeriodSQL := ""
	if f.HasDateRange() {
		activeInPeriodSQL, args, _ = f.appendPeriodTaskSet("ta", args, nextArg)
	}
	taskWhereSQL := fmt.Sprintf("%s%s", taskScopeSQL, activeInPeriodSQL)

	inner := fmt.Sprintf(`
		SELECT x.project_id::uuid,
			x.total_tasks,
			x.completed_on_time,
			x.overdue_tasks,
			CASE WHEN (x.completed_on_time + x.overdue_tasks) > 0
				THEN ROUND(100.0 * x.completed_on_time::numeric / (x.completed_on_time + x.overdue_tasks)::numeric, 2)
				ELSE 0::numeric END AS on_time_rate,
			x.avg_delay_days
		FROM (
			SELECT ta.project_id,
				COUNT(*) AS total_tasks,
				COUNT(*) FILTER (WHERE %s) AS completed_on_time,
				COUNT(*) FILTER (WHERE %s) AS overdue_tasks,
				COALESCE(AVG(
					CASE
						WHEN ta.completed_at IS NOT NULL AND ta.due_date IS NOT NULL AND ta.completed_at > ta.due_date
							THEN GREATEST(CEIL(EXTRACT(EPOCH FROM (ta.completed_at - ta.due_date)) / 86400), 1)
						WHEN ta.completed_at IS NULL AND ta.due_date IS NOT NULL AND ta.due_date < CURRENT_TIMESTAMP
							THEN GREATEST(CEIL(EXTRACT(EPOCH FROM (CURRENT_TIMESTAMP - ta.due_date)) / 86400), 1)
						ELSE NULL
					END
				), 0) AS avg_delay_days
			FROM task_analytics ta
			%s
			WHERE %s
			GROUP BY ta.project_id
		) x`, completedOnTimeSQL, taskOverdueSQL, usersJoinSQL, taskWhereSQL)

	query := fmt.Sprintf(`
		SELECT t.project_id::text, COALESCE(p.project_name, ''),
			COALESCE(t.total_tasks, 0), COALESCE(t.completed_on_time, 0), COALESCE(t.overdue_tasks, 0),
			COALESCE(t.on_time_rate, 0)::double precision, COALESCE(t.avg_delay_days, 0)
		FROM (%s) t
		LEFT JOIN projects p ON p.project_id = t.project_id
		ORDER BY COALESCE(p.project_name, '')`, inner)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var projects []*analyticsv1.ProjectTimelineControl
	for rows.Next() {
		p := &analyticsv1.ProjectTimelineControl{}
		if err := rows.Scan(&p.ProjectId, &p.ProjectName, &p.TotalTasks, &p.CompletedOnTime, &p.OverdueTasks, &p.OnTimeRate, &p.AvgDelayDays); err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, rows.Err()
}

func (r *Repository) GetEmployeeProductivity(f AnalyticsFilter) ([]*analyticsv1.EmployeeProductivity, error) {
	taskScopeSQL, args, nextArg := f.appendTaskScope("ta", 1)
	activeInPeriodSQL := ""
	if f.HasDateRange() {
		activeInPeriodSQL, args, nextArg = f.appendPeriodTaskSet("ta", args, nextArg)
	}
	taskWhereSQL := fmt.Sprintf("%s%s", taskScopeSQL, activeInPeriodSQL)

	completedFilter := ""
	if f.HasDateRange() {
		var completedInPeriodSQL string
		completedInPeriodSQL, args, _ = f.appendCompletedInPeriod("ta", args, nextArg)
		completedFilter = completedInPeriodSQL
	}

	assigneeKeySQL := `COALESCE(NULLIF(lower(trim(u.email)), ''), ta.assigned_user_id::text)`
	query := fmt.Sprintf(`
		SELECT
			(array_agg(ta.assigned_user_id::text ORDER BY u.updated_at DESC NULLS LAST, ta.assigned_user_id::text))[1] AS user_id,
			COALESCE(NULLIF(MAX(u.full_name), ''), (array_agg(ta.assigned_user_id::text ORDER BY ta.assigned_user_id::text))[1]) AS full_name,
			COALESCE(NULLIF(MAX(u.email), ''), '') AS email,
			COUNT(*) FILTER (WHERE ta.completed_at IS NOT NULL%s)::int AS tasks_completed,
			COUNT(*) FILTER (WHERE ta.completed_at IS NOT NULL AND (%s))::int AS tasks_overdue,
			COALESCE(AVG(ta.cycle_time_days) FILTER (WHERE ta.completed_at IS NOT NULL%s), 0) AS avg_cycle_time,
			COALESCE(ROUND(100.0 * COUNT(*) FILTER (WHERE ta.completed_at IS NOT NULL%s) / NULLIF(COUNT(*), 0), 2), 0) AS completion_rate,
			COALESCE(ROUND(100.0 * COUNT(*) FILTER (WHERE %s%s)
				/ NULLIF(COUNT(*) FILTER (WHERE %s) + COUNT(*) FILTER (WHERE %s), 0), 2), 0) AS on_time_rate
		FROM task_analytics ta
		%s
		WHERE ta.assigned_user_id IS NOT NULL AND %s
		GROUP BY %s
		ORDER BY COUNT(*) FILTER (WHERE ta.completed_at IS NOT NULL%s) DESC,
		         COALESCE(NULLIF(MAX(u.full_name), ''), (array_agg(ta.assigned_user_id::text ORDER BY ta.assigned_user_id::text))[1]) ASC`,
		completedFilter, taskOverdueSQL, completedFilter, completedFilter,
		completedOnTimeSQL, completedFilter, completedOnTimeSQL, taskOverdueSQL,
		usersJoinSQL, taskWhereSQL, assigneeKeySQL, completedFilter)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var employees []*analyticsv1.EmployeeProductivity
	for rows.Next() {
		e := &analyticsv1.EmployeeProductivity{}
		if err := rows.Scan(&e.UserId, &e.FullName, &e.Email, &e.TasksCompleted, &e.TasksOverdue, &e.AvgCycleTime, &e.CompletionRate, &e.OnTimeRate); err != nil {
			return nil, err
		}
		employees = append(employees, e)
	}
	return employees, rows.Err()
}

func scanDepartmentWorkloadRows(rows *sql.Rows, err error) ([]*analyticsv1.DepartmentWorkload, error) {
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*analyticsv1.DepartmentWorkload
	for rows.Next() {
		w := &analyticsv1.DepartmentWorkload{}
		if err := rows.Scan(&w.DepartmentId, &w.DepartmentName, &w.Wip, &w.Completed, &w.Overdue, &w.AvgCycleTime, &w.Productivity, &w.OnTimeRate); err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

func scanWeeklyTrendRows(rows *sql.Rows, err error) ([]*analyticsv1.WeeklyTrend, error) {
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var trends []*analyticsv1.WeeklyTrend
	for rows.Next() {
		t := &analyticsv1.WeeklyTrend{}
		var bucket time.Time
		if err := rows.Scan(&bucket, &t.Created, &t.Completed, &t.Overdue, &t.CompletionRate, &t.OnTimeRate); err != nil {
			return nil, err
		}
		t.Week = bucket.Format("2006-01-02")
		trends = append(trends, t)
	}
	return trends, rows.Err()
}
