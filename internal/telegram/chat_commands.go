package telegram

import (
	"context"
	"fmt"
	"strings"
)

type ChatOrchestrator interface {
	Run(ctx context.Context) (int, error)
}

// Add a reference to the chat orchestrator in Bot
func (b *Bot) SetChatOrchestrator(orch ChatOrchestrator) {
	b.chatOrch = orch
}

func (b *Bot) handleChatCommand(ctx context.Context, cmd string) string {
	switch cmd {
	case "sync":
		return b.handleSyncChat(ctx)
	case "dialogs":
		return b.handleDialogs(ctx)
	case "unread":
		return b.handleUnread(ctx)
	case "orders":
		return b.handleOrders(ctx)
	default:
		return ""
	}
}

func (b *Bot) handleOrders(ctx context.Context) string {
	projects, err := b.db.GetSuitableProjects(ctx, b.cfg.KworkSuitableScore, 10)
	if err != nil {
		return "Ошибка БД: " + err.Error()
	}

	if len(projects) == 0 {
		return fmt.Sprintf("Сейчас нет подходящих заказов с оценкой ≥%d.", b.cfg.KworkSuitableScore)
	}

	for _, p := range projects {
		proj, err := b.db.GetProject(ctx, p.ID)
		if err != nil || proj == nil {
			continue
		}
		eval, err := b.db.GetEvaluationByExternalID(ctx, proj.Source, proj.ExternalID)
		if err != nil || eval == nil {
			continue
		}
		b.SendSuitableProjectNotification(proj, eval)
	}
	
	return ""
}

func (b *Bot) handleSyncChat(ctx context.Context) string {
	if b.chatOrch == nil {
		return "Chat sync is not enabled."
	}
	
	count, err := b.chatOrch.Run(ctx)
	if err != nil {
		return fmt.Sprintf("❌ Ошибка синхронизации: %v", err)
	}
	
	return fmt.Sprintf("✅ Синхронизация завершена. Новых сообщений: %d", count)
}

func (b *Bot) handleDialogs(ctx context.Context) string {
	// Let's list recent active dialogs from DB
	query := `SELECT counterparty_username, unread_count FROM kwork_conversations ORDER BY last_message_at DESC LIMIT 10`
	rows, err := b.db.QueryContext(ctx, query)
	if err != nil {
		return "Ошибка БД: " + err.Error()
	}
	defer rows.Close()

	var sb strings.Builder
	sb.WriteString("<b>Последние диалоги:</b>\n\n")
	found := false
	for rows.Next() {
		found = true
		var user string
		var unread int
		if err := rows.Scan(&user, &unread); err != nil {
			continue
		}
		if unread > 0 {
			sb.WriteString(fmt.Sprintf("📬 <b>%s</b> (%d новых)\n", user, unread))
		} else {
			sb.WriteString(fmt.Sprintf("💬 %s\n", user))
		}
	}
	if !found {
		return "Диалогов пока нет."
	}
	return sb.String()
}

func (b *Bot) handleUnread(ctx context.Context) string {
	query := `SELECT sender_username, text FROM kwork_messages 
	          WHERE direction = 'incoming' AND conversation_id IN 
	          (SELECT id FROM kwork_conversations WHERE unread_count > 0)
			  ORDER BY sent_at DESC LIMIT 5`
			  
	rows, err := b.db.QueryContext(ctx, query)
	if err != nil {
		return "Ошибка БД: " + err.Error()
	}
	defer rows.Close()

	var sb strings.Builder
	sb.WriteString("<b>Непрочитанные (последние 5):</b>\n\n")
	found := false
	for rows.Next() {
		found = true
		var user, text string
		if err := rows.Scan(&user, &text); err != nil {
			continue
		}
		if len(text) > 100 {
			text = text[:100] + "..."
		}
		sb.WriteString(fmt.Sprintf("📩 <b>%s</b>: %s\n", user, text))
	}
	if !found {
		return "Нет непрочитанных сообщений."
	}
	return sb.String()
}
