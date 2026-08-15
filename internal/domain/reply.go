package domain

import (
	"database/sql"
	"time"
)

const (
	ReplyDraftStatusGenerated = "generated"
	ReplyDraftStatusApproved  = "approved"
	ReplyDraftStatusRejected  = "rejected"
	ReplyDraftStatusSent      = "sent"
)

type ReplyDraft struct {
	ID int64

	ConversationID int64

	ContextMessageID sql.NullInt64 // The message ID this draft is replying to

	ModelVersion string
	DraftText    string

	Status string // "pending", "approved", "rejected", "sent"

	SentMessageID sql.NullInt64 // If it was successfully sent and we detected it, or if user sent it.

	CreatedAt time.Time
	UpdatedAt time.Time
}
