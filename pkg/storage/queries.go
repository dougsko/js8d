package storage

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/dougsko/js8d/pkg/protocol"
)

// MessageQuery represents query parameters for retrieving messages
type MessageQuery struct {
	Limit      int
	Offset     int
	Since      *time.Time
	Until      *time.Time
	Callsign   string
	Direction  string // "RX", "TX", or "" for both
	MessageType string
	UnreadOnly bool
}

// ConversationSummary represents a conversation with a callsign
type ConversationSummary struct {
	Callsign        string    `json:"callsign"`
	LastMessageID   int       `json:"last_message_id"`
	LastMessageTime time.Time `json:"last_message_time"`
	LastMessageText string    `json:"last_message_text"`
	UnreadCount     int       `json:"unread_count"`
	TotalMessages   int       `json:"total_messages"`
}

// MessageStats represents database statistics
type MessageStats struct {
	TotalMessages int       `json:"total_messages"`
	TotalRX       int       `json:"total_rx"`
	TotalTX       int       `json:"total_tx"`
	LastCleanup   time.Time `json:"last_cleanup"`
}

// GetMessages retrieves messages based on query parameters
func (ms *MessageStore) GetMessages(query MessageQuery) ([]protocol.Message, error) {
	var args []interface{}
	var conditions []string

	sqlQuery := `
		SELECT id, timestamp, from_callsign, to_callsign, message_text,
			   snr, frequency, mode
		FROM messages
		WHERE 1=1
	`

	if query.Since != nil {
		conditions = append(conditions, "timestamp >= ?")
		args = append(args, query.Since)
	}

	if query.Until != nil {
		conditions = append(conditions, "timestamp <= ?")
		args = append(args, query.Until)
	}

	if query.Callsign != "" {
		conditions = append(conditions, "(from_callsign = ? OR to_callsign = ?)")
		args = append(args, query.Callsign, query.Callsign)
	}

	if query.Direction != "" {
		conditions = append(conditions, "direction = ?")
		args = append(args, query.Direction)
	}

	if query.MessageType != "" {
		conditions = append(conditions, "message_type = ?")
		args = append(args, query.MessageType)
	}

	if query.UnreadOnly {
		conditions = append(conditions, "is_read = FALSE")
	}

	// Add conditions to query
	for _, condition := range conditions {
		sqlQuery += " AND " + condition
	}

	// Order by timestamp descending (newest first)
	sqlQuery += " ORDER BY timestamp DESC"

	// Add limit and offset
	if query.Limit > 0 {
		sqlQuery += " LIMIT ?"
		args = append(args, query.Limit)

		if query.Offset > 0 {
			sqlQuery += " OFFSET ?"
			args = append(args, query.Offset)
		}
	}

	rows, err := ms.db.Query(sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query messages: %w", err)
	}
	defer rows.Close()

	var messages []protocol.Message
	for rows.Next() {
		var msg protocol.Message
		err := rows.Scan(
			&msg.ID,
			&msg.Timestamp,
			&msg.From,
			&msg.To,
			&msg.Message,
			&msg.SNR,
			&msg.Frequency,
			&msg.Mode,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}
		messages = append(messages, msg)
	}

	return messages, rows.Err()
}

// GetConversations retrieves conversation summaries
func (ms *MessageStore) GetConversations(limit int) ([]ConversationSummary, error) {
	query := `
		SELECT c.callsign, c.last_message_id, c.last_message_time,
			   c.unread_count, m.message_text,
			   (SELECT COUNT(*) FROM messages WHERE from_callsign = c.callsign OR to_callsign = c.callsign) as total_messages
		FROM conversations c
		LEFT JOIN messages m ON c.last_message_id = m.id
		ORDER BY c.last_message_time DESC
	`

	args := []interface{}{}
	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}

	rows, err := ms.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query conversations: %w", err)
	}
	defer rows.Close()

	var conversations []ConversationSummary
	for rows.Next() {
		var conv ConversationSummary
		var lastMessageText sql.NullString

		err := rows.Scan(
			&conv.Callsign,
			&conv.LastMessageID,
			&conv.LastMessageTime,
			&conv.UnreadCount,
			&lastMessageText,
			&conv.TotalMessages,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan conversation: %w", err)
		}

		if lastMessageText.Valid {
			conv.LastMessageText = lastMessageText.String
		}

		conversations = append(conversations, conv)
	}

	return conversations, rows.Err()
}

// GetMessagesByCallsign retrieves all messages for a specific callsign
func (ms *MessageStore) GetMessagesByCallsign(callsign string, limit int, offset int) ([]protocol.Message, error) {
	query := MessageQuery{
		Callsign: callsign,
		Limit:    limit,
		Offset:   offset,
	}
	return ms.GetMessages(query)
}

// GetRecentMessages retrieves the most recent messages
func (ms *MessageStore) GetRecentMessages(limit int) ([]protocol.Message, error) {
	query := MessageQuery{
		Limit: limit,
	}
	return ms.GetMessages(query)
}

// GetUnreadMessages retrieves all unread messages
func (ms *MessageStore) GetUnreadMessages() ([]protocol.Message, error) {
	query := MessageQuery{
		UnreadOnly: true,
		Limit:      1000, // Reasonable limit for unread messages
	}
	return ms.GetMessages(query)
}

// MarkMessagesAsRead marks messages as read for a specific callsign
func (ms *MessageStore) MarkMessagesAsRead(callsign string) error {
	tx, err := ms.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Mark messages as read
	_, err = tx.Exec(`
		UPDATE messages SET is_read = TRUE, updated_at = CURRENT_TIMESTAMP
		WHERE (from_callsign = ? OR to_callsign = ?) AND is_read = FALSE
	`, callsign, callsign)
	if err != nil {
		return fmt.Errorf("failed to mark messages as read: %w", err)
	}

	// Reset unread count for conversation
	_, err = tx.Exec(`
		UPDATE conversations SET unread_count = 0, updated_at = CURRENT_TIMESTAMP
		WHERE callsign = ?
	`, callsign)
	if err != nil {
		return fmt.Errorf("failed to reset unread count: %w", err)
	}

	return tx.Commit()
}

// GetMessageStats retrieves database statistics
func (ms *MessageStore) GetMessageStats() (*MessageStats, error) {
	var stats MessageStats
	var lastCleanup sql.NullTime

	err := ms.db.QueryRow(`
		SELECT total_messages, total_rx, total_tx, last_cleanup
		FROM message_stats WHERE id = 1
	`).Scan(&stats.TotalMessages, &stats.TotalRX, &stats.TotalTX, &lastCleanup)

	if err != nil {
		return nil, fmt.Errorf("failed to get message stats: %w", err)
	}

	if lastCleanup.Valid {
		stats.LastCleanup = lastCleanup.Time
	}

	return &stats, nil
}

// SearchMessages performs a full-text search on message content
func (ms *MessageStore) SearchMessages(searchTerm string, limit int) ([]protocol.Message, error) {
	query := `
		SELECT id, timestamp, from_callsign, to_callsign, message_text,
			   snr, frequency, mode
		FROM messages
		WHERE message_text LIKE ?
		ORDER BY timestamp DESC
	`

	args := []interface{}{"%" + searchTerm + "%"}
	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}

	rows, err := ms.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search messages: %w", err)
	}
	defer rows.Close()

	var messages []protocol.Message
	for rows.Next() {
		var msg protocol.Message
		err := rows.Scan(
			&msg.ID,
			&msg.Timestamp,
			&msg.From,
			&msg.To,
			&msg.Message,
			&msg.SNR,
			&msg.Frequency,
			&msg.Mode,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan search result: %w", err)
		}
		messages = append(messages, msg)
	}

	return messages, rows.Err()
}

// GetMessageCount returns the total number of messages
func (ms *MessageStore) GetMessageCount() (int, error) {
	var count int
	err := ms.db.QueryRow("SELECT COUNT(*) FROM messages").Scan(&count)
	return count, err
}

// GetUnreadCount returns the total number of unread messages
func (ms *MessageStore) GetUnreadCount() (int, error) {
	var count int
	err := ms.db.QueryRow("SELECT COUNT(*) FROM messages WHERE is_read = FALSE").Scan(&count)
	return count, err
}