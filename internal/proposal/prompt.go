package proposal

import (
	"fmt"
	"strings"

	"kwork-assistant/internal/domain"
)

// sanitizeDescription is now handled by SanitizeText in sanitizer.go

func BuildPrompt(p domain.Project, eval domain.ProjectEvaluation) string {
	var b strings.Builder

	b.WriteString("Ты — опытный full-stack разработчик (Go, React, JS, Python, TG Bots, Web).\n")
	b.WriteString("Тебе нужно написать ОТКЛИК (proposal) на заказ с фриланс-биржи.\n\n")
	
	b.WriteString("--- КРИТИЧЕСКИ ВАЖНЫЕ ПРАВИЛА (PROPOSAL-V4) ---\n")
	b.WriteString("1. ЖИВОЙ СТИЛЬ (HUMAN SELLING COVER LETTER). Пиши так, как пишет живой крутой специалист клиенту. БЕЗ \"Уважаемый\", \"С удовольствием\", \"Готов выполнить\". Поздоровайся (\"Здравствуйте!\" или \"Привет!\").\n")
	b.WriteString("2. КОРОТКОЕ ИНТРО. В начале коротко представься как разработчик.\n")
	b.WriteString("3. ПОКАЖИ ПОНИМАНИЕ ЗАДАЧИ. Напиши суть того, как ты понял задачу клиента, покажи, что ты вник в его проблему.\n")
	b.WriteString("4. БЮДЖЕТ. ОБЯЗАТЕЛЬНО упомяни бюджет органично в тексте:\n")
	b.WriteString("   - Если указан точный бюджет, подтверди, что готов сделать за эти деньги.\n")
	b.WriteString("   - Если указан диапазон, скажи, что цена обсуждается в этих рамках после уточнения деталей.\n")
	b.WriteString("   - Если бюджет не указан, напиши, что точную цену сможешь назвать после обсуждения.\n")
	b.WriteString("5. ПОРТФОЛИО. Скажи клиенту, что примеры твоих работ можно посмотреть в профиле.\n")
	b.WriteString("6. ГОТОВНОСТЬ НАЧАТЬ. Упомяни, что готов приступить к работе.\n")
	b.WriteString("7. БЕЗОПАСНАЯ ОПЛАТА. Используй фразу про безопасную оплату: \"Оплата через Сейф Kwork, вы ничем не рискуете\" или аналогичную.\n")
	b.WriteString("8. ДЕДЛАЙН. Спроси про дедлайн (какие сроки).\n")
	b.WriteString("9. ТЕХНИЧЕСКИЕ ДЕТАЛИ. Максимум ОДНО предложение с техническим подходом/деталями, без перегруза терминами. Клиенту нужен результат, а не код.\n")
	b.WriteString("10. ВОПРОС. Максимум ОДИН уточняющий вопрос (записывается в поле question).\n")
	b.WriteString("11. ОБЪЕМ. Текст должен быть в пределах 450–750 символов.\n")
	b.WriteString("12. НИКАКОГО КОПИРОВАНИЯ КОНТАКТОВ И ССЫЛОК. ЗАПРЕЩЕНО использовать email или URL из задачи. ЗАПРЕЩЕНО выдумывать опыт, которого нет в задаче.\n")
	b.WriteString("13. СТРУКТУРА JSON. Верни ТОЛЬКО JSON без markdown блоков, строго по схеме.\n\n")

	b.WriteString("--- ДАННЫЕ ЗАКАЗА ---\n")
	b.WriteString(fmt.Sprintf("ЗАГОЛОВОК: %s\n", SanitizeText(p.Title)))
	b.WriteString(fmt.Sprintf("ОПИСАНИЕ: %s\n", SanitizeText(p.Description)))

	b.WriteString("\n--- ТВОИ ПРЕДЫДУЩИЕ ОЦЕНКИ (ВНУТРЕННИЙ КОНТЕКСТ) ---\n")
	b.WriteString(fmt.Sprintf("КАТЕГОРИЯ: %s\n", eval.Category))
	b.WriteString(fmt.Sprintf("ПРИЧИНЫ: %s\n", strings.Join(eval.Reasons, ", ")))
	b.WriteString(fmt.Sprintf("ОЦЕНКА УСИЛИЙ: %s\n", eval.EstimatedEffort))
	b.WriteString(fmt.Sprintf("СУТЬ (САММАРИ): %s\n", eval.Summary))
	b.WriteString(fmt.Sprintf("СЛОЖНОСТЬ: %s\n", eval.Complexity))
	b.WriteString(fmt.Sprintf("РИСКИ: %s\n", eval.RiskLevel))
	
	if p.BudgetTo.Valid && p.BudgetTo.Float64 > 0 {
		b.WriteString(fmt.Sprintf("БЮДЖЕТ: %.0f %s\n", p.BudgetTo.Float64, p.Currency.String))
	}

	b.WriteString(`
Твоя задача — вернуть СТРОГО JSON следующего формата:
{
  "proposal": "<сам текст отклика, который отправится клиенту (БЕЗ ВОДЫ И ШАБЛОНОВ)>",
  "question": "<один точный уточняющий вопрос для клиента, если нужен>",
  "internal_approach": "<твой внутренний комментарий, почему ты написал именно так>",
  "confidence": "high" | "medium" | "low",
  "warnings": ["<предупреждение 1 (если есть)>"]
}`)

	return b.String()
}
