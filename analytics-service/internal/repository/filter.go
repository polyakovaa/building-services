package repository

import (
	"fmt"
	"strings"

	"github.com/lib/pq"
)
type AnalyticsFilter struct {
	AssigneeDeptID string
	ProjectID      string
	ProjectIDs     []string
	FromDate       string
	ToDate         string
}

func NewAnalyticsFilter(dept, projectID string, projectIDs []string, from, to string) AnalyticsFilter {
	return AnalyticsFilter{
		AssigneeDeptID: dept,
		ProjectID:      projectID,
		ProjectIDs:     projectIDs,
		FromDate:       from,
		ToDate:         to,
	}
}

func (f AnalyticsFilter) HasDateRange() bool {
	return f.FromDate != "" && f.ToDate != ""
}

const (
	usersJoinSQL         = `LEFT JOIN users u ON u.id = ta.assigned_user_id`
	taskOverdueSQL       = `((ta.completed_at IS NOT NULL AND ta.due_date IS NOT NULL AND ta.completed_at > ta.due_date) OR (ta.completed_at IS NULL AND ta.due_date IS NOT NULL AND ta.due_date < CURRENT_TIMESTAMP))`
	completedOnTimeSQL   = `(ta.completed_at IS NOT NULL AND ta.due_date IS NOT NULL AND ta.completed_at <= ta.due_date)`
	wipPredSQL           = `ta.completed_at IS NULL AND COALESCE(ta.status, 0) != 3`
	assigneeDepartmentSQL = `COALESCE(u.department_id, ta.department_id)`
)

func (f AnalyticsFilter) omitProjectFilter() bool {
	return f.ProjectID == "" && len(f.ProjectIDs) == 0
}

func (f AnalyticsFilter) appendTaskScope(tableAlias string, startArg int) (taskScopeSQL string, args []interface{}, nextArg int) {
	nextArg = startArg
	var parts []string
	if f.AssigneeDeptID != "" {
		args = append(args, f.AssigneeDeptID)
		parts = append(parts, fmt.Sprintf("%s = $%d", assigneeDepartmentSQL, nextArg))
		nextArg++
	}
	if !f.omitProjectFilter() {
		if f.ProjectID != "" {
			args = append(args, f.ProjectID)
			parts = append(parts, fmt.Sprintf("%s.project_id = $%d", tableAlias, nextArg))
			nextArg++
		} else if len(f.ProjectIDs) > 0 {
			args = append(args, pq.Array(f.ProjectIDs))
			parts = append(parts, fmt.Sprintf("%s.project_id = ANY($%d::uuid[])", tableAlias, nextArg))
			nextArg++
		}
	}
	if len(parts) == 0 {
		return "TRUE", args, nextArg
	}
	return strings.Join(parts, " AND "), args, nextArg
}

func (f AnalyticsFilter) appendProjectID(tableAlias string, startArg int) (projectScopeSQL string, args []interface{}, nextArg int) {
	nextArg = startArg
	if f.omitProjectFilter() {
		return "TRUE", args, nextArg
	}
	if f.ProjectID != "" {
		args = append(args, f.ProjectID)
		return fmt.Sprintf("%s.project_id = $%d", tableAlias, nextArg), args, nextArg + 1
	}
	if len(f.ProjectIDs) > 0 {
		args = append(args, pq.Array(f.ProjectIDs))
		return fmt.Sprintf("%s.project_id = ANY($%d::uuid[])", tableAlias, nextArg), args, nextArg + 1
	}
	return "TRUE", args, nextArg
}

func (f AnalyticsFilter) appendPeriodTaskSet(tableAlias string, args []interface{}, startArg int) (activeInPeriodSQL string, outArgs []interface{}, nextArg int) {
	if !f.HasDateRange() {
		return "", args, startArg
	}
	outArgs = append(args, f.ToDate, f.FromDate)
	activeInPeriodSQL = fmt.Sprintf(
		` AND %s.created_at::date <= $%d::date AND (%s.completed_at IS NULL OR %s.completed_at::date >= $%d::date)`,
		tableAlias, startArg, tableAlias, tableAlias, startArg+1)
	return activeInPeriodSQL, outArgs, startArg + 2
}

func (f AnalyticsFilter) appendCompletedInPeriod(tableAlias string, args []interface{}, startArg int) (completedInPeriodSQL string, outArgs []interface{}, nextArg int) {
	if !f.HasDateRange() {
		return "", args, startArg
	}
	outArgs = append(args, f.FromDate, f.ToDate)
	completedInPeriodSQL = fmt.Sprintf(` AND %s.completed_at IS NOT NULL AND %s.completed_at::date >= $%d::date AND %s.completed_at::date <= $%d::date`,
		tableAlias, tableAlias, startArg, tableAlias, startArg+1)
	return completedInPeriodSQL, outArgs, startArg + 2
}
