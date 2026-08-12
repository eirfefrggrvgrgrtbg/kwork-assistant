package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"kwork-assistant/internal/config"
)

func runKworkDialogs() {
	cfg, _ := config.Load()
	src, err := getKworkSource(cfg)
	if err != nil {
		fmt.Printf("Error configuring Kwork source: %v\n", err)
		os.Exit(1)
	}
	ctx := context.Background()

	dialogs, err := src.FetchDialogs(ctx)
	if err != nil {
		slog.Error("Failed to fetch dialogs", "error", err)
		os.Exit(1)
	}

	fmt.Printf("--- DIALOGS (%d) ---\n", len(dialogs))
	for _, d := range dialogs {
		fmt.Printf("User: %s (ID: %d)\n", d.Username, d.UserID)
		fmt.Printf("  Unread: %d\n", d.UnreadCount)
		fmt.Printf("  Last message snippet: %s\n", d.LastMessage)
		if d.LastMessageObj != nil {
			direction := "INCOMING"
			if d.LastMessageObj.FromUsername != d.Username {
				direction = "OUTGOING (ME)"
			}
			fmt.Printf("  Last message direction: %s\n", direction)
			fmt.Printf("  Last message time: %d\n", d.LastMessageObj.Time)
		}
		fmt.Println("--------------------")
	}
}

func runKworkDialogMessages(username string) {
	cfg, _ := config.Load()
	src, err := getKworkSource(cfg)
	if err != nil {
		fmt.Printf("Error configuring Kwork source: %v\n", err)
		os.Exit(1)
	}
	ctx := context.Background()

	messages, err := src.FetchDialogMessages(ctx, username)
	if err != nil {
		slog.Error("Failed to fetch messages", "username", username, "error", err)
		os.Exit(1)
	}

	fmt.Printf("--- MESSAGES WITH %s (%d) ---\n", username, len(messages))
	for _, m := range messages {
		direction := "INCOMING"
		sender := m.FromUsername
		if m.FromUsername != username {
			direction = "OUTGOING"
			sender = "ME"
		}
		fmt.Printf("[%s] [%s] ID:%d Time:%d\n", direction, sender, m.MessageID, m.Time)
		fmt.Printf("Text: %s\n", m.Message)
		fmt.Println("--------------------")
	}
}
