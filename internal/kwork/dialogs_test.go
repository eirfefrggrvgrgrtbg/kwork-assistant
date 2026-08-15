package kwork

import (
	"encoding/json"
	"testing"
)

func TestDialogLastMessageDirection(t *testing.T) {
	// A real payload simulation where FromUsername ("Служба поддержки") != Username ("Support")
	// but FromUserID == UserID (71232)
	payload := `[{
		"unread_count": 0,
		"last_message": "Здравствуйте.\n\nОткрепили данный ИНН от удаленного аккаунта.\n\nЕсли понадобится помощь, обращайтесь.",
		"time": 1786300854,
		"user_id": 71232,
		"username": "Support",
		"profilepicture": "https://...",
		"status": "online",
		"has_active_order": false,
		"archived": false,
		"lastMessage": {
			"unread": false,
			"fromUsername": "Служба поддержки",
			"fromUserId": 71232,
			"type": "text",
			"time": 1786300854,
			"message": "Здравствуйте.\n\nОткрепили данный ИНН от удаленного аккаунта.\n\nЕсли понадобится помощь, обращайтесь."
		}
	}]`

	var dialogs []Dialog
	if err := json.Unmarshal([]byte(payload), &dialogs); err != nil {
		t.Fatalf("Failed to parse payload: %v", err)
	}

	if len(dialogs) != 1 {
		t.Fatalf("Expected 1 dialog, got %d", len(dialogs))
	}

	d := dialogs[0]

	direction := "INCOMING"
	if d.LastMessageObj.FromUserID != d.UserID {
		direction = "OUTGOING (ME)"
	}

	if direction != "INCOMING" {
		t.Errorf("Expected direction INCOMING, got %s", direction)
	}
}
