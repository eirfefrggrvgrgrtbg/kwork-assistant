package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"kwork-assistant/internal/database"
)

type Config struct {
	Token       string
	OwnerChatID int64
}

type Bot struct {
	api    *tgbotapi.BotAPI
	cfg    Config
	db     *database.DB
	logger *slog.Logger
}

func NewBot(cfg Config, db *database.DB, logger *slog.Logger) (*Bot, error) {
	if cfg.Token == "" {
		return nil, fmt.Errorf("telegram token is empty")
	}
	api, err := tgbotapi.NewBotAPI(cfg.Token)
	if err != nil {
		return nil, err
	}
	
	return &Bot{
		api:    api,
		cfg:    cfg,
		db:     db,
		logger: logger.With("component", "telegram_bot"),
	}, nil
}

func (b *Bot) Health() error {
	_, err := b.api.GetMe()
	return err
}

func (b *Bot) SendMessage(chatID int64, text string) (int64, error) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeHTML
	msg.DisableWebPagePreview = true

	sentMsg, err := b.api.Send(msg)
	if err != nil {
		return 0, err
	}
	return int64(sentMsg.MessageID), nil
}

func (b *Bot) GetOwnerChatID() int64 {
	return b.cfg.OwnerChatID
}

func (b *Bot) SendOwnerMessage(text string) (int64, error) {
	if b.cfg.OwnerChatID == 0 {
		return 0, fmt.Errorf("owner chat ID is not set")
	}
	return b.SendMessage(b.cfg.OwnerChatID, text)
}

func (b *Bot) SendMessageWithButton(chatID int64, text string, buttonText, url string) (int64, error) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeHTML
	msg.DisableWebPagePreview = true
	
	if url != "" {
		btn := tgbotapi.NewInlineKeyboardButtonURL(buttonText, url)
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(btn),
		)
		msg.ReplyMarkup = keyboard
	}

	sentMsg, err := b.api.Send(msg)
	if err != nil {
		return 0, err
	}
	return int64(sentMsg.MessageID), nil
}

func (b *Bot) StartPolling(ctx context.Context) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.api.GetUpdatesChan(u)

	for {
		select {
		case <-ctx.Done():
			b.logger.Info("Stopping telegram polling")
			b.api.StopReceivingUpdates()
			return
		case update := <-updates:
			if update.Message != nil {
				b.handleMessage(ctx, update.Message)
			}
		}
	}
}

func (b *Bot) handleMessage(ctx context.Context, msg *tgbotapi.Message) {
	if msg.Chat.ID != b.cfg.OwnerChatID {
		b.logger.Warn("Unauthorized access attempt", "chat_id", msg.Chat.ID, "username", msg.From.UserName)
		b.SendMessage(msg.Chat.ID, "Access denied.")
		return
	}

	if !msg.IsCommand() {
		return
	}

	cmd := msg.Command()
	var response string

	switch cmd {
	case "status":
		response = b.handleStatus(ctx)
	case "today":
		response = b.handleToday(ctx)
	case "stats":
		response = b.handleStats(ctx)
	case "recent":
		response = b.handleRecent(ctx)
	case "skipped":
		response = b.handleSkipped(ctx)
	case "pause":
		response = b.handlePause(ctx)
	case "resume":
		response = b.handleResume(ctx)
	default:
		response = "Неизвестная команда."
	}

	if response != "" {
		b.SendMessage(msg.Chat.ID, response)
	}
}

func (b *Bot) handleStatus(ctx context.Context) string {
	var sb strings.Builder
	sb.WriteString("<b>Kwork Assistant</b>\n\n")
	
	// Assuming external health checks are passed since we are running, or we just display simple status
	sb.WriteString("TELEGRAM: ✅\n")
	
	autoProcessing, _ := b.db.GetAppMeta(ctx, "auto_processing_enabled")
	if autoProcessing == "true" {
		sb.WriteString("AUTO PROCESSING: ▶️ Включено\n")
	} else {
		sb.WriteString("AUTO PROCESSING: ⏸️ Выключено\n")
	}
	
	return sb.String()
}

func (b *Bot) handlePause(ctx context.Context) string {
	err := b.db.SetAppMeta(ctx, "auto_processing_enabled", "false")
	if err != nil {
		return "Ошибка при сохранении статуса: " + err.Error()
	}
	return "⏸️ Автоматическая обработка остановлена."
}

func (b *Bot) handleResume(ctx context.Context) string {
	err := b.db.SetAppMeta(ctx, "auto_processing_enabled", "true")
	if err != nil {
		return "Ошибка при сохранении статуса: " + err.Error()
	}
	return "▶️ Автоматическая обработка запущена."
}

// Stubs for stats, implement via simple SQL count later
func (b *Bot) handleToday(ctx context.Context) string {
	return "Сегодня статистика в разработке..."
}

func (b *Bot) handleStats(ctx context.Context) string {
	return "Общая статистика в разработке..."
}

func (b *Bot) handleRecent(ctx context.Context) string {
	return "Последние проекты в разработке..."
}

func (b *Bot) handleSkipped(ctx context.Context) string {
	return "Пропущенные проекты в разработке..."
}
