package reply

import (
	"errors"
	"regexp"
	"strings"
)

var (
	emailRegex    = regexp.MustCompile(`(?i)[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}`)
	phoneRegex    = regexp.MustCompile(`(?i)(?:\+7|8)[\s\-]?\(?[0-9]{3}\)?[\s\-]?[0-9]{3}[\s\-]?[0-9]{2}[\s\-]?[0-9]{2}`)
	telegramRegex = regexp.MustCompile(`(?i)(?:t\.me/|@)[A-Za-z0-9_]{5,}`)
	skypeRegex    = regexp.MustCompile(`(?i)(?:skype:|skype\s)[A-Za-z0-9_.-]+`)
)

func ValidateDraft(draft string) error {
	if strings.TrimSpace(draft) == "" {
		return errors.New("draft is empty")
	}

	if emailRegex.MatchString(draft) {
		return errors.New("draft contains email address")
	}

	if phoneRegex.MatchString(draft) {
		return errors.New("draft contains phone number")
	}

	if telegramRegex.MatchString(draft) {
		return errors.New("draft contains telegram username/link")
	}

	if skypeRegex.MatchString(draft) {
		return errors.New("draft contains skype username")
	}

	// General off-platform words check
	lowerDraft := strings.ToLower(draft)
	if strings.Contains(lowerDraft, "вконтакте") || strings.Contains(lowerDraft, "whatsapp") || strings.Contains(lowerDraft, "viber") {
		return errors.New("draft contains forbidden off-platform messengers")
	}

	// Prevent hallucinated technical claims like "связана с маской" 
	// Not strictly enforced here unless we want keyword blocks, but we rely on prompt.
	
	return nil
}
