package storage

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/dougsko/js8d/pkg/protocol"
	_ "github.com/mattn/go-sqlite3"
)

// MessageStore handles persistent storage of JS8 messages
type MessageStore struct {
	db          *sql.DB
	dbPath      string
	maxMessages int
}

// NewMessageStore creates a new message store with SQLite backend
func NewMessageStore(dbPath string, maxMessages int) (*MessageStore, error) {
	store := &MessageStore{
		dbPath:      dbPath,
		maxMessages: maxMessages,
	}

	if err := store.initialize(); err != nil {
		return nil, fmt.Errorf("failed to initialize message store: %w", err)
	}

	return store, nil
}

// initialize sets up the database connection and creates tables
func (ms *MessageStore) initialize() error {
	// Create database directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(ms.dbPath), 0755); err != nil {
		return fmt.Errorf("failed to create database directory: %w", err)
	}

	// Handle empty database path
	if ms.dbPath == "" {
		ms.dbPath = "./js8d.db" // Default database path
	}

	// Build connection string properly with query parameters
	connectionString := ms.dbPath + "?_busy_timeout=10000&_journal_mode=WAL&_foreign_keys=on"

	// Open database connection
	db, err := sql.Open("sqlite3", connectionString)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	ms.db = db

	// Create tables
	if err := ms.createTables(); err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}

	// Create indexes for performance
	if err := ms.createIndexes(); err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	log.Printf("Message store initialized: %s (max %d messages)", ms.dbPath, ms.maxMessages)
	return nil
}

// createTables creates the database schema
func (ms *MessageStore) createTables() error {
	schema := `
	CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		from_callsign TEXT NOT NULL,
		to_callsign TEXT NOT NULL DEFAULT '',
		message_text TEXT NOT NULL,
		snr REAL NOT NULL DEFAULT 0.0,
		frequency INTEGER NOT NULL DEFAULT 0,
		mode TEXT NOT NULL DEFAULT 'NORMAL',
		direction TEXT NOT NULL CHECK (direction IN ('RX', 'TX')),
		message_type TEXT NOT NULL DEFAULT 'MESSAGE',
		is_read BOOLEAN NOT NULL DEFAULT FALSE,
		grid_square TEXT DEFAULT '',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS conversations (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		callsign TEXT NOT NULL UNIQUE,
		last_message_id INTEGER,
		last_message_time DATETIME,
		unread_count INTEGER NOT NULL DEFAULT 0,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (last_message_id) REFERENCES messages(id) ON DELETE SET NULL
	);

	CREATE TABLE IF NOT EXISTS message_stats (
		id INTEGER PRIMARY KEY,
		total_messages INTEGER NOT NULL DEFAULT 0,
		total_rx INTEGER NOT NULL DEFAULT 0,
		total_tx INTEGER NOT NULL DEFAULT 0,
		last_cleanup DATETIME,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	-- Initialize stats if empty
	INSERT OR IGNORE INTO message_stats (id, total_messages, total_rx, total_tx)
	VALUES (1, 0, 0, 0);
	`

	_, err := ms.db.Exec(schema)
	return err
}

// createIndexes creates database indexes for performance
func (ms *MessageStore) createIndexes() error {
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_messages_timestamp ON messages(timestamp DESC)",
		"CREATE INDEX IF NOT EXISTS idx_messages_from_callsign ON messages(from_callsign)",
		"CREATE INDEX IF NOT EXISTS idx_messages_to_callsign ON messages(to_callsign)",
		"CREATE INDEX IF NOT EXISTS idx_messages_direction ON messages(direction)",
		"CREATE INDEX IF NOT EXISTS idx_messages_is_read ON messages(is_read)",
		"CREATE INDEX IF NOT EXISTS idx_messages_message_type ON messages(message_type)",
		"CREATE INDEX IF NOT EXISTS idx_conversations_callsign ON conversations(callsign)",
		"CREATE INDEX IF NOT EXISTS idx_conversations_last_message_time ON conversations(last_message_time DESC)",
		"CREATE INDEX IF NOT EXISTS idx_conversations_unread_count ON conversations(unread_count)",
	}

	for _, indexSQL := range indexes {
		if _, err := ms.db.Exec(indexSQL); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	return nil
}

// StoreMessage stores a message in the database
func (ms *MessageStore) StoreMessage(msg protocol.Message, direction string, messageType string) error {
	tx, err := ms.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Insert message
	query := `
		INSERT INTO messages (
			timestamp, from_callsign, to_callsign, message_text,
			snr, frequency, mode, direction, message_type
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := tx.Exec(query,
		msg.Timestamp, msg.From, msg.To, msg.Message,
		msg.SNR, msg.Frequency, msg.Mode, direction, messageType,
	)
	if err != nil {
		return fmt.Errorf("failed to insert message: %w", err)
	}

	messageID, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get message ID: %w", err)
	}

	// Update conversation
	if err := ms.updateConversation(tx, msg.From, messageID, msg.Timestamp, direction); err != nil {
		return fmt.Errorf("failed to update conversation: %w", err)
	}

	// Update stats
	if err := ms.updateStats(tx, direction); err != nil {
		return fmt.Errorf("failed to update stats: %w", err)
	}

	// Check if we need to cleanup old messages
	if err := ms.cleanupOldMessages(tx); err != nil {
		log.Printf("Warning: failed to cleanup old messages: %v", err)
	}

	return tx.Commit()
}

// updateConversation updates the conversation record for a callsign
func (ms *MessageStore) updateConversation(tx *sql.Tx, callsign string, messageID int64, timestamp time.Time, direction string) error {
	// Insert or update conversation
	query := `
		INSERT INTO conversations (callsign, last_message_id, last_message_time, unread_count)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(callsign) DO UPDATE SET
			last_message_id = excluded.last_message_id,
			last_message_time = excluded.last_message_time,
			unread_count = CASE
				WHEN excluded.unread_count > 0 AND ? = 'RX' THEN unread_count + 1
				ELSE unread_count
			END,
			updated_at = CURRENT_TIMESTAMP
	`

	unreadIncrement := 0
	if direction == "RX" {
		unreadIncrement = 1
	}

	_, err := tx.Exec(query, callsign, messageID, timestamp, unreadIncrement, direction)
	return err
}

// updateStats updates message statistics
func (ms *MessageStore) updateStats(tx *sql.Tx, direction string) error {
	query := `
		UPDATE message_stats SET
			total_messages = total_messages + 1,
			total_rx = CASE WHEN ? = 'RX' THEN total_rx + 1 ELSE total_rx END,
			total_tx = CASE WHEN ? = 'TX' THEN total_tx + 1 ELSE total_tx END,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = 1
	`

	_, err := tx.Exec(query, direction, direction)
	return err
}

// CleanupOldMessages removes messages beyond the maximum limit (exported for manual cleanup)
func (ms *MessageStore) CleanupOldMessages() error {
	tx, err := ms.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := ms.cleanupOldMessages(tx); err != nil {
		return err
	}

	return tx.Commit()
}

// cleanupOldMessages removes messages beyond the maximum limit
func (ms *MessageStore) cleanupOldMessages(tx *sql.Tx) error {
	if ms.maxMessages <= 0 {
		return nil // No limit
	}

	// Count current messages
	var count int
	err := tx.QueryRow("SELECT COUNT(*) FROM messages").Scan(&count)
	if err != nil {
		return err
	}

	if count <= ms.maxMessages {
		return nil // Within limit
	}

	// Delete oldest messages beyond limit
	deleteCount := count - ms.maxMessages
	query := `
		DELETE FROM messages
		WHERE id IN (
			SELECT id FROM messages
			ORDER BY timestamp ASC
			LIMIT ?
		)
	`

	_, err = tx.Exec(query, deleteCount)
	if err != nil {
		return err
	}

	// Update cleanup timestamp
	_, err = tx.Exec("UPDATE message_stats SET last_cleanup = CURRENT_TIMESTAMP WHERE id = 1")
	return err
}

// Close closes the database connection
func (ms *MessageStore) Close() error {
	if ms.db != nil {
		return ms.db.Close()
	}
	return nil
}