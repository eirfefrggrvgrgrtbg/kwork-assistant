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
	
	b.WriteString("--- КРИТИЧЕСКИ ВАЖНЫЕ ПРАВИЛА (PROPOSAL-V4) ---\n")
	b.WriteString("1. СТИЛЬ ФРИЛАНСЕРА. Никакого маркетингового бреда, никаких \"вы ничем не рискуете\", никаких \"с удовольствием\". Пиши сухо, по делу и профессионально.\n")
	b.WriteString("2. СТРОГАЯ СТРУКТУРА. Твой ответ должен СТРОГО следовать этой структуре:\n")
	b.WriteString("   - Начни с: \"Здравствуйте! Я full-stack разработчик, задачу посмотрел.\"\n")
	b.WriteString("   - Покажи ОДНИМ-ДВУМЯ предложениями, что понял суть задачи.\n")
	
	budgetStr := "не указан"
	if p.BudgetTo.Valid && p.BudgetTo.Float64 > 0 {
		if p.BudgetFrom.Valid && p.BudgetFrom.Float64 > 0 && p.BudgetFrom.Float64 != p.BudgetTo.Float64 {
			budgetStr = fmt.Sprintf("range: %.0f–%.0f %s", p.BudgetFrom.Float64, p.BudgetTo.Float64, p.Currency.String)
			b.WriteString(fmt.Sprintf("   - Затем про бюджет (у клиента %s). Напиши: \"Вижу бюджет %.0f–%.0f ₽ — подскажите, на какую сумму в этом диапазоне вы ориентируетесь?\"\n", budgetStr, p.BudgetFrom.Float64, p.BudgetTo.Float64))
		} else {
			budgetStr = fmt.Sprintf("exact: %.0f %s", p.BudgetTo.Float64, p.Currency.String)
			b.WriteString(fmt.Sprintf("   - Затем про бюджет (у клиента %s). Напиши: \"Вижу бюджет %.0f ₽. После уточнения полного объёма скажу, укладывается ли задача в него.\"\n", budgetStr, p.BudgetTo.Float64))
		}
	} else {
		b.WriteString("   - Бюджет не указан. Напиши: \"Подскажите, какой бюджет вы закладываете?\"\n")
	}

	b.WriteString("   - Затем: \"Примеры работ есть в профиле.\"\n")
	b.WriteString("   - Затем: \"Готов подключиться после уточнения деталей.\"\n")
	b.WriteString("   - Затем ОБЯЗАТЕЛЬНО: \"Готов работать через Kwork — оплату получаю после сдачи и принятия результата.\"\n")
	b.WriteString("   - И в конце (вопрос про сроки): \"Подскажите, к какой дате нужен готовый результат?\"\n")
	b.WriteString("3. ОДИН ВОПРОС. Сгенерируй МАКСИМУМ ОДИН уточняющий вопрос по самой задаче и положи его в JSON поле 'question'. В самом тексте 'proposal' этот вопрос писать НЕ НУЖНО (только вопрос про сроки).\n")
	b.WriteString("4. НИКАКОГО ВЫДУМАННОГО ОПЫТА. ЗАПРЕЩЕНО писать \"у меня много таких проектов\", \"с опытом в Python/Go\", \"делал такие интеграции\". Ты просто \"full-stack разработчик\".\n")
	b.WriteString("5. ТЕХНИЧЕСКАЯ ЧАСТЬ. В 'proposal' можно использовать МАКСИМУМ ОДНО техническое предложение (например, \"После уточнения версии X смогу определить вариант интеграции с Y.\").\n")
	b.WriteString("6. РАЗМЕР. Целевой размер текста (proposal) — 450–650 символов. МАКСИМУМ 1000 символов.\n")
	b.WriteString("7. СТРУКТУРА JSON. Верни ТОЛЬКО JSON без markdown блоков.\n\n")

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
  "question": "<один точный уточняющий технический вопрос>",
  "internal_approach": "<короткий комментарий для себя>",
  "confidence": "high" | "medium" | "low",
  "warnings": ["<предупреждение>"]
}`)

	return b.String()
}
