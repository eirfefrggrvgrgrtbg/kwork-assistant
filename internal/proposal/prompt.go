package proposal

import (
	"fmt"
	"strings"

	"kwork-assistant/internal/domain"
)

// sanitizeDescription is now handled by SanitizeText in sanitizer.go

func BuildPrompt(p domain.Project, eval domain.ProjectEvaluation) string {
	var b strings.Builder

	b.WriteString("Ты — AI, который пишет ОТКЛИК (proposal) от лица реального full-stack разработчика на заказ с фриланс-биржи Kwork.\n\n")
	
	b.WriteString("--- КРИТИЧЕСКИ ВАЖНЫЕ ПРАВИЛА (PROPOSAL-V5) ---\n")
	b.WriteString("1. НАЧАЛО (HOOK). ЗАПРЕЩЕНО использовать шаблонное \"Здравствуйте! Я full-stack разработчик, задачу посмотрел.\". Начни отклик сразу с контекста задачи, использовав минимум 2 конкретные детали из описания. Пример: \"Здравствуйте! По описанию у вас уже есть Unity 2D проект, и сейчас нужно довести physics merge до релиза.\"\n")
	b.WriteString("2. СЛЕДУЮЩИЙ ШАГ. Коротко покажи, что логично сделать первым. Пример: \"Я бы начал с быстрого просмотра текущей сборки и списка проблем...\"\n")
	
	budget := p.GetBudget()
	budgetStr := "не указан"

	switch budget.Type {
	case domain.BudgetRange:
		budgetStr = fmt.Sprintf("range: %.0f–%.0f %s", budget.Min, budget.Max, p.Currency.String)
		b.WriteString(fmt.Sprintf("3. БЮДЖЕТ (у клиента %s). Напиши: \"Бюджет %.0f–%.0f ₽ вижу: после просмотра проекта скажу, в какую часть диапазона укладывается работа.\"\n", budgetStr, budget.Min, budget.Max))
	case domain.BudgetExact:
		budgetStr = fmt.Sprintf("exact: %.0f %s", budget.Min, p.Currency.String)
		b.WriteString(fmt.Sprintf("3. БЮДЖЕТ (у клиента %s). Напиши: \"Бюджет %.0f ₽ вижу. Если объём соответствует описанию, ориентир понятен.\"\n", budgetStr, budget.Min))
	case domain.BudgetUnknown:
		b.WriteString("3. БЮДЖЕТ не указан. Напиши: \"По стоимости смогу сориентировать после уточнения объёма.\"\n")
	}

	b.WriteString("4. ДОВЕРИЕ И ОПЛАТА. Коротко (по желанию): \"Примеры работ есть в профиле.\" ОБЯЗАТЕЛЬНО добавь: \"Работа через Kwork — оплата после приёмки результата.\"\n")
	b.WriteString("5. CTA (CALL TO ACTION). Закончи отклик ОДНИМ сильным вопросом или призывом к действию, который реально двигает проект, и вставь его прямо в текст отклика. Поле 'question' в JSON оставь пустым (\"\"). НЕ спрашивай про дедлайн, если это не самая важная вещь.\n")
	b.WriteString("6. ЗАПРЕТЫ. Никакого выдуманного опыта. Не пиши детали архитектуры и список языков. Не задавай вопрос про бюджет (сколько клиент готов заплатить).\n")
	b.WriteString("7. РАЗМЕР. Целевой размер текста — 300–550 символов. Мягкий максимум 650. ЖЁСТКИЙ МАКСИМУМ 1000 символов. Пиши короткими абзацами (не более трех).\n")
	b.WriteString("8. СТРУКТУРА JSON. Верни ТОЛЬКО JSON без markdown блоков.\n\n")

	b.WriteString("--- ДАННЫЕ ЗАКАЗА ---\n")
	b.WriteString(fmt.Sprintf("ЗАГОЛОВОК: %s\n", SanitizeText(p.Title)))
	b.WriteString(fmt.Sprintf("ОПИСАНИЕ: %s\n", SanitizeText(p.Description)))

	b.WriteString("\n--- ТВОИ ПРЕДЫДУЩИЕ ОЦЕНКИ (ВНУТРЕННИЙ КОНТЕКСТ) ---\n")
	b.WriteString(fmt.Sprintf("КАТЕГОРИЯ: %s\n", eval.Category))
	b.WriteString(fmt.Sprintf("СУТЬ (САММАРИ): %s\n", eval.Summary))
	
	b.WriteString(`
Твоя задача — вернуть СТРОГО JSON следующего формата:
{
  "proposal": "<сам текст отклика, строго по структуре>",
  "question": "",
  "internal_approach": "<короткий комментарий для себя>",
  "confidence": "high" | "medium" | "low",
  "warnings": ["<предупреждение>"]
}
`)

	return b.String()
}
