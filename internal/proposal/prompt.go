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
	
	b.WriteString("--- КРИТИЧЕСКИ ВАЖНЫЕ ПРАВИЛА (PROPOSAL-V3) ---\n")
	b.WriteString("1. НИКАКОЙ ВОДЫ. Не пиши 'Уважаемый', 'С удовольствием', 'Готов выполнить', 'Я специализируюсь', 'Большой опыт'.\n")
	b.WriteString("2. БЕЗ ШАБЛОНОВ. ЗАПРЕЩЕНО начинать со слов \"Здравствуйте. Посмотрел задачу...\", \"Посмотрел задачу по...\", \"Ознакомился с...\". Начинай сразу с сути технического решения или вопроса.\n")
	b.WriteString("3. НЕ ВРАТЬ И НЕ ГАЛЛЮЦИНИРОВАТЬ. NEVER claim an action has already been performed unless the application context explicitly proves it was performed. ЗАПРЕЩЕНО писать \"проверил\", \"протестировал\", \"воспроизвел\", \"изучил код\", \"посмотрел проект\" если ты этого не делал. Пиши \"По описанию проблема похожа на...\", \"Судя по задаче...\", \"После получения доступа проверю...\".\n")
	b.WriteString("4. ОПЫТ. Не пиши \"У меня есть аналогичные проекты\", \"Делал такие проекты\". Пиши \"Могу показать демо-концепт похожего интерфейса\" и используй слова \"демо\" или \"концепт\".\n")
	b.WriteString("5. АРХИТЕКТУРА. Не навязывай сложную архитектуру (микросервисы, Kubernetes, CDN), язык или фреймворк, если клиент этого не просит. Предлагай это только как один из вариантов после анализа.\n")
	b.WriteString("6. ПОКАЖИ ПОНИМАНИЕ. Используй конкретную деталь из описания (название технологии, CMS, специфика), чтобы доказать, что читал.\n")
	b.WriteString("7. НИКАКОГО КОПИРОВАНИЯ КОНТАКТОВ И ССЫЛОК. ЗАПРЕЩЕНО использовать любые email или URL из описания задачи (даже в качестве примера!). Всегда заменяй их на общие слова (например, 'многоуровневые домены' вместо конкретного email).\n")
	b.WriteString("8. ВОПРОС ПО СУЩЕСТВУ. Задай ровно один ключевой уточняющий вопрос (в поле question), если он нужен. Не спрашивай то, что уже написано.\n")
	b.WriteString("9. СТРУКТУРА JSON. Верни ТОЛЬКО JSON без markdown блоков, строго по схеме.\n")
	b.WriteString("10. ТЕХНИЧЕСКИЕ ГИПОТЕЗЫ. Никогда не утверждай точную техническую причину проблемы, если она не подтверждена. Запрещено: \"проблема вызвана...\", \"причина в...\", \"решение — заменить...\", \"связана с\", \"ошибка связана с\". Вместо этого пиши гипотезы: \"по описанию похоже...\", \"первым делом проверю, где именно (фронтенд/бекенд/плагин)...\", \"если проблема в X, исправлю X...\". Отделяй известные факты от гипотез, указывай конкретный порядок диагностики и следующее за ней исправление. ЗАПРЕЩЕНО выдумывать форматы данных вроде \"user..domain\", если они не указаны в явном виде.\n\n")

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
