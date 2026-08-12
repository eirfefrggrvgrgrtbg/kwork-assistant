package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	AppEnv            string
	DatabasePath      string
	OllamaBaseURL     string
	OllamaModel       string
	OllamaKeepAlive   string
	AITimeoutSeconds  int
	AIContextTokens   int
	AIMaxOutputTokens int
	AITemperature     float64

	KworkLogin     string
	KworkPassword  string
	KworkPhoneLast string
	KworkPollLimit int

	IMAPHost          string
	IMAPPort          int
	IMAPUsername      string
	IMAPPassword      string
	IMAPUseTLS        bool
	EmailFolder       string
	EmailPollInterval string
	EmailLookback     int
	EmailSender       string
	EmailSubject      string

	TelegramBotToken    string
	TelegramOwnerChatID int64
}

func Load() (*Config, error) {
	cfg := &Config{
		AppEnv:          getEnvOrDefault("APP_ENV", "development"),
		DatabasePath:    getEnvOrDefault("DATABASE_PATH", "./data/kwork-assistant.db"),
		OllamaBaseURL:   getEnvOrDefault("OLLAMA_BASE_URL", "http://127.0.0.1:11434"),
		OllamaModel:     getEnvOrDefault("OLLAMA_MODEL", "gemma4:e4b"),
		OllamaKeepAlive: getEnvOrDefault("OLLAMA_KEEP_ALIVE", "2m"),
		KworkLogin:      os.Getenv("KWORK_LOGIN"),
		KworkPassword:   os.Getenv("KWORK_PASSWORD"),
		KworkPhoneLast:  os.Getenv("KWORK_PHONE_LAST"),
		IMAPHost:            os.Getenv("EMAIL_IMAP_HOST"),
		IMAPUsername:        os.Getenv("EMAIL_IMAP_USERNAME"),
		IMAPPassword:        os.Getenv("EMAIL_IMAP_PASSWORD"),
		IMAPUseTLS:          getEnvOrDefault("EMAIL_IMAP_TLS", "true") == "true",
		EmailFolder:         getEnvOrDefault("EMAIL_FOLDER", "INBOX"),
		EmailPollInterval:   getEnvOrDefault("EMAIL_POLL_INTERVAL", "60s"),
		EmailSender:         getEnvOrDefault("EMAIL_SENDER_FILTER", "notify@kwork.ru"),
		EmailSubject:        os.Getenv("EMAIL_SUBJECT_FILTER"),
		TelegramBotToken:    os.Getenv("TELEGRAM_BOT_TOKEN"),
	}

	if portStr := os.Getenv("EMAIL_IMAP_PORT"); portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil {
			cfg.IMAPPort = p
		}
	} else {
		cfg.IMAPPort = 993
	}
	
	if lookbackStr := os.Getenv("EMAIL_LOOKBACK_HOURS"); lookbackStr != "" {
		if lb, err := strconv.Atoi(lookbackStr); err == nil {
			cfg.EmailLookback = lb
		}
	} else {
		cfg.EmailLookback = 48
	}

	if chatIDStr := os.Getenv("TELEGRAM_OWNER_CHAT_ID"); chatIDStr != "" {
		if id, err := strconv.ParseInt(chatIDStr, 10, 64); err == nil {
			cfg.TelegramOwnerChatID = id
		}
	}

	pollLimit, err := parseIntEnv("KWORK_POLL_LIMIT", 50)
	if err != nil {
		return nil, err
	}
	cfg.KworkPollLimit = pollLimit

	if cfg.OllamaModel == "" {
		return nil, fmt.Errorf("OLLAMA_MODEL is required")
	}

	timeout, err := parseIntEnv("AI_TIMEOUT_SECONDS", 120)
	if err != nil {
		return nil, err
	}
	cfg.AITimeoutSeconds = timeout

	ctxTokens, err := parseIntEnv("AI_CONTEXT_TOKENS", 8192)
	if err != nil {
		return nil, err
	}
	cfg.AIContextTokens = ctxTokens

	maxOut, err := parseIntEnv("AI_MAX_OUTPUT_TOKENS", 700)
	if err != nil {
		return nil, err
	}
	cfg.AIMaxOutputTokens = maxOut

	temp, err := parseFloatEnv("AI_TEMPERATURE", 0.3)
	if err != nil {
		return nil, err
	}
	cfg.AITemperature = temp

	return cfg, nil
}

func getEnvOrDefault(key, def string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return def
}

func parseIntEnv(key string, def int) (int, error) {
	val := os.Getenv(key)
	if val == "" {
		return def, nil
	}
	parsed, err := strconv.Atoi(val)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %v", key, err)
	}
	return parsed, nil
}

func parseFloatEnv(key string, def float64) (float64, error) {
	val := os.Getenv(key)
	if val == "" {
		return def, nil
	}
	parsed, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %v", key, err)
	}
	return parsed, nil
}
