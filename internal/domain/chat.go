package domain

import (
	"database/sql"
	"time"
)

type Direction string

const (
	DirectionIncoming Direction = "incoming"
	DirectionOutgoing Direction = "outgoing"
)

type KworkConversation struct {
	ID int64

	CounterpartyUserID      int64
	CounterpartyUsername    string
	CounterpartyDisplayName string

	ProjectID sql.NullInt64 // nullable

	LastExternalMessageID int64
	LastMessageAt         time.Time

	UnreadCount int

	CreatedAt time.Time
	UpdatedAt time.Time
}

type KworkMessage struct {
	ID int64

	ConversationID int64

	ExternalMessageID int64 // if possible

	SenderUserID   int64
	SenderUsername string

	Direction Direction
	Text      string

	SentAt time.Time

	RawHash string // for fallback deduplication

	CreatedAt time.Time
}
