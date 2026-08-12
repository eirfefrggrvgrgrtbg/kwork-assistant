package proposal

import (
	"regexp"
)

// SanitizeText removes or masks emails, URLs, Telegram usernames, and phone numbers.
func SanitizeText(text string) string {
	// 1. Email
	reEmail := regexp.MustCompile(`(?i)[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}`)
	text = reEmail.ReplaceAllString(text, "[EMAIL УДАЛЕН]")

	// 2. URLs (http/https)
	reURL := regexp.MustCompile(`(?i)https?:\/\/[a-z0-9.\-]+(\.[a-z]{2,})?(\/[^\s]*)?`)
	text = reURL.ReplaceAllString(text, "[URL УДАЛЕН]")

	// 3. www. URLs
	reWWW := regexp.MustCompile(`(?i)www\.[a-z0-9.\-]+\.[a-z]{2,}(\/[^\s]*)?`)
	text = reWWW.ReplaceAllString(text, "[URL УДАЛЕН]")

	// 4. t.me links
	reTMe := regexp.MustCompile(`(?i)t\.me\/[a-z0-9_]+`)
	text = reTMe.ReplaceAllString(text, "[TELEGRAM УДАЛЕН]")

	// 5. Telegram usernames (@username)
	reTG := regexp.MustCompile(`@[a-zA-Z0-9_]{5,}`)
	text = reTG.ReplaceAllString(text, "[TELEGRAM УДАЛЕН]")

	// 6. Phone numbers (e.g. +7 999 123-45-67, 8(999)123-45-67, +79991234567)
	// We use a relatively simple regex that catches common formats in RU/CIS.
	rePhone := regexp.MustCompile(`(?i)(\+7|8)[\s\-\(]*\d{3}[\s\-\)]*\d{3}[\s\-]*\d{2}[\s\-]*\d{2}`)
	text = rePhone.ReplaceAllString(text, "[ТЕЛЕФОН УДАЛЕН]")

	// 7. Plain domains (e.g. example.ru, domain.com)
	// Avoid replacing common words like "v.1.0" or "node.js". We restrict to common TLDs.
	reDomain := regexp.MustCompile(`(?i)\b[a-z0-9\-]+\.(ru|com|org|net|su|рф)\b`)
	text = reDomain.ReplaceAllString(text, "[САЙТ УДАЛЕН]")

	return text
}
