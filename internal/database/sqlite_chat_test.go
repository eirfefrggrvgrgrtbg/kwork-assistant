package database

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"kwork-assistant/internal/domain"
)

// TestSaveKworkMessage_Dedup verifies that:
//  1. First save of an external_message_id returns true (created).
//  2. Second save of the same external_message_id returns false (dedup).
//  3. A third call (simulating application restart + same payload) still returns false.
//
// This is a regression test for the SQLite LastInsertId bug where ON CONFLICT DO NOTHING
// would still return a non-zero rowid from the last session insert.
func TestSaveKworkMessage_Dedup(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := InitDB(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	defer db.Close()
	ctx := context.Background()

	// Need a conversation row first (FK).
	conv := &domain.KworkConversation{
		CounterpartyUserID:   999,
		CounterpartyUsername: "Support",
		LastMessageAt:        time.Now(),
	}
	if err := db.UpsertKworkConversation(ctx, conv); err != nil {
		t.Fatalf("UpsertKworkConversation: %v", err)
	}

	msg := &domain.KworkMessage{
		ConversationID:    conv.ID,
		ExternalMessageID: 406705212,
		SenderUserID:      1,
		SenderUsername:    "Support",
		Direction:         domain.DirectionIncoming,
		Text:              "Здравствуйте. Открепили данный ИНН.",
		SentAt:            time.Unix(1786300854, 0),
		RawHash:           "abc",
	}

	// sync #1 — must be new
	created, err := db.SaveKworkMessage(ctx, msg)
	if err != nil {
		t.Fatalf("sync1 SaveKworkMessage: %v", err)
	}
	if !created {
		t.Error("sync1: expected created=true for first insert")
	}

	// sync #2 — same payload, same session → must NOT be new
	msg2 := *msg
	msg2.ID = 0
	created2, err := db.SaveKworkMessage(ctx, &msg2)
	if err != nil {
		t.Fatalf("sync2 SaveKworkMessage: %v", err)
	}
	if created2 {
		t.Error("sync2: expected created=false for duplicate (same session)")
	}

	// sync #3 — simulate restart: reopen DB, same external_message_id
	db.Close()
	db2, err := InitDB(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatalf("InitDB2: %v", err)
	}
	defer db2.Close()

	// Insert an unrelated message first to advance last_insert_rowid
	unrelated := &domain.KworkMessage{
		ConversationID:    conv.ID,
		ExternalMessageID: 999999,
		SenderUserID:      1,
		SenderUsername:    "Support",
		Direction:         domain.DirectionOutgoing,
		Text:              "ok",
		SentAt:            time.Now(),
		RawHash:           "xyz",
	}
	db2.SaveKworkMessage(ctx, unrelated)

	msg3 := *msg
	msg3.ID = 0
	created3, err := db2.SaveKworkMessage(ctx, &msg3)
	if err != nil {
		t.Fatalf("sync3 SaveKworkMessage: %v", err)
	}
	if created3 {
		t.Error("sync3 (after restart): expected created=false — duplicate must not be re-sent to Telegram")
	}
}
