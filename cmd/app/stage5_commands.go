package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	
	"kwork-assistant/internal/ai"
	"kwork-assistant/internal/config"
	"kwork-assistant/internal/database"
	"kwork-assistant/internal/evaluation"
	"kwork-assistant/internal/pipeline"
	"kwork-assistant/internal/projectwatch"
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
	bot, err := telegram.NewBot(cfg, db, slog.Default())
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
	bot, err := telegram.NewBot(cfg, db, slog.Default())
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

func runKworkProjectWatch(ctx context.Context, cfg *config.Config, db *database.DB, aiClient ai.AIClient) {
	fmt.Println("Running Project Watcher One-Shot...")

	bot, err := telegram.NewBot(cfg, db, slog.Default())
	if err != nil {
		fmt.Printf("Failed to init telegram bot: %v\n", err)
		os.Exit(1)
	}

	kworkSrc, err := getKworkSource(cfg)
	if err != nil {
		fmt.Printf("Failed to init Kwork source: %v\n", err)
		os.Exit(1)
	}
	if kworkSrc == nil {
		fmt.Println("Kwork credentials missing")
		os.Exit(1)
	}

	evaluator := evaluation.NewEvaluator(aiClient, cfg.OllamaModel, "evaluation-v2")
	
	// Force enable for this one-shot run
	cfg.KworkProjectWatchEnabled = true
	if cfg.KworkPollLimit == 0 {
		cfg.KworkPollLimit = 50
	}
	
	pWatcher := projectwatch.NewWatcher(cfg, db, kworkSrc, evaluator, cfg.OllamaModel, "evaluation-v2", bot, slog.Default())
	
	stats, err := pWatcher.Run(ctx)
	if err != nil {
		fmt.Printf("Project watch failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nProjects fetched: %d\n", stats.ProjectsFetched)
	fmt.Printf("New projects: %d\n", stats.NewProjects)
	fmt.Printf("Already known: %d\n", stats.AlreadyKnown)
	fmt.Printf("Evaluated: %d\n", stats.Evaluated)
	fmt.Printf("Skipped evaluation: %d\n", stats.SkippedEvaluation)
	fmt.Printf("Website: %d\n", stats.Website)
	fmt.Printf("Telegram: %d\n", stats.Telegram)
	fmt.Printf("Skip: %d\n", stats.Skip)
	fmt.Printf("Suitable >= %d: %d\n", cfg.KworkSuitableScore, stats.SuitableAbove80)
	fmt.Printf("Telegram notifications sent: %d\n", stats.TelegramNotifsSent)
	fmt.Printf("Telegram duplicates skipped: %d\n", stats.TelegramDupesSkipped)
	fmt.Printf("Proposal generations: %d\n", stats.ProposalGenerations)
}
