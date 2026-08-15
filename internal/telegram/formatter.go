package telegram

import (
	"fmt"
	"strings"

	"kwork-assistant/internal/domain"
)

func FormatSuitableProject(p domain.Project, eval domain.ProjectEvaluation, draft domain.ProposalDraft) string {
	var sb strings.Builder

	sb.WriteString("🔥 <b>Новый подходящий проект</b>\n\n")
	
	categoryIcon := "🌐"
	if eval.Category == "telegram" {
		categoryIcon = "🤖"
	}
	sb.WriteString(fmt.Sprintf("[%d] %s %s\n", eval.Score, categoryIcon, strings.ToUpper(eval.Category)))
	sb.WriteString(fmt.Sprintf("<b>%s</b>\n\n", escapeHTML(p.Title)))

	budgetStr := "не указан"
	budget := p.GetBudget()
	if budget.Type == domain.BudgetRange {
		budgetStr = fmt.Sprintf("%.0f–%.0f %s", budget.Min, budget.Max, p.Currency.String)
	} else if budget.Type == domain.BudgetExact {
		budgetStr = fmt.Sprintf("%.0f %s", budget.Min, p.Currency.String)
	}
	sb.WriteString(fmt.Sprintf("💰 Бюджет: %s\n", budgetStr))
	
	sb.WriteString(fmt.Sprintf("⚙️ Сложность: %s\n", eval.Complexity))
	sb.WriteString(fmt.Sprintf("⚠️ Риск: %s\n\n", eval.RiskLevel))
	
	sb.WriteString("<b>Кратко:</b>\n")
	sb.WriteString(fmt.Sprintf("<i>%s</i>\n\n", escapeHTML(eval.Summary)))

	sb.WriteString("📝 <b>Готовый отклик:</b>\n")
	sb.WriteString(fmt.Sprintf("<code>%s</code>\n\n", escapeHTML(draft.Proposal)))

	sb.WriteString("❓ <b>Вопрос:</b>\n")
	if draft.Question != "" {
		sb.WriteString(fmt.Sprintf("<i>%s</i>\n", escapeHTML(draft.Question)))
	} else {
		sb.WriteString("<i>нет</i>\n")
	}

	return sb.String()
}

func escapeHTML(text string) string {
	text = strings.ReplaceAll(text, "&", "&amp;")
	text = strings.ReplaceAll(text, "<", "&lt;")
	text = strings.ReplaceAll(text, ">", "&gt;")
	return text
}
