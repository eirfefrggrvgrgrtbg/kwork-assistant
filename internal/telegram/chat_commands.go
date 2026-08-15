package telegram

import (
	"context"
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
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

	var sb strings.Builder
	sb.WriteString("🔥 Подходящие заказы\n\n")
	
	var kbRows [][]tgbotapi.InlineKeyboardButton

	for _, p := range projects {
		budgetStr := "Бюджет не указан"
		if p.BudgetFrom.Valid && p.BudgetTo.Valid && p.BudgetFrom.Float64 > 0 && p.BudgetTo.Float64 > 0 && p.BudgetFrom.Float64 != p.BudgetTo.Float64 {
			budgetStr = fmt.Sprintf("💰 %.0f–%.0f ₽", p.BudgetFrom.Float64, p.BudgetTo.Float64)
		} else {
			val := p.BudgetTo.Float64
			if val == 0 && p.BudgetFrom.Valid {
				val = p.BudgetFrom.Float64
			}
			if val > 0 {
				budgetStr = fmt.Sprintf("💰 %.0f ₽", val)
			}
		}

		sb.WriteString(fmt.Sprintf("[%d] %s\n%s\n%s\n\n", p.Score, strings.ToUpper(p.Category), p.Title, budgetStr))
		
		btn := tgbotapi.NewInlineKeyboardButtonData(
			"📝 Сгенерировать отклик", 
			fmt.Sprintf("generate_proposal:%d", p.ID),
		)
		kbRows = append(kbRows, tgbotapi.NewInlineKeyboardRow(btn))
	}
	
	if len(projects) == 0 {
		return fmt.Sprintf("Сейчас нет подходящих заказов с оценкой ≥%d.", b.cfg.KworkSuitableScore)
	}

	if b.api != nil {
		msg := tgbotapi.NewMessage(b.cfg.TelegramOwnerChatID, sb.String())
		msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(kbRows...)
		b.api.Send(msg)
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
