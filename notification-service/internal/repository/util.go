package repository

import (
	"database/sql"
	"errors"

	"github.com/lib/pq"
)

type rowScanner interface {
	Scan(dest ...interface{}) error
}

func nullIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func scanNotification(s rowScanner) (Notification, error) {
	var n Notification
	var readAt sql.NullTime
	err := s.Scan(
		&n.ID,
		&n.RecipientUserID,
		&n.Type,
		&n.Priority,
		&n.Title,
		&n.Message,
		&n.ProjectID,
		&n.TaskID,
		&n.ActorUserID,
		&n.ActionURL,
		&n.SourceEventType,
		&n.SourceEventKey,
		&n.Payload,
		&readAt,
		&n.CreatedAt,
	)
	if err != nil {
		return Notification{}, err
	}
	if readAt.Valid {
		n.ReadAt = &readAt.Time
	}
	return n, nil
}

func IsUniqueViolation(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr.Code == "23505"
}
