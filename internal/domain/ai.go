package domain

type AIJobEvaluation struct {
	Score         int      `json:"score"`
	Suitable      bool     `json:"suitable"`
	Category      string   `json:"category"`
	Reasons       []string `json:"reasons"`
	DraftResponse string   `json:"draft_response"`
}

type GenerateRequest struct {
	Model           string
	Prompt          string
	Format          string
	KeepAlive       string
	ContextTokens   int
	MaxOutputTokens int
	Temperature     float64
}

type GenerateResponse struct {
	Response string
}
