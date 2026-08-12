package types

// LastMessage представляет последнее сообщение в диалоге
type LastMessage struct {
	Unread       bool   `json:"unread"`
	FromUsername string `json:"fromUsername"`
	FromUserID   int    `json:"fromUserId"`
	Type         string `json:"type"`
	Time         int    `json:"time"`
	Message      string `json:"message"`
}

// Dialog представляет диалог
type Dialog struct {
	UnreadCount    int          `json:"unread_count"`
	LastMessageText string      `json:"last_message"`
	Time           int          `json:"time"`
	UserID         int          `json:"user_id"`
	Username       string       `json:"username"`
	ProfilePicture string       `json:"profilepicture"`
	Link           string       `json:"link"`
	Status         string       `json:"status"`
	BlockedByUser  bool         `json:"blocked_by_user"`
	AllowedDialog  bool         `json:"allowedDialog"`
	LastMessage    *LastMessage `json:"lastMessage,omitempty"`
	HasActiveOrder bool         `json:"has_active_order"`
	Archived       bool         `json:"archived"`
	IsStarred      bool         `json:"isStarred"`
}

// InboxMessage представляет сообщение в диалоге
type InboxMessage struct {
	MessageID          int         `json:"message_id"`
	ToID               int         `json:"to_id"`
	ToUsername         string      `json:"to_username"`
	ToLiveDate         int         `json:"to_live_date"`
	FromID             int         `json:"from_id"`
	FromUsername       string      `json:"from_username"`
	FromLiveDate       int         `json:"from_live_date"`
	FromProfilePicture string      `json:"from_profilepicture"`
	Message            string      `json:"message"`
	Time               int         `json:"time"`
	Unread             bool        `json:"unread"`
	Type               string      `json:"type,omitempty"`
	Status             string      `json:"status"`
	CreatedOrderID     interface{} `json:"created_order_id,omitempty"`
	Forwarded          bool        `json:"forwarded"`
	UpdatedAt          int         `json:"updated_at,omitempty"`
	MessagePage        int         `json:"message_page"`
}
