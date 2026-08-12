package main

import (
	"context"
	"fmt"
	"log/slog"
	
	"kwork-assistant/internal/ai"
	"kwork-assistant/internal/config"
	"kwork-assistant/internal/database"
	"kwork-assistant/internal/pipeline"
	"kwork-assistant/internal/telegram"
)

func runEmailFetch(ctx context.Context, cfg *config.Config, db *database.DB) {
	fmt.Println("Not implemented yet.")
}

func runEmailList(ctx context.Context, db *database.DB) {
	fmt.Println("Not implemented yet.")
}

func runEmailShow(ctx context.Context, db *database.DB, idStr string) {
	fmt.Println("Not implemented yet.")
}

func runEmailParse(ctx context.Context, db *database.DB, idStr string) {
	fmt.Println("Not implemented yet.")
}

func runTelegramTest(ctx context.Context, cfg *config.Config, db *database.DB) {
	if cfg.TelegramBotToken == "" || cfg.TelegramOwnerChatID == 0 {
		fmt.Println("TELEGRAM_BOT_TOKEN or TELEGRAM_OWNER_CHAT_ID is missing")
		return
	}
	bot, err := telegram.NewBot(telegram.Config{
		Token:       cfg.TelegramBotToken,
		OwnerChatID: cfg.TelegramOwnerChatID,
	}, db, slog.Default())
	if err != nil {
		fmt.Printf("Failed to init bot: %v\n", err)
		return
	}
	_, err = bot.SendOwnerMessage("✅ Kwork Assistant подключен.")
	if err != nil {
		fmt.Printf("Failed to send test message: %v\n", err)
		return
	}
	fmt.Println("Test message sent successfully.")
}

func runTelegramRetry(ctx context.Context, cfg *config.Config, db *database.DB) {
	fmt.Println("Not implemented yet.")
}

func runProcessPending(ctx context.Context, cfg *config.Config, db *database.DB, aiClient ai.AIClient, limit int) {
	bot, err := telegram.NewBot(telegram.Config{
		Token:       cfg.TelegramBotToken,
		OwnerChatID: cfg.TelegramOwnerChatID,
	}, db, slog.Default())
	if err != nil {
		slog.Error("Failed to init bot for pipeline", "error", err)
	}

	pipelineCfg := pipeline.Config{
		Model:          cfg.OllamaModel,
		SystemPrompt:   "v2",
		ProposalPrompt: "v3",
	}

	svc := pipeline.NewService(db, aiClient, bot, pipelineCfg, slog.Default())
	err = svc.ProcessPending(ctx, limit)
	if err != nil {
		slog.Error("Pipeline error", "error", err)
	} else {
		slog.Info("Pipeline completed")
	}
}
