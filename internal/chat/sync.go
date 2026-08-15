package chat

import (
	"context"
	"fmt"
	"log/slog"

	"kwork-assistant/internal/config"
	"kwork-assistant/internal/database"
	"kwork-assistant/internal/telegram"
)

type SyncOrchestrator struct {
	cfg     *config.Config
	db      *database.DB
	service *Service
	bot     *telegram.Bot
}

func NewSyncOrchestrator(cfg *config.Config, db *database.DB, service *Service, bot *telegram.Bot) *SyncOrchestrator {
	return &SyncOrchestrator{
		cfg:     cfg,
		db:      db,
		service: service,
		bot:     bot,
	}
}

// Run executes a single sync pass for Kwork dialogs and messages.
// Returns the number of newly discovered *incoming* messages.
func (o *SyncOrchestrator) Run(ctx context.Context) (int, error) {
	if !o.cfg.KworkChatSyncEnabled {
		return 0, nil
	}

	// Determine if this is a baseline sync
	var count int
	err := o.db.QueryRowContext(ctx, "SELECT count(*) FROM kwork_conversations").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count conversations: %w", err)
	}

	isBaseline := count == 0

	if isBaseline {
		slog.Info("Starting baseline chat sync. No Telegram notifications will be sent for existing history.")
	}

	newMsgs, err := o.service.SyncConversations(ctx)
	if err != nil {
		return 0, fmt.Errorf("chat sync failed: %w", err)
	}

	if isBaseline {
		slog.Info("Baseline chat sync completed.", "total_messages_saved", len(newMsgs))
		return 0, nil // return 0 new messages to notify
	}

	// Filter for incoming messages only, since we don't need to notify about our own sent messages.
	var incomingCount int
	for _, m := range newMsgs {
		if string(m.Direction) == "incoming" {
			if o.bot != nil {
				_, err := o.bot.SendChatMessageNotification(m.SenderUsername, m.Text, m.ConversationID, m.ID)
				if err != nil {
					slog.Error("Failed to send chat notification", "username", m.SenderUsername, "error", err)
				}
			}
			incomingCount++
		}
	}

	if incomingCount > 0 {
		slog.Info("Chat sync completed with new incoming messages", "new_incoming", incomingCount)
	}

	return incomingCount, nil
}
