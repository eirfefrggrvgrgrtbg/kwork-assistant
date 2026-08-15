package chat

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"time"

	"kwork-assistant/internal/database"
	"kwork-assistant/internal/domain"
	"kwork-assistant/internal/kwork"
)

type Service struct {
	db     *database.DB
	source *kwork.KworkProjectSource
}

func NewService(db *database.DB, source *kwork.KworkProjectSource) *Service {
	return &Service{
		db:     db,
		source: source,
	}
}

// SyncConversations fetches the latest dialogs from Kwork and upserts them.
// It also fetches messages for active dialogs.
func (s *Service) SyncConversations(ctx context.Context) ([]domain.KworkMessage, error) {
	dialogs, err := s.source.FetchDialogs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch dialogs: %w", err)
	}

	var allNewMessages []domain.KworkMessage

	for _, d := range dialogs {
		// Create or update conversation
		conv := &domain.KworkConversation{
			CounterpartyUserID:      int64(d.UserID),
			CounterpartyUsername:    d.Username,
			CounterpartyDisplayName: d.Username, // Kwork API might just have username
			UnreadCount:             d.UnreadCount,
			LastMessageAt:           time.Unix(int64(d.Time), 0),
		}

		if d.LastMessageObj != nil {
			conv.LastExternalMessageID = 0 // we don't have it at the dialog level usually
		}

		if err := s.db.UpsertKworkConversation(ctx, conv); err != nil {
			slog.Error("Failed to upsert conversation", "user", d.Username, "error", err)
			continue
		}

		// Sync messages for this conversation
		newMsgs, err := s.SyncMessagesForConversation(ctx, conv.ID, d.Username)
		if err != nil {
			slog.Error("Failed to sync messages", "user", d.Username, "error", err)
		} else {
			allNewMessages = append(allNewMessages, newMsgs...)
		}
	}
	return allNewMessages, nil
}

// SyncMessagesForConversation fetches recent messages and saves them to DB.
// Returns an error if fetching fails, and a list of newly inserted messages.
func (s *Service) SyncMessagesForConversation(ctx context.Context, convID int64, username string) ([]domain.KworkMessage, error) {
	messages, err := s.source.FetchDialogMessages(ctx, username)
	if err != nil {
		return nil, err
	}

	var newMessages []domain.KworkMessage

	// Reverse iterate to save oldest first, though it's an upsert so order is mostly for logical insert time
	for i := len(messages) - 1; i >= 0; i-- {
		m := messages[i]
		
		direction := domain.DirectionIncoming
		if m.FromUsername != username {
			direction = domain.DirectionOutgoing
		}

		rawHash := computeHash(m.Message, int64(m.Time), m.FromUsername)

		msg := &domain.KworkMessage{
			ConversationID:    convID,
			ExternalMessageID: int64(m.MessageID),
			SenderUserID:      int64(m.FromID),
			SenderUsername:    m.FromUsername,
			Direction:         direction,
			Text:              m.Message,
			SentAt:            time.Unix(int64(m.Time), 0),
			RawHash:           rawHash,
		}

		isNew, err := s.db.SaveKworkMessage(ctx, msg)
		if err != nil {
			slog.Error("Failed to save message", "msgID", m.MessageID, "error", err)
		} else if isNew {
			newMessages = append(newMessages, *msg)
		}
	}

	return newMessages, nil
}

func computeHash(text string, timestamp int64, sender string) string {
	data := fmt.Sprintf("%s|%d|%s", text, timestamp, sender)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}
