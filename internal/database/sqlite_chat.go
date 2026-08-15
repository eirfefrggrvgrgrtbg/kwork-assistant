package database

import (
	"context"
	"database/sql"

	"kwork-assistant/internal/domain"
)

func (db *DB) UpsertKworkConversation(ctx context.Context, c *domain.KworkConversation) error {
	var projectID sql.NullInt64
	if c.ProjectID.Valid {
		projectID = c.ProjectID
	}

	query := `
		INSERT INTO kwork_conversations (
			counterparty_user_id, counterparty_username, counterparty_display_name,
			project_id, last_external_message_id, last_message_at, unread_count
		) VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(counterparty_user_id) DO UPDATE SET
			counterparty_username = excluded.counterparty_username,
			counterparty_display_name = excluded.counterparty_display_name,
			project_id = COALESCE(excluded.project_id, kwork_conversations.project_id),
			last_external_message_id = COALESCE(excluded.last_external_message_id, kwork_conversations.last_external_message_id),
			last_message_at = excluded.last_message_at,
			unread_count = excluded.unread_count,
			updated_at = CURRENT_TIMESTAMP
	`
	res, err := db.ExecContext(ctx, query,
		c.CounterpartyUserID, c.CounterpartyUsername, c.CounterpartyDisplayName,
		projectID, c.LastExternalMessageID, c.LastMessageAt, c.UnreadCount,
	)
	if err != nil {
		return err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	if id > 0 {
		c.ID = id
	} else {
		// If it was an update, we need to fetch the ID
		err = db.QueryRowContext(ctx, "SELECT id FROM kwork_conversations WHERE counterparty_user_id = ?", c.CounterpartyUserID).Scan(&c.ID)
		if err != nil {
			return err
		}
	}
	return nil
}

func (db *DB) SaveKworkMessage(ctx context.Context, m *domain.KworkMessage) (bool, error) {
	query := `
		INSERT INTO kwork_messages (
			conversation_id, external_message_id, sender_user_id, sender_username,
			direction, text, sent_at, raw_hash
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(external_message_id) DO NOTHING
	`
	res, err := db.ExecContext(ctx, query,
		m.ConversationID, m.ExternalMessageID, m.SenderUserID, m.SenderUsername,
		string(m.Direction), m.Text, m.SentAt, m.RawHash,
	)
	if err != nil {
		return false, err
	}

	// CRITICAL: Use RowsAffected, NOT LastInsertId.
	// SQLite's last_insert_rowid() with ON CONFLICT DO NOTHING returns the
	// last *session* rowid even when nothing was inserted — making it
	// impossible to detect a conflict. RowsAffected returns 0 on DO NOTHING.
	rows, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	if rows > 0 {
		// Fetch the real ID for the caller.
		db.QueryRowContext(ctx, "SELECT id FROM kwork_messages WHERE external_message_id = ?", m.ExternalMessageID).Scan(&m.ID)
		return true, nil
	}
	return false, nil
}

func (db *DB) SaveReplyDraft(ctx context.Context, d *domain.ReplyDraft) error {
	query := `
		INSERT INTO reply_drafts (
			conversation_id, context_message_id, model_version, draft_text,
			status, sent_message_id
		) VALUES (?, ?, ?, ?, ?, ?)
	`
	res, err := db.ExecContext(ctx, query,
		d.ConversationID, d.ContextMessageID, d.ModelVersion, d.DraftText,
		d.Status, d.SentMessageID,
	)
	if err != nil {
		return err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	if id > 0 {
		d.ID = id
	}
	return nil
}

func (db *DB) GetUnprocessedKworkMessages(ctx context.Context, conversationID int64) ([]domain.KworkMessage, error) {
	query := `
		SELECT m.id, m.conversation_id, m.external_message_id, m.sender_user_id, m.sender_username, m.direction, m.text, m.sent_at, m.raw_hash, m.created_at
		FROM kwork_messages m
		LEFT JOIN reply_drafts rd ON m.id = rd.context_message_id
		WHERE m.conversation_id = ? AND m.direction = 'incoming' AND rd.id IS NULL
		ORDER BY m.sent_at ASC
	`
	rows, err := db.QueryContext(ctx, query, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []domain.KworkMessage
	for rows.Next() {
		var m domain.KworkMessage
		var dir string
		if err := rows.Scan(
			&m.ID, &m.ConversationID, &m.ExternalMessageID, &m.SenderUserID, &m.SenderUsername,
			&dir, &m.Text, &m.SentAt, &m.RawHash, &m.CreatedAt,
		); err != nil {
			return nil, err
		}
		m.Direction = domain.Direction(dir)
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}
