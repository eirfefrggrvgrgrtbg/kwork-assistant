package kwork

// Dialog represents a conversation with another user.
type Dialog struct {
	UnreadCount    int          `json:"unread_count"`
	LastMessage    string       `json:"last_message"`
	Time           int          `json:"time"`
	UserID         int          `json:"user_id"`
	Username       string       `json:"username"`
	ProfilePicture string       `json:"profilepicture"`
	Status         string       `json:"status"`
	HasActiveOrder bool         `json:"has_active_order"`
	Archived       bool         `json:"archived"`
	LastMessageObj *LastMessage `json:"lastMessage,omitempty"`
}

// LastMessage represents the summary of the last message in a Dialog.
type LastMessage struct {
	Unread       bool   `json:"unread"`
	FromUsername string `json:"fromUsername"`
	FromUserID   int    `json:"fromUserId"`
	Type         string `json:"type"`
	Time         int    `json:"time"`
	Message      string `json:"message"`
}

// InboxMessage represents a single message within a conversation.
type InboxMessage struct {
	MessageID          int         `json:"message_id"`
	ToID               int         `json:"to_id"`
	ToUsername         string      `json:"to_username"`
	FromID             int         `json:"from_id"`
	FromUsername       string      `json:"from_username"`
	Message            string      `json:"message"`
	Time               int         `json:"time"`
	Unread             bool        `json:"unread"`
	Type               string      `json:"type,omitempty"`
	Status             string      `json:"status"`
}
