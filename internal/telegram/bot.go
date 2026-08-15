package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"kwork-assistant/internal/config"
	"kwork-assistant/internal/database"
	"kwork-assistant/internal/domain"
)

type Bot struct {
	api    *tgbotapi.BotAPI
	cfg    *config.Config
	db     *database.DB
	logger *slog.Logger
	chatOrch ChatOrchestrator
	replyGen ReplyGenerator
	propGen  ProposalGenerator
}

type ReplyGenerator interface {
	GenerateDraft(ctx context.Context, conversationID int64, latestMessageID int64) error
}

type ProposalGenerator interface {
	GenerateDraft(ctx context.Context, projectID int64, forceRegenerate bool) (string, error)
}

func (b *Bot) SetReplyGenerator(gen ReplyGenerator) {
	b.replyGen = gen
}

func (b *Bot) SetProposalGenerator(gen ProposalGenerator) {
	b.propGen = gen
}

func NewBot(cfg *config.Config, db *database.DB, logger *slog.Logger) (*Bot, error) {
	if cfg.TelegramBotToken == "" {
		return nil, fmt.Errorf("telegram token is empty")
	}
	api, err := tgbotapi.NewBotAPI(cfg.TelegramBotToken)
	if err != nil {
		return nil, err
	}
	
	b := &Bot{
		api:    api,
		cfg:    cfg,
		db:     db,
		logger: logger.With("component", "telegram_bot"),
	}

	b.RegisterCommands()
	return b, nil
}

func (b *Bot) RegisterCommands() {
	if b.api == nil {
		return
	}
	commands := []tgbotapi.BotCommand{
		{Command: "start", Description: "Главное меню"},
		{Command: "status", Description: "Состояние системы"},
		{Command: "dialogs", Description: "Диалоги Kwork"},
		{Command: "sync", Description: "Синхронизация"},
		{Command: "unread", Description: "Новые сообщения"},
		{Command: "pause", Description: "Пауза"},
		{Command: "resume", Description: "Продолжить"},
	}
	config := tgbotapi.NewSetMyCommands(commands...)
	b.api.Request(config)
}

func (b *Bot) Health() error {
	_, err := b.api.GetMe()
	return err
}

func (b *Bot) SendMessage(chatID int64, text string) (int64, error) {
	if b.api == nil {
		return 0, nil // For tests
	}
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
	return b.cfg.TelegramOwnerChatID
}

func (b *Bot) SendOwnerMessage(text string) (int64, error) {
	if b.cfg.TelegramOwnerChatID == 0 {
		return 0, fmt.Errorf("owner chat ID is not set")
	}
	return b.SendMessage(b.cfg.TelegramOwnerChatID, text)
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

	if b.api == nil {
		return 0, nil
	}
	sentMsg, err := b.api.Send(msg)
	if err != nil {
		return 0, err
	}
	return int64(sentMsg.MessageID), nil
}

func (b *Bot) SendMessageWithKeyboard(chatID int64, text string, keyboard *tgbotapi.ReplyKeyboardMarkup) (int64, error) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeHTML
	msg.DisableWebPagePreview = true
	
	if keyboard != nil {
		msg.ReplyMarkup = keyboard
	}

	if b.api == nil {
		return 0, nil
	}
	sentMsg, err := b.api.Send(msg)
	if err != nil {
		return 0, err
	}
	return int64(sentMsg.MessageID), nil
}

func (b *Bot) SendChatMessageNotification(senderUsername, messageText string, convID, msgID int64) (int64, error) {
	if b.cfg.TelegramOwnerChatID == 0 {
		return 0, fmt.Errorf("owner chat ID is not set")
	}

	escapedText := strings.ReplaceAll(messageText, "<", "&lt;")
	escapedText = strings.ReplaceAll(escapedText, ">", "&gt;")
	
	// truncate if too long
	if len(escapedText) > 500 {
		escapedText = escapedText[:500] + "..."
	}

	text := fmt.Sprintf("💬 <b>Новое сообщение от %s</b>\n\n%s", senderUsername, escapedText)
	
	msg := tgbotapi.NewMessage(b.cfg.TelegramOwnerChatID, text)
	msg.ParseMode = tgbotapi.ModeHTML
	
	btn := tgbotapi.NewInlineKeyboardButtonData("📝 Сгенерировать ответ", fmt.Sprintf("generate_reply:%d:%d", convID, msgID))
	keyboard := tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(btn))
	msg.ReplyMarkup = keyboard

	sentMsg, err := b.api.Send(msg)
	if err != nil {
		return 0, err
	}
	return int64(sentMsg.MessageID), nil
}

func (b *Bot) SendSuitableProjectNotification(project *domain.Project, eval *domain.ProjectEvaluation) (int, error) {
	if b.cfg.TelegramOwnerChatID == 0 {
		return 0, fmt.Errorf("owner chat ID is not set")
	}

	formatThousands := func(n float64) string {
		s := fmt.Sprintf("%.0f", n)
		if len(s) > 3 {
			return s[:len(s)-3] + " " + s[len(s)-3:]
		}
		return s
	}

	var budgetStr string
	if (!project.BudgetFrom.Valid || project.BudgetFrom.Float64 == 0) && (!project.BudgetTo.Valid || project.BudgetTo.Float64 == 0) {
		budgetStr = "не указан"
	} else if project.BudgetFrom.Valid && project.BudgetTo.Valid && project.BudgetFrom.Float64 != project.BudgetTo.Float64 && project.BudgetFrom.Float64 > 0 && project.BudgetTo.Float64 > 0 {
		budgetStr = fmt.Sprintf("%s–%s ₽", formatThousands(project.BudgetFrom.Float64), formatThousands(project.BudgetTo.Float64))
	} else {
		val := project.BudgetTo.Float64
		if val == 0 && project.BudgetFrom.Valid {
			val = project.BudgetFrom.Float64
		}
		budgetStr = fmt.Sprintf("%s ₽", formatThousands(val))
	}

	titleStr := fmt.Sprintf("<b>%s</b>", project.Title)
	
	text := fmt.Sprintf("🔥 <b>Новый подходящий заказ</b>\n\n[%d] %s\n%s\n\n💰 Бюджет: %s\n⚙️ Сложность: %s\n⚠️ Риск: %s\n\n%s",
		eval.Score, eval.Category, titleStr, budgetStr, eval.Complexity, eval.RiskLevel, eval.Summary)

	msg := tgbotapi.NewMessage(b.cfg.TelegramOwnerChatID, text)
	msg.ParseMode = tgbotapi.ModeHTML
	msg.DisableWebPagePreview = true

	btnGen := tgbotapi.NewInlineKeyboardButtonData("📝 Сгенерировать отклик", fmt.Sprintf("generate_proposal:%d", project.ID))
	
	var row []tgbotapi.InlineKeyboardButton
	row = append(row, btnGen)
	if project.URL != "" {
		btnKwork := tgbotapi.NewInlineKeyboardButtonURL("🔗 Открыть заказ", project.URL)
		row = append(row, btnKwork)
	}
	
	keyboard := tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(row...))
	msg.ReplyMarkup = keyboard

	if b.api == nil {
		return 0, nil
	}
	sentMsg, err := b.api.Send(msg)
	if err != nil {
		return 0, err
	}
	return sentMsg.MessageID, nil
}

func (b *Bot) UpdateSuitableProjectNotification(project *domain.Project, eval *domain.ProjectEvaluation, messageID int) error {
	if b.cfg.TelegramOwnerChatID == 0 {
		return fmt.Errorf("owner chat ID is not set")
	}

	formatThousands := func(n float64) string {
		s := fmt.Sprintf("%.0f", n)
		if len(s) > 3 {
			return s[:len(s)-3] + " " + s[len(s)-3:]
		}
		return s
	}

	var budgetStr string
	if (!project.BudgetFrom.Valid || project.BudgetFrom.Float64 == 0) && (!project.BudgetTo.Valid || project.BudgetTo.Float64 == 0) {
		budgetStr = "не указан"
	} else if project.BudgetFrom.Valid && project.BudgetTo.Valid && project.BudgetFrom.Float64 != project.BudgetTo.Float64 && project.BudgetFrom.Float64 > 0 && project.BudgetTo.Float64 > 0 {
		budgetStr = fmt.Sprintf("%s–%s ₽", formatThousands(project.BudgetFrom.Float64), formatThousands(project.BudgetTo.Float64))
	} else {
		val := project.BudgetTo.Float64
		if val == 0 && project.BudgetFrom.Valid {
			val = project.BudgetFrom.Float64
		}
		budgetStr = fmt.Sprintf("%s ₽", formatThousands(val))
	}

	titleStr := fmt.Sprintf("<b>%s</b>", project.Title)
	
	isSuitable := eval.Score >= b.cfg.KworkSuitableScore && eval.Suitable && (eval.Category == "website" || eval.Category == "telegram")
	
	var text string
	var keyboard tgbotapi.InlineKeyboardMarkup

	if isSuitable {
		text = fmt.Sprintf("🔥 <b>Новый подходящий заказ</b>\n\n[%d] %s\n%s\n\n💰 Бюджет: %s\n⚙️ Сложность: %s\n⚠️ Риск: %s\n\n%s",
			eval.Score, eval.Category, titleStr, budgetStr, eval.Complexity, eval.RiskLevel, eval.Summary)
		
		btnGen := tgbotapi.NewInlineKeyboardButtonData("📝 Сгенерировать отклик", fmt.Sprintf("generate_proposal:%d", project.ID))
		var row []tgbotapi.InlineKeyboardButton
		row = append(row, btnGen)
		if project.URL != "" {
			btnKwork := tgbotapi.NewInlineKeyboardButtonURL("🔗 Открыть заказ", project.URL)
			row = append(row, btnKwork)
		}
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(row...))
	} else {
		text = fmt.Sprintf("⚠️ <b>Проект переоценён (Не подходит)</b>\n\n[%d] %s\n%s\n\n💰 Бюджет: %s\n\n%s",
			eval.Score, eval.Category, titleStr, budgetStr, eval.Summary)
		var row []tgbotapi.InlineKeyboardButton
		if project.URL != "" {
			btnKwork := tgbotapi.NewInlineKeyboardButtonURL("🔗 Открыть заказ", project.URL)
			row = append(row, btnKwork)
		}
		if len(row) > 0 {
			keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(row...))
		} else {
			keyboard = tgbotapi.NewInlineKeyboardMarkup() // Empty keyboard
		}
	}

	editMsg := tgbotapi.NewEditMessageTextAndMarkup(b.cfg.TelegramOwnerChatID, messageID, text, keyboard)
	editMsg.ParseMode = tgbotapi.ModeHTML
	editMsg.DisableWebPagePreview = true

	if b.api == nil {
		return nil
	}
	_, err := b.api.Send(editMsg)
	return err
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
			} else if update.CallbackQuery != nil {
				b.handleCallback(ctx, update.CallbackQuery)
			}
		}
	}
}

func (b *Bot) handleMessage(ctx context.Context, msg *tgbotapi.Message) {
	if msg.Chat.ID != b.cfg.TelegramOwnerChatID {
		b.logger.Warn("Unauthorized access attempt", "chat_id", msg.Chat.ID, "username", msg.From.UserName)
		b.SendMessage(msg.Chat.ID, "Access denied.")
		return
	}

	if msg.IsCommand() {
		b.routeCommand(ctx, msg, msg.Command())
		return
	}

	switch msg.Text {
	case "🔥 Заказы":
		b.routeCommand(ctx, msg, "orders")
	case "📊 Статус":
		b.routeCommand(ctx, msg, "status")
	case "🔄 Синхронизировать":
		b.routeCommand(ctx, msg, "sync")
	case "💬 Диалоги":
		b.routeCommand(ctx, msg, "dialogs")
	case "📥 Непрочитанные":
		b.routeCommand(ctx, msg, "unread")
	case "⏸ Пауза":
		b.routeCommand(ctx, msg, "pause")
	case "▶️ Продолжить":
		b.routeCommand(ctx, msg, "resume")
	}
}

func (b *Bot) buildMainMenu(ctx context.Context) tgbotapi.ReplyKeyboardMarkup {
	autoProcessing, _ := b.db.GetAppMeta(ctx, "auto_processing_enabled")
	var toggleBtn tgbotapi.KeyboardButton
	if autoProcessing == "true" {
		toggleBtn = tgbotapi.NewKeyboardButton("⏸ Пауза")
	} else {
		toggleBtn = tgbotapi.NewKeyboardButton("▶️ Продолжить")
	}

	kb := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🔥 Заказы"),
			tgbotapi.NewKeyboardButton("💬 Диалоги"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("📥 Непрочитанные"),
			tgbotapi.NewKeyboardButton("🔄 Синхронизировать"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("📊 Статус"),
			toggleBtn,
		),
	)
	kb.ResizeKeyboard = true
	kb.OneTimeKeyboard = false
	return kb
}

func (b *Bot) routeCommand(ctx context.Context, msg *tgbotapi.Message, cmd string) {
	var response string
	var keyboard *tgbotapi.ReplyKeyboardMarkup

	switch cmd {
	case "start":
		response = "✅ Kwork Assistant запущен."
		kb := b.buildMainMenu(ctx)
		keyboard = &kb
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
		kb := b.buildMainMenu(ctx)
		keyboard = &kb
	case "resume":
		response = b.handleResume(ctx)
		kb := b.buildMainMenu(ctx)
		keyboard = &kb
	case "sync", "dialogs", "unread", "orders":
		response = b.handleChatCommand(ctx, cmd)
	default:
		response = "Неизвестная команда."
	}

	if response != "" {
		if keyboard != nil {
			b.SendMessageWithKeyboard(msg.Chat.ID, response, keyboard)
		} else {
			b.SendMessage(msg.Chat.ID, response)
		}
	}
}

func (b *Bot) handleCallback(ctx context.Context, callback *tgbotapi.CallbackQuery) {
	if callback.Message.Chat.ID != b.cfg.TelegramOwnerChatID {
		b.logger.Warn("Unauthorized callback attempt", "chat_id", callback.Message.Chat.ID, "username", callback.From.UserName)
		return
	}

	data := callback.Data
	if strings.HasPrefix(data, "generate_reply:") {
		// e.g. generate_reply:convID:msgID
		parts := strings.Split(data, ":")
		if len(parts) == 3 {
			var convID, msgID int64
			fmt.Sscanf(parts[1], "%d", &convID)
			fmt.Sscanf(parts[2], "%d", &msgID)

			if b.replyGen != nil {
				go func() {
					b.SendMessage(callback.Message.Chat.ID, "⏳ Генерирую ответ...")
					err := b.replyGen.GenerateDraft(context.Background(), convID, msgID)
					if err != nil {
						b.SendMessage(callback.Message.Chat.ID, "❌ Ошибка при генерации ответа: "+err.Error())
					} else {
						// Let's assume GenerateDraft handles saving and we can query it
						b.SendMessage(callback.Message.Chat.ID, "✅ Ответ сгенерирован! (Проверьте БД / CLI)")
					}
				}()
			} else {
				b.SendMessage(callback.Message.Chat.ID, "❌ Reply Generator не настроен.")
			}
		}
	} else if strings.HasPrefix(data, "generate_proposal:") || strings.HasPrefix(data, "regenerate_proposal:") {
		forceRegen := strings.HasPrefix(data, "regenerate_proposal:")
		parts := strings.Split(data, ":")
		if len(parts) == 2 {
			var projectID int64
			fmt.Sscanf(parts[1], "%d", &projectID)

			if b.propGen != nil {
				go func() {
					ctx := context.Background()
					proj, err := b.db.GetProject(ctx, projectID)
					if err != nil {
						b.logger.Error("failed to get project", "projectID", projectID, "error", err)
						return
					}
					eval, err := b.db.GetEvaluationByExternalID(ctx, proj.Source, proj.ExternalID)
					if err != nil {
						b.logger.Error("failed to get evaluation", "projectID", projectID, "error", err)
						return
					}

					if eval.Score < b.cfg.KworkSuitableScore || !eval.Suitable || (eval.Category != "website" && eval.Category != "telegram") {
						msg := tgbotapi.NewMessage(callback.Message.Chat.ID, "⚠️ Проект был переоценён и больше не проходит текущий порог 80.")
						if b.api != nil {
							b.api.Send(msg)
						}
						return
					}

					// Check if we already have a draft for proposal-v4
					isCached := false
					if !forceRegen {
						draft, err := b.db.GetProposalDraftByExternalID(ctx, proj.Source, proj.ExternalID)
						if err == nil && draft != nil && draft.EvaluationID == eval.ID && draft.PromptVersion == "proposal-v4" {
							isCached = true
						}
					}
					
					var progressMsgID int
					var ticker *time.Ticker
					var done chan bool
					
					if isCached {
						// ACK the callback
						callbackConfig := tgbotapi.NewCallback(callback.ID, "✅ Использую сохранённый отклик.")
						if b.api != nil {
							b.api.Request(callbackConfig)
						}
					} else {
						// ACK the callback
						callbackConfig := tgbotapi.NewCallback(callback.ID, "Генерирую отклик...")
						if b.api != nil {
							b.api.Request(callbackConfig)
						}
						
						// Start progress
						if b.api != nil {
							msg := tgbotapi.NewMessage(callback.Message.Chat.ID, "⏳ Генерирую отклик...\nПрошло: 0 сек.")
							sentMsg, err := b.api.Send(msg)
							if err == nil {
								progressMsgID = sentMsg.MessageID
								ticker = time.NewTicker(5 * time.Second)
								done = make(chan bool)
								startTime := time.Now()
								
								go func() {
									for {
										select {
										case <-done:
											return
										case <-ticker.C:
											elapsed := int(time.Since(startTime).Seconds())
											timeStr := fmt.Sprintf("%d сек.", elapsed)
											if elapsed >= 60 {
												timeStr = fmt.Sprintf("%d мин %d сек.", elapsed/60, elapsed%60)
											}
											editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, progressMsgID, fmt.Sprintf("⏳ Генерирую отклик...\nПрошло: %s", timeStr))
											b.api.Send(editMsg)
										}
									}
								}()
							}
						}
					}

					b.logger.Info("proposal_generation_started", "projectID", projectID)
					startTime := time.Now()
					
					text, err := b.propGen.GenerateDraft(ctx, projectID, forceRegen)
					
					// Stop ticker
					if ticker != nil {
						ticker.Stop()
						done <- true
					}
					
					if err != nil {
						b.logger.Error("proposal_generation_failed", "projectID", projectID, "error", err, "duration", time.Since(startTime))
						
						elapsed := int(time.Since(startTime).Seconds())
						if progressMsgID != 0 && b.api != nil {
							editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, progressMsgID, fmt.Sprintf("⚠️ Не удалось сгенерировать отклик за %d сек.\nПопробуйте ещё раз.", elapsed))
							btnRetry := tgbotapi.NewInlineKeyboardButtonData("🔄 Попробовать снова", fmt.Sprintf("regenerate_proposal:%d", projectID))
							keyboard := tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(btnRetry))
							editMarkup := tgbotapi.NewEditMessageReplyMarkup(callback.Message.Chat.ID, progressMsgID, keyboard)
							b.api.Send(editMsg)
							b.api.Send(editMarkup)
						} else {
							// Send new failure message if progress failed to send
							msg := tgbotapi.NewMessage(callback.Message.Chat.ID, "⚠️ Не удалось сгенерировать отклик.\nПопробуйте ещё раз.")
							btnRetry := tgbotapi.NewInlineKeyboardButtonData("🔄 Попробовать снова", fmt.Sprintf("regenerate_proposal:%d", projectID))
							msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(btnRetry))
							if b.api != nil {
								b.api.Send(msg)
							}
						}
					} else {
						b.logger.Info("proposal_generation_finished", "projectID", projectID, "duration", time.Since(startTime))
						
						if progressMsgID != 0 && b.api != nil {
							elapsed := int(time.Since(startTime).Seconds())
							editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, progressMsgID, fmt.Sprintf("✅ Отклик готов за %d сек.", elapsed))
							b.api.Send(editMsg)
						}
						
						// Send the text
						msg := tgbotapi.NewMessage(callback.Message.Chat.ID, "📝 <b>Готовый отклик</b>\n\n<pre>"+text+"</pre>")
						msg.ParseMode = tgbotapi.ModeHTML
						
						btnRegen := tgbotapi.NewInlineKeyboardButtonData("🔄 Перегенерировать", fmt.Sprintf("regenerate_proposal:%d", projectID))
						
						var row []tgbotapi.InlineKeyboardButton
						row = append(row, btnRegen)
						if proj.URL != "" {
							btnKwork := tgbotapi.NewInlineKeyboardButtonURL("🔗 Открыть заказ", proj.URL)
							row = append(row, btnKwork)
						}
						msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(row...))
						
						if b.api != nil {
							b.api.Send(msg)
							b.logger.Info("proposal_telegram_sent", "projectID", projectID)
						}
					}
				}()
				return // We already ACKed
			} else {
				b.SendMessage(callback.Message.Chat.ID, "❌ Proposal Generator не настроен.")
			}
		}
	}
	
	callbackConfig := tgbotapi.NewCallback(callback.ID, "")
	if b.api != nil {
		b.api.Request(callbackConfig)
	}
}

func (b *Bot) handleStatus(ctx context.Context) string {
	var sb strings.Builder
	sb.WriteString("<b>Kwork Assistant</b>\n\n")
	
	// Assuming external health checks are passed since we are running, or we just display simple status
	sb.WriteString("TELEGRAM: ✅\n")
	
	autoProcessing, _ := b.db.GetAppMeta(ctx, "auto_processing_enabled")
	if autoProcessing == "true" {
		sb.WriteString("KWORK PROJECT WATCH: ✅\n")
		sb.WriteString("KWORK CHAT WATCH: ✅\n")
	} else {
		sb.WriteString("KWORK PROJECT WATCH: ⏸\n")
		sb.WriteString("KWORK CHAT WATCH: ⏸\n")
	}
	
	sb.WriteString(fmt.Sprintf("\nSuitable threshold: %d\n", b.cfg.KworkSuitableScore))
	
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
