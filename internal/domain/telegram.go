package domain

import (
	"time"
)

type NotificationType string

const (
	NotificationTypeSuitable NotificationType = "suitable"
	NotificationTypeDaily    NotificationType = "daily"
)

type NotificationStatus string

const (
	NotificationStatusPending NotificationStatus = "pending"
	NotificationStatusSent    NotificationStatus = "sent"
	NotificationStatusFailed  NotificationStatus = "failed"
)

type TelegramNotification struct {
	ID                int64
	ProjectID         int64
	EvaluationID      int64
	ProposalDraftID   int64
	NotificationType  NotificationType
	ChatID            int64
	TelegramMessageID int64
	Status            NotificationStatus
	Error             string
	CreatedAt         time.Time
	SentAt            time.Time
}
