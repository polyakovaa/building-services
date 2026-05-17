package util

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"
)

func ParseTime(value string) *time.Time {
	if value == "" {
		return nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return nil
	}
	return &parsed
}

func ParseOccurredAt(event map[string]interface{}) time.Time {
	s, _ := event["occurred_at"].(string)
	if s == "" {
		return time.Now().UTC()
	}
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t
	}
	return time.Now().UTC()
}

func EventKey(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

func TaskURL(taskID string) string {
	if taskID == "" {
		return "/tasks"
	}
	return "/task/" + taskID
}

func ProjectURL(projectID string) string {
	if projectID == "" {
		return "/projects"
	}
	return "/project/" + projectID
}

func NotificationMessage(event map[string]interface{}, extra map[string]interface{}) string {
	taskTitle, _ := event["task_title"].(string)
	if taskTitle == "" {
		taskTitle, _ = event["title"].(string)
	}
	projectName, _ := event["project_name"].(string)

	message := map[string]interface{}{
		"task_title":   taskTitle,
		"project_name": projectName,
	}
	for key, value := range extra {
		message[key] = value
	}
	body, err := json.Marshal(message)
	if err != nil {
		return "{}"
	}
	return string(body)
}
