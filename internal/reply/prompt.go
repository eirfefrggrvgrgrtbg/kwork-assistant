package reply

import (
	"fmt"
	"strings"

	"kwork-assistant/internal/domain"
)

// BuildPrompt constructs the prompt for Gemma based on project context and chat history.
func BuildPrompt(project *domain.Project, history []domain.KworkMessage) string {
	var sb strings.Builder

	sb.WriteString("You are an expert IT freelancer communicating with a client on Kwork.\n")
	sb.WriteString("Your goal is to write a polite, professional, and helpful response to the client's latest message.\n\n")

	sb.WriteString("CRITICAL SAFETY RULES:\n")
	sb.WriteString("1. DO NOT suggest sharing contacts (email, phone, telegram, skype, etc.). All communication must stay on Kwork.\n")
	sb.WriteString("2. DO NOT promise exact prices or deadlines if they were not already agreed upon. Say you need to clarify details first.\n")
	sb.WriteString("3. Keep the response concise, clear, and relevant. Do not hallucinate technical causes.\n")
	sb.WriteString("4. Return ONLY valid JSON with two fields: \"draft_text\" (your response) and \"warnings\" (array of strings if you noticed risks).\n\n")

	if project != nil {
		sb.WriteString("--- PROJECT CONTEXT ---\n")
		sb.WriteString(fmt.Sprintf("Title: %s\n", project.Title))
		sb.WriteString(fmt.Sprintf("Description: %s\n", project.Description))
		if project.BudgetFrom.Valid {
			sb.WriteString(fmt.Sprintf("Budget: %.2f\n", project.BudgetFrom.Float64))
		}
		sb.WriteString("-----------------------\n\n")
	}

	sb.WriteString("--- CHAT HISTORY ---\n")
	for _, m := range history {
		role := "CLIENT"
		if m.Direction == domain.DirectionOutgoing {
			role = "ME"
		}
		sb.WriteString(fmt.Sprintf("[%s]: %s\n", role, m.Text))
	}
	sb.WriteString("--------------------\n\n")
	
	sb.WriteString("Write the next message as ME. Output JSON only.\n")

	return sb.String()
}
