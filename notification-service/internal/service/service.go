package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"building-services/notification-service/internal/repository"
	"building-services/notification-service/internal/util"

	notificationv1 "building-services/gen/notification/v1"
)

type Repository interface {
	CreateNotification(ctx context.Context, params repository.CreateNotificationParams) error
	UpsertNotificationTask(ctx context.Context, task repository.NotificationTask) error
	UpdateTaskAssignee(ctx context.Context, taskID, assigneeUserID string) error
	UpdateTaskDeadline(ctx context.Context, taskID string, deadline *time.Time, assigneeUserID, taskTitle, projectName string) error
	UpdateTaskStatus(ctx context.Context, taskID string, status int32, completedAt *time.Time, taskTitle, projectName string) error
	ListUpcomingDeadlineTasks(ctx context.Context, now, until time.Time) ([]repository.NotificationTask, error)
	ListOverdueTasks(ctx context.Context, now time.Time) ([]repository.NotificationTask, error)
	ListNotifications(ctx context.Context, userID string, pageSize int, pageToken string, unreadOnly bool) ([]repository.Notification, string, error)
	GetUnreadCount(ctx context.Context, userID string) (int32, error)
	MarkAsRead(ctx context.Context, userID, notificationID string) (repository.Notification, error)
	MarkAllAsRead(ctx context.Context, userID string) error
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) ListNotifications(ctx context.Context, userID string, req *notificationv1.ListNotificationsRequest) (*notificationv1.ListNotificationsResponse, error) {
	items, nextToken, err := s.repo.ListNotifications(ctx, userID, int(req.PageSize), req.PageToken, req.UnreadOnly)
	if err != nil {
		return nil, err
	}

	out := make([]*notificationv1.Notification, 0, len(items))
	for _, item := range items {
		out = append(out, repository.ToProto(item))
	}

	return &notificationv1.ListNotificationsResponse{
		Notifications: out,
		NextPageToken: nextToken,
	}, nil
}

func (s *Service) GetUnreadCount(ctx context.Context, userID string) (*notificationv1.UnreadCountResponse, error) {
	count, err := s.repo.GetUnreadCount(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &notificationv1.UnreadCountResponse{Count: count}, nil
}

func (s *Service) MarkAsRead(ctx context.Context, userID, notificationID string) (*notificationv1.Notification, error) {
	if notificationID == "" {
		return nil, fmt.Errorf("notification_id required")
	}

	notification, err := s.repo.MarkAsRead(ctx, userID, notificationID)
	if err != nil {
		return nil, err
	}
	return repository.ToProto(notification), nil
}

func (s *Service) MarkAllAsRead(ctx context.Context, userID string) (*notificationv1.UnreadCountResponse, error) {
	if err := s.repo.MarkAllAsRead(ctx, userID); err != nil {
		return nil, err
	}
	return &notificationv1.UnreadCountResponse{Count: 0}, nil
}

func (s *Service) ProcessDeadlineReminders(ctx context.Context, now time.Time) error {
	upcoming, err := s.repo.ListUpcomingDeadlineTasks(ctx, now, now.Add(72*time.Hour))
	if err != nil {
		return err
	}
	for _, task := range upcoming {
		if err := s.createDeadlineReminder(ctx, task, "task_deadline_upcoming", "deadline_upcoming", notificationv1.NotificationPriority_NOTIFICATION_PRIORITY_WARNING); err != nil {
			return err
		}
	}

	overdue, err := s.repo.ListOverdueTasks(ctx, now)
	if err != nil {
		return err
	}
	for _, task := range overdue {
		if err := s.createDeadlineReminder(ctx, task, "task_deadline_overdue", "deadline_overdue", notificationv1.NotificationPriority_NOTIFICATION_PRIORITY_CRITICAL); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) ProcessProjectEvent(ctx context.Context, eventType, eventKey string, event map[string]interface{}, payload []byte) error {
	switch eventType {
	case "task.created":
		return s.handleTaskCreated(ctx, eventType, eventKey, event, payload)
	case "task.assigned":
		return s.handleTaskAssigned(ctx, eventType, eventKey, event, payload)
	case "task.deadline_changed":
		return s.handleDeadlineChanged(ctx, eventType, eventKey, event, payload)
	case "task.status_changed":
		return s.handleStatusChanged(ctx, eventType, eventKey, event, payload)
	case "project.member_added":
		return s.handleProjectMemberAdded(ctx, eventType, eventKey, event, payload)
	default:
		return nil
	}
}

func (s *Service) handleTaskCreated(ctx context.Context, eventType, eventKey string, event map[string]interface{}, payload []byte) error {
	taskID, _ := event["task_id"].(string)
	projectID, _ := event["project_id"].(string)
	assigneeID, _ := event["user_id"].(string)
	projectName, _ := event["project_name"].(string)
	taskTitle, _ := event["task_title"].(string)
	if taskTitle == "" {
		taskTitle, _ = event["title"].(string)
	}

	deadlineStr, _ := event["deadline"].(string)
	var status int32
	if v, ok := event["status"].(float64); ok {
		status = int32(v)
	}

	if err := s.repo.UpsertNotificationTask(ctx, repository.NotificationTask{
		TaskID:         taskID,
		ProjectID:      projectID,
		AssigneeUserID: assigneeID,
		TaskTitle:      taskTitle,
		ProjectName:    projectName,
		Deadline:       util.ParseTime(deadlineStr),
		Status:         status,
	}); err != nil {
		return err
	}

	if assigneeID == "" {
		return nil
	}

	return s.saveNotification(ctx, saveNotificationParams{
		recipientUserID:  assigneeID,
		notificationType: "task_created",
		priority:         notificationv1.NotificationPriority_NOTIFICATION_PRIORITY_INFO,
		title:            "task_created",
		message:          util.NotificationMessage(event, nil),
		actionURL:        util.TaskURL(taskID),
		sourceEventType:  eventType,
		sourceEventKey:   eventKey,
		event:            event,
		payload:          payload,
	})
}

func (s *Service) handleTaskAssigned(ctx context.Context, eventType, eventKey string, event map[string]interface{}, payload []byte) error {
	taskID, _ := event["task_id"].(string)
	assigneeID, _ := event["to_user_id"].(string)

	if err := s.repo.UpdateTaskAssignee(ctx, taskID, assigneeID); err != nil {
		return err
	}
	if assigneeID == "" {
		return nil
	}

	return s.saveNotification(ctx, saveNotificationParams{
		recipientUserID:  assigneeID,
		notificationType: "task_assigned",
		priority:         notificationv1.NotificationPriority_NOTIFICATION_PRIORITY_INFO,
		title:            "task_assigned",
		message:          util.NotificationMessage(event, nil),
		actionURL:        util.TaskURL(taskID),
		sourceEventType:  eventType,
		sourceEventKey:   eventKey,
		event:            event,
		payload:          payload,
	})
}

func (s *Service) handleDeadlineChanged(ctx context.Context, eventType, eventKey string, event map[string]interface{}, payload []byte) error {
	taskID, _ := event["task_id"].(string)
	newDeadline, _ := event["new_deadline"].(string)
	assigneeID, _ := event["assignee_user_id"].(string)
	projectName, _ := event["project_name"].(string)
	taskTitle, _ := event["task_title"].(string)
	if taskTitle == "" {
		taskTitle, _ = event["title"].(string)
	}

	if err := s.repo.UpdateTaskDeadline(ctx, taskID, util.ParseTime(newDeadline), assigneeID, taskTitle, projectName); err != nil {
		return err
	}
	if assigneeID == "" {
		return nil
	}

	return s.saveNotification(ctx, saveNotificationParams{
		recipientUserID:  assigneeID,
		notificationType: "task_deadline_changed",
		priority:         notificationv1.NotificationPriority_NOTIFICATION_PRIORITY_WARNING,
		title:            "task_deadline_changed",
		message:          util.NotificationMessage(event, map[string]interface{}{"new_deadline": newDeadline}),
		actionURL:        util.TaskURL(taskID),
		sourceEventType:  eventType,
		sourceEventKey:   eventKey,
		event:            event,
		payload:          payload,
	})
}

func (s *Service) handleStatusChanged(ctx context.Context, eventType, eventKey string, event map[string]interface{}, payload []byte) error {
	taskID, _ := event["task_id"].(string)
	assigneeID, _ := event["assignee_user_id"].(string)
	projectName, _ := event["project_name"].(string)
	taskTitle, _ := event["task_title"].(string)
	if taskTitle == "" {
		taskTitle, _ = event["title"].(string)
	}

	var toStatus int32
	if v, ok := event["to_status"].(float64); ok {
		toStatus = int32(v)
	}

	var completedAt *time.Time
	if toStatus == 3 {
		occurredAt, _ := event["occurred_at"].(string)
		completedAt = util.ParseTime(occurredAt)
		if completedAt == nil {
			now := time.Now().UTC()
			completedAt = &now
		}
	}

	if err := s.repo.UpdateTaskStatus(ctx, taskID, toStatus, completedAt, taskTitle, projectName); err != nil {
		return err
	}
	if assigneeID == "" {
		return nil
	}

	priority := notificationv1.NotificationPriority_NOTIFICATION_PRIORITY_INFO
	if toStatus == 4 {
		priority = notificationv1.NotificationPriority_NOTIFICATION_PRIORITY_WARNING
	}

	return s.saveNotification(ctx, saveNotificationParams{
		recipientUserID:  assigneeID,
		notificationType: "task_status_changed",
		priority:         priority,
		title:            "task_status_changed",
		message:          util.NotificationMessage(event, map[string]interface{}{"to_status": int(toStatus)}),
		actionURL:        util.TaskURL(taskID),
		sourceEventType:  eventType,
		sourceEventKey:   eventKey,
		event:            event,
		payload:          payload,
	})
}

func (s *Service) handleProjectMemberAdded(ctx context.Context, eventType, eventKey string, event map[string]interface{}, payload []byte) error {
	userID, _ := event["user_id"].(string)
	projectID, _ := event["project_id"].(string)
	if userID == "" {
		return nil
	}

	return s.saveNotification(ctx, saveNotificationParams{
		recipientUserID:  userID,
		notificationType: "project_member_added",
		priority:         notificationv1.NotificationPriority_NOTIFICATION_PRIORITY_INFO,
		title:            "project_member_added",
		message:          util.NotificationMessage(event, nil),
		actionURL:        util.ProjectURL(projectID),
		sourceEventType:  eventType,
		sourceEventKey:   eventKey,
		event:            event,
		payload:          payload,
	})
}

func (s *Service) createDeadlineReminder(ctx context.Context, task repository.NotificationTask, notificationType, keyPrefix string, priority notificationv1.NotificationPriority) error {
	if task.AssigneeUserID == "" || task.TaskID == "" || task.Deadline == nil {
		return nil
	}

	event := map[string]interface{}{
		"event_type":    notificationType,
		"task_id":       task.TaskID,
		"project_id":    task.ProjectID,
		"task_title":    task.TaskTitle,
		"project_name":  task.ProjectName,
		"deadline":      task.Deadline.UTC().Format(time.RFC3339Nano),
		"actor_user_id": "",
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}

	deadlineDay := task.Deadline.UTC().Format("2006-01-02")
	sourceEventKey := fmt.Sprintf("%s:%s:%s", keyPrefix, task.TaskID, deadlineDay)
	deadline, _ := event["deadline"].(string)

	return s.saveNotification(ctx, saveNotificationParams{
		recipientUserID:  task.AssigneeUserID,
		notificationType: notificationType,
		priority:         priority,
		title:            notificationType,
		message:          util.NotificationMessage(event, map[string]interface{}{"deadline": deadline}),
		actionURL:        util.TaskURL(task.TaskID),
		sourceEventType:  notificationType,
		sourceEventKey:   sourceEventKey,
		event:            event,
		payload:          payload,
	})
}

type saveNotificationParams struct {
	recipientUserID  string
	notificationType string
	priority         notificationv1.NotificationPriority
	title            string
	message          string
	actionURL        string
	sourceEventType  string
	sourceEventKey   string
	event            map[string]interface{}
	payload          []byte
}

func (s *Service) saveNotification(ctx context.Context, p saveNotificationParams) error {
	actorUserID, _ := p.event["actor_user_id"].(string)
	if actorUserID != "" && actorUserID == p.recipientUserID {
		return nil
	}

	projectID, _ := p.event["project_id"].(string)
	taskID, _ := p.event["task_id"].(string)

	return s.repo.CreateNotification(ctx, repository.CreateNotificationParams{
		RecipientUserID: p.recipientUserID,
		Type:            p.notificationType,
		Priority:        int32(p.priority),
		Title:           p.title,
		Message:         p.message,
		ProjectID:       projectID,
		TaskID:          taskID,
		ActorUserID:     actorUserID,
		ActionURL:       p.actionURL,
		SourceEventType: p.sourceEventType,
		SourceEventKey:  p.sourceEventKey,
		Payload:         p.payload,
	})
}
