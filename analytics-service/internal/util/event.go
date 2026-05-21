package util

import (
	"time"

	"building-services/analytics-service/internal/repository"
)

func AssigneeFromEvent(event map[string]interface{}) (repository.User, bool) {
	userID, _ := event["user_id"].(string)
	if userID == "" {
		userID, _ = event["assignee_user_id"].(string)
	}
	if userID == "" {
		userID, _ = event["to_user_id"].(string)
	}
	if userID == "" {
		return repository.User{}, false
	}

	dept, _ := event["department_id"].(string)
	if dept == "" {
		dept, _ = event["assignee_department_id"].(string)
	}
	if dept == "" {
		dept, _ = event["to_department_id"].(string)
	}

	name, _ := event["assignee_full_name"].(string)
	email, _ := event["assignee_email"].(string)

	return repository.User{
		ID:           userID,
		DepartmentID: dept,
		FullName:     name,
		Email:        email,
	}, true
}

func ParseEventTime(occurredAtStr string) time.Time {
	if occurredAtStr == "" {
		return time.Now()
	}
	if t, err := time.Parse(time.RFC3339Nano, occurredAtStr); err == nil {
		return t
	}
	if t, err := time.Parse(time.RFC3339, occurredAtStr); err == nil {
		return t
	}
	return time.Now()
}

func LaborFromEvent(event map[string]interface{}) (activityTypeID string, plannedHours, actualHours float64) {
	activityTypeID, _ = event["activity_type_id"].(string)
	if v, ok := event["planned_hours"].(float64); ok {
		plannedHours = v
	}
	if v, ok := event["actual_hours"].(float64); ok {
		actualHours = v
	}
	return activityTypeID, plannedHours, actualHours
}

func ParseOptionalTime(value string) *time.Time {
	if value == "" {
		return nil
	}
	if t, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return &t
	}
	return nil
}
