package domain

import (
	"time"
)

type ProcessingStatus string

const (
	ProcessingStatusNew       ProcessingStatus = "new"
	ProcessingStatusParsed    ProcessingStatus = "parsed"
	ProcessingStatusIgnored   ProcessingStatus = "ignored"
	ProcessingStatusProcessed     ProcessingStatus = "processed"
	ProcessingStatusTerminalError ProcessingStatus = "terminal_error"
	ProcessingStatusRetryableError ProcessingStatus = "retryable_error"
)

type InboundEmail struct {
	ID               int64
	ProviderUID      string
	MessageID        string
	Sender           string
	Subject          string
	TextBody         string
	HTMLBody         string
	ReceivedAt       time.Time
	RawHash          string
	ProcessingStatus ProcessingStatus
	ProcessingError  string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type ParseConfidence string

const (
	ParseConfidenceHigh   ParseConfidence = "high"
	ParseConfidenceMedium ParseConfidence = "medium"
	ParseConfidenceLow    ParseConfidence = "low"
)

type EmailProjectCandidate struct {
	ExternalID      int64
	URL             string
	Title           string
	Description     string
	BudgetFrom      float64
	BudgetTo        float64
	Currency        string
	Category        string
	ParseConfidence ParseConfidence
	MissingFields   []string
}
