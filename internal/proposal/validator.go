package proposal

import (
	"fmt"
	"strings"
	"regexp"

	"kwork-assistant/internal/domain"
)

func Validate(draft domain.ProposalDraft) error {
	if strings.TrimSpace(draft.Proposal) == "" {
		return fmt.Errorf("proposal cannot be empty")
	}
	if len([]rune(draft.Proposal)) > 1000 {
		return fmt.Errorf("proposal too long: %d chars (max 1000)", len([]rune(draft.Proposal)))
	}

	if draft.Confidence != "high" && draft.Confidence != "medium" && draft.Confidence != "low" {
		return fmt.Errorf("invalid confidence enum: %s", draft.Confidence)
	}

	if strings.Contains(draft.Proposal, "```") {
		return fmt.Errorf("proposal contains markdown blocks, which is not allowed")
	}

	if strings.Contains(draft.Proposal, "{") && strings.Contains(draft.Proposal, "}") {
		return fmt.Errorf("proposal looks like JSON, which is not allowed")
	}

	lower := strings.ToLower(draft.Proposal)
	if strings.Contains(lower, "уважаемый") || strings.Contains(lower, "с удовольствием") {
		return fmt.Errorf("proposal contains forbidden boilerplate (уважаемый/с удовольствием)")
	}

	if strings.Contains(lower, "я специализируюсь") || strings.Contains(lower, "огромный опыт") {
		return fmt.Errorf("proposal contains forbidden hallucinated experience claims")
	}

	// Unsupported Technical Certainty
	certaintyRegexes := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(проблема|ошибка)[^.?!]*(связана с|вызвана)`),
		regexp.MustCompile(`(?i)причина в`),
		regexp.MustCompile(`(?i)причиной является`),
		regexp.MustCompile(`(?i)\bвызван[оа]?\b`),
		regexp.MustCompile(`(?i)ошибка находится в`),
		regexp.MustCompile(`(?i)это стандартная проблема`),
		regexp.MustCompile(`(?i)(нужно|необходимо)\s+(заменить|изменить|исправить)\s+regex`),
		regexp.MustCompile(`(?i)решение\s*[-:—]\s*заменить`),
	}

	for _, re := range certaintyRegexes {
		if re.MatchString(draft.Proposal) {
			return fmt.Errorf("unsupported technical certainty matched by regex: %s", re.String())
		}
	}

	// Fake Format Halucination
	if strings.Contains(lower, "user..domain") || strings.Contains(lower, "user@sub.domain") {
		return fmt.Errorf("hallucinated fake email example 'user..domain' or 'user@sub.domain'")
	}

	// Asserting Access to Code
	codePhrases := []string{
		"изучу приложенный код",
		"начну с анализа кода во вложении",
		"начну с анализа кода",
	}
	for _, phrase := range codePhrases {
		if strings.Contains(lower, phrase) {
			return fmt.Errorf("hallucinated access to code without proof: %s", phrase)
		}
	}

	// Detect Synthetic / Leaked Contacts
	reEmail := regexp.MustCompile(`(?i)[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}`)
	if reEmail.MatchString(draft.Proposal) {
		return fmt.Errorf("proposal contains an email address (real or synthetic)")
	}

	reURL := regexp.MustCompile(`(?i)https?:\/\/[a-z0-9.\-]+(\.[a-z]{2,})?(\/[^\s]*)?`)
	if reURL.MatchString(draft.Proposal) {
		return fmt.Errorf("proposal contains a URL (real or synthetic)")
	}

	reWWW := regexp.MustCompile(`(?i)www\.[a-z0-9.\-]+\.[a-z]{2,}(\/[^\s]*)?`)
	if reWWW.MatchString(draft.Proposal) {
		return fmt.Errorf("proposal contains a www. URL (real or synthetic)")
	}

	reTMe := regexp.MustCompile(`(?i)t\.me\/[a-z0-9_]+`)
	if reTMe.MatchString(draft.Proposal) {
		return fmt.Errorf("proposal contains a Telegram link (real or synthetic)")
	}

	reTG := regexp.MustCompile(`@[a-zA-Z0-9_]{5,}`)
	if reTG.MatchString(draft.Proposal) {
		return fmt.Errorf("proposal contains a Telegram username (real or synthetic)")
	}

	rePhone := regexp.MustCompile(`(?i)(\+7|8)[\s\-\(]*\d{3}[\s\-\)]*\d{3}[\s\-]*\d{2}[\s\-]*\d{2}`)
	if rePhone.MatchString(draft.Proposal) {
		return fmt.Errorf("proposal contains a phone number (real or synthetic)")
	}

	// Check explicit domains like example.ru, site.ru, etc. that might escape simple word boundary
	reDomain := regexp.MustCompile(`(?i)\b(?:example|test|site)\.(ru|com|org|net)\b`)
	if reDomain.MatchString(draft.Proposal) {
		return fmt.Errorf("proposal contains a synthetic domain example")
	}

	return nil
}
