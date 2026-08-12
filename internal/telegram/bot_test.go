package telegram

import (
	"context"
	"strings"
	"testing"
	"kwork-assistant/internal/database"
)

func setupTestDB(t *testing.T) *database.DB {
	db, err := database.InitDB(":memory:")
	if err != nil {
		t.Fatalf("Failed to init in-memory db: %v", err)
	}
	return db
}

func TestBotHandleCommands(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// 1. Initial State
	ctx := context.Background()
	val, _ := db.GetAppMeta(ctx, "auto_processing_enabled")
	if val != "false" && val != "" {
		t.Errorf("Expected initial auto_processing_enabled to be empty/false, got %v", val)
	}

	bot := &Bot{
		cfg: Config{OwnerChatID: 12345},
		db:  db,
	}

	// 2. Test handleResume
	res := bot.handleResume(ctx)
	if !strings.Contains(res, "запущена") {
		t.Errorf("Unexpected resume response: %v", res)
	}
	val, _ = db.GetAppMeta(ctx, "auto_processing_enabled")
	if val != "true" {
		t.Errorf("Expected auto_processing_enabled to be true, got %v", val)
	}

	// 3. Test handlePause
	res = bot.handlePause(ctx)
	if !strings.Contains(res, "остановлена") {
		t.Errorf("Unexpected pause response: %v", res)
	}
	val, _ = db.GetAppMeta(ctx, "auto_processing_enabled")
	if val != "false" {
		t.Errorf("Expected auto_processing_enabled to be false, got %v", val)
	}
}
