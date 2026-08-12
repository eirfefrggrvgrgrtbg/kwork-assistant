package intake

import (
	"regexp"
	"strconv"
	"strings"

	"kwork-assistant/internal/domain"
	"kwork-assistant/internal/email"
)

var (
	// Extracts project ID from URL like kwork.ru/projects/123456/view
	projectIDRegex = regexp.MustCompile(`projects/(\d+)`)
	budgetRegex    = regexp.MustCompile(`Бюджет:\s*до\s*([\d\s]+)\s*(руб|₽|usd|\$)`)
)

func ParseKworkEmail(e domain.InboundEmail) (*domain.EmailProjectCandidate, bool) {
	// Simple detection: is this from Kwork?
	if !strings.Contains(strings.ToLower(e.Sender), "kwork.ru") && !strings.Contains(strings.ToLower(e.Subject), "kwork") && !strings.Contains(strings.ToLower(e.TextBody), "kwork") {
		return nil, false
	}

	plainText := e.TextBody
	if plainText == "" && e.HTMLBody != "" {
		plainText = email.StripHTML(e.HTMLBody)
	}

	candidate := &domain.EmailProjectCandidate{
		ParseConfidence: domain.ParseConfidenceLow,
		MissingFields:   []string{},
	}

	// Try to find ExternalID
	match := projectIDRegex.FindStringSubmatch(plainText)
	if match == nil && e.HTMLBody != "" {
		match = projectIDRegex.FindStringSubmatch(e.HTMLBody)
	}
	
	if match != nil && len(match) > 1 {
		id, err := strconv.ParseInt(match[1], 10, 64)
		if err == nil {
			candidate.ExternalID = id
			candidate.URL = "https://kwork.ru/projects/" + match[1] + "/view"
		}
	} else {
		candidate.MissingFields = append(candidate.MissingFields, "ExternalID")
	}

	// Title
	// Email subject might be like "Новый проект: Создать сайт"
	titlePrefixes := []string{"Новый проект:", "Новый проект на бирже:"}
	for _, prefix := range titlePrefixes {
		if strings.HasPrefix(e.Subject, prefix) {
			candidate.Title = strings.TrimSpace(strings.TrimPrefix(e.Subject, prefix))
			break
		}
	}
	if candidate.Title == "" {
		candidate.Title = e.Subject
	}

	// Budget
	bMatch := budgetRegex.FindStringSubmatch(plainText)
	if bMatch != nil && len(bMatch) > 2 {
		amountStr := strings.ReplaceAll(bMatch[1], " ", "")
		amount, _ := strconv.ParseFloat(amountStr, 64)
		candidate.BudgetTo = amount
		candidate.BudgetFrom = amount // Assuming Kwork mostly sends "up to X"
		
		currency := strings.ToLower(bMatch[2])
		if currency == "руб" || currency == "₽" {
			candidate.Currency = "RUB"
		} else {
			candidate.Currency = "USD"
		}
	} else {
		candidate.MissingFields = append(candidate.MissingFields, "Budget")
	}

	// Description
	// For description, since we don't know the exact format, we just take the plain text
	// and try to trim some boilerplate.
	lines := strings.Split(plainText, "\n")
	var descLines []string
	capture := false
	for _, line := range lines {
		l := strings.TrimSpace(line)
		if l == "" {
			continue
		}
		if strings.Contains(strings.ToLower(l), "описание:") || strings.Contains(strings.ToLower(l), "задача:") {
			capture = true
			continue
		}
		if capture {
			if strings.Contains(l, "Бюджет:") || strings.Contains(l, "Желаемые навыки:") || strings.Contains(l, "Посмотреть проект") {
				break
			}
			descLines = append(descLines, l)
		}
	}
	
	candidate.Description = strings.Join(descLines, "\n")
	if candidate.Description == "" {
		// Fallback: just use a chunk of text body
		if len(plainText) > 500 {
			candidate.Description = plainText[:500] + "..."
		} else {
			candidate.Description = plainText
		}
		candidate.MissingFields = append(candidate.MissingFields, "Description")
	}

	// Determine confidence
	if candidate.ExternalID != 0 && candidate.BudgetTo > 0 && len(descLines) > 0 {
		candidate.ParseConfidence = domain.ParseConfidenceHigh
	} else if candidate.ExternalID != 0 {
		candidate.ParseConfidence = domain.ParseConfidenceMedium
	} else {
		candidate.ParseConfidence = domain.ParseConfidenceLow
	}

	return candidate, true
}
