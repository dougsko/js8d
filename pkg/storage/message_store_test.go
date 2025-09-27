package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dougsko/js8d/pkg/protocol"
	_ "github.com/mattn/go-sqlite3"
)

func TestNewMessageStore(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "js8d-storage-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Run("Valid Store Creation", func(t *testing.T) {
		dbPath := filepath.Join(tempDir, "test.db")
		store, err := NewMessageStore(dbPath, 1000)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		defer store.Close()

		if store.dbPath != dbPath {
			t.Errorf("Expected dbPath %s, got %s", dbPath, store.dbPath)
		}
		if store.maxMessages != 1000 {
			t.Errorf("Expected maxMessages 1000, got %d", store.maxMessages)
		}

		// Verify database file was created
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			t.Error("Expected database file to be created")
		}
	})

	t.Run("Store Creation with Nested Directory", func(t *testing.T) {
		dbPath := filepath.Join(tempDir, "nested", "dir", "test.db")
		store, err := NewMessageStore(dbPath, 500)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		defer store.Close()

		// Verify nested directory was created
		if _, err := os.Stat(filepath.Dir(dbPath)); os.IsNotExist(err) {
			t.Error("Expected nested directory to be created")
		}
	})

	t.Run("Invalid Directory Path", func(t *testing.T) {
		// Try to create database in a read-only directory
		dbPath := "/root/readonly/test.db"
		_, err := NewMessageStore(dbPath, 1000)
		if err == nil {
			t.Error("Expected error for invalid directory path, got nil")
		}
	})
}

func TestMessageStoreInitialization(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "js8d-storage-init-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "init_test.db")
	store, err := NewMessageStore(dbPath, 1000)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	t.Run("Tables Created", func(t *testing.T) {
		tables := []string{"messages", "conversations", "message_stats"}
		for _, table := range tables {
			var count int
			err := store.db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&count)
			if err != nil {
				t.Errorf("Failed to check table %s: %v", table, err)
			}
			if count != 1 {
				t.Errorf("Expected table %s to exist, got count %d", table, count)
			}
		}
	})

	t.Run("Indexes Created", func(t *testing.T) {
		expectedIndexes := []string{
			"idx_messages_timestamp",
			"idx_messages_from_callsign",
			"idx_messages_to_callsign",
			"idx_messages_direction",
			"idx_conversations_callsign",
		}

		for _, index := range expectedIndexes {
			var count int
			err := store.db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name=?", index).Scan(&count)
			if err != nil {
				t.Errorf("Failed to check index %s: %v", index, err)
			}
			if count != 1 {
				t.Errorf("Expected index %s to exist, got count %d", index, count)
			}
		}
	})

	t.Run("Stats Initialized", func(t *testing.T) {
		var count int
		err := store.db.QueryRow("SELECT COUNT(*) FROM message_stats").Scan(&count)
		if err != nil {
			t.Errorf("Failed to check stats table: %v", err)
		}
		if count != 1 {
			t.Errorf("Expected 1 row in message_stats, got %d", count)
		}
	})
}

func TestStoreMessage(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "js8d-store-message-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "store_test.db")
	store, err := NewMessageStore(dbPath, 1000)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	testTime := time.Now().Truncate(time.Second)
	testMessage := protocol.Message{
		Timestamp: testTime,
		From:      "K3DEP",
		To:        "N0ABC",
		Message:   "Hello world test",
		SNR:       -12.5,
		Frequency: 14078000,
		Mode:      "JS8",
	}

	t.Run("Store RX Message", func(t *testing.T) {
		err := store.StoreMessage(testMessage, "RX", "MESSAGE")
		if err != nil {
			t.Fatalf("Failed to store message: %v", err)
		}

		// Verify message was stored
		var count int
		err = store.db.QueryRow("SELECT COUNT(*) FROM messages").Scan(&count)
		if err != nil {
			t.Errorf("Failed to count messages: %v", err)
		}
		if count != 1 {
			t.Errorf("Expected 1 message, got %d", count)
		}

		// Verify message content
		var stored protocol.Message
		var direction, messageType string
		err = store.db.QueryRow(`
			SELECT timestamp, from_callsign, to_callsign, message_text,
				   snr, frequency, mode, direction, message_type
			FROM messages WHERE id = 1
		`).Scan(
			&stored.Timestamp, &stored.From, &stored.To, &stored.Message,
			&stored.SNR, &stored.Frequency, &stored.Mode, &direction, &messageType,
		)
		if err != nil {
			t.Fatalf("Failed to retrieve stored message: %v", err)
		}

		if stored.From != testMessage.From {
			t.Errorf("Expected from %s, got %s", testMessage.From, stored.From)
		}
		if stored.To != testMessage.To {
			t.Errorf("Expected to %s, got %s", testMessage.To, stored.To)
		}
		if stored.Message != testMessage.Message {
			t.Errorf("Expected message %s, got %s", testMessage.Message, stored.Message)
		}
		if stored.SNR != testMessage.SNR {
			t.Errorf("Expected SNR %f, got %f", testMessage.SNR, stored.SNR)
		}
		if direction != "RX" {
			t.Errorf("Expected direction RX, got %s", direction)
		}
		if messageType != "MESSAGE" {
			t.Errorf("Expected message type MESSAGE, got %s", messageType)
		}
	})

	t.Run("Store TX Message", func(t *testing.T) {
		txMessage := testMessage
		txMessage.From = "N0ABC"
		txMessage.To = "K3DEP"
		txMessage.Message = "Reply message"

		err := store.StoreMessage(txMessage, "TX", "REPLY")
		if err != nil {
			t.Fatalf("Failed to store TX message: %v", err)
		}

		// Verify total message count
		var count int
		err = store.db.QueryRow("SELECT COUNT(*) FROM messages").Scan(&count)
		if err != nil {
			t.Errorf("Failed to count messages: %v", err)
		}
		if count != 2 {
			t.Errorf("Expected 2 messages, got %d", count)
		}
	})

	t.Run("Conversation Updated", func(t *testing.T) {
		// Check that conversation was created/updated
		var callsign string
		var unreadCount int
		err := store.db.QueryRow(`
			SELECT callsign, unread_count FROM conversations WHERE callsign = ?
		`, "K3DEP").Scan(&callsign, &unreadCount)
		if err != nil {
			t.Fatalf("Failed to get conversation: %v", err)
		}

		if callsign != "K3DEP" {
			t.Errorf("Expected callsign K3DEP, got %s", callsign)
		}
		if unreadCount != 1 {
			t.Errorf("Expected unread count 1, got %d", unreadCount)
		}
	})

	t.Run("Stats Updated", func(t *testing.T) {
		stats, err := store.GetMessageStats()
		if err != nil {
			t.Fatalf("Failed to get stats: %v", err)
		}

		if stats.TotalMessages != 2 {
			t.Errorf("Expected total messages 2, got %d", stats.TotalMessages)
		}
		if stats.TotalRX != 1 {
			t.Errorf("Expected total RX 1, got %d", stats.TotalRX)
		}
		if stats.TotalTX != 1 {
			t.Errorf("Expected total TX 1, got %d", stats.TotalTX)
		}
	})
}

func TestCleanupOldMessages(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "js8d-cleanup-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "cleanup_test.db")
	store, err := NewMessageStore(dbPath, 3) // Small limit for testing
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// Add messages beyond the limit
	baseTime := time.Now().Add(-time.Hour)
	for i := 0; i < 5; i++ {
		msg := protocol.Message{
			Timestamp: baseTime.Add(time.Duration(i) * time.Minute),
			From:      "K3DEP",
			To:        "N0ABC",
			Message:   fmt.Sprintf("Message %d", i+1),
			SNR:       -10.0,
			Frequency: 14078000,
			Mode:      "JS8",
		}
		err := store.StoreMessage(msg, "RX", "MESSAGE")
		if err != nil {
			t.Fatalf("Failed to store message %d: %v", i+1, err)
		}
	}

	t.Run("Automatic Cleanup During Store", func(t *testing.T) {
		// Should have been cleaned up to max limit
		count, err := store.GetMessageCount()
		if err != nil {
			t.Fatalf("Failed to get message count: %v", err)
		}
		if count != 3 {
			t.Errorf("Expected 3 messages after cleanup, got %d", count)
		}

		// Verify newest messages were kept
		messages, err := store.GetRecentMessages(10)
		if err != nil {
			t.Fatalf("Failed to get recent messages: %v", err)
		}
		if len(messages) != 3 {
			t.Errorf("Expected 3 recent messages, got %d", len(messages))
		}

		// Should have messages 3, 4, 5 (newest)
		expectedMessages := []string{"Message 5", "Message 4", "Message 3"}
		for i, msg := range messages {
			if msg.Message != expectedMessages[i] {
				t.Errorf("Expected message %s, got %s", expectedMessages[i], msg.Message)
			}
		}
	})

	t.Run("Manual Cleanup", func(t *testing.T) {
		// Add one more message
		msg := protocol.Message{
			Timestamp: time.Now(),
			From:      "K3DEP",
			To:        "N0ABC",
			Message:   "Message 6",
			SNR:       -10.0,
			Frequency: 14078000,
			Mode:      "JS8",
		}
		err := store.StoreMessage(msg, "RX", "MESSAGE")
		if err != nil {
			t.Fatalf("Failed to store additional message: %v", err)
		}

		// Should still have 3 messages
		count, err := store.GetMessageCount()
		if err != nil {
			t.Fatalf("Failed to get message count: %v", err)
		}
		if count != 3 {
			t.Errorf("Expected 3 messages after additional store, got %d", count)
		}
	})

	t.Run("No Cleanup When Under Limit", func(t *testing.T) {
		// Create new store with higher limit
		dbPath2 := filepath.Join(tempDir, "no_cleanup_test.db")
		store2, err := NewMessageStore(dbPath2, 10)
		if err != nil {
			t.Fatalf("Failed to create store: %v", err)
		}
		defer store2.Close()

		// Add 3 messages
		for i := 0; i < 3; i++ {
			msg := protocol.Message{
				Timestamp: time.Now().Add(time.Duration(i) * time.Minute),
				From:      "K3DEP",
				To:        "N0ABC",
				Message:   fmt.Sprintf("Test %d", i+1),
				SNR:       -10.0,
				Frequency: 14078000,
				Mode:      "JS8",
			}
			err := store2.StoreMessage(msg, "RX", "MESSAGE")
			if err != nil {
				t.Fatalf("Failed to store message: %v", err)
			}
		}

		count, err := store2.GetMessageCount()
		if err != nil {
			t.Fatalf("Failed to get message count: %v", err)
		}
		if count != 3 {
			t.Errorf("Expected 3 messages (no cleanup), got %d", count)
		}
	})
}

func TestMessageStoreClose(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "js8d-close-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "close_test.db")
	store, err := NewMessageStore(dbPath, 1000)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	t.Run("Close Successfully", func(t *testing.T) {
		err := store.Close()
		if err != nil {
			t.Errorf("Expected no error on close, got: %v", err)
		}
	})

	t.Run("Close Nil Database", func(t *testing.T) {
		store.db = nil
		err := store.Close()
		if err != nil {
			t.Errorf("Expected no error closing nil database, got: %v", err)
		}
	})
}

func TestMessageStoreIntegration(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "js8d-integration-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "integration_test.db")
	store, err := NewMessageStore(dbPath, 1000)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	t.Run("Full Message Lifecycle", func(t *testing.T) {
		// Store multiple messages
		messages := []protocol.Message{
			{
				Timestamp: time.Now().Add(-3 * time.Minute),
				From:      "K3DEP",
				To:        "N0ABC",
				Message:   "CQ CQ DE K3DEP",
				SNR:       -8.5,
				Frequency: 14078000,
				Mode:      "JS8",
			},
			{
				Timestamp: time.Now().Add(-2 * time.Minute),
				From:      "N0ABC",
				To:        "K3DEP",
				Message:   "K3DEP DE N0ABC",
				SNR:       -12.0,
				Frequency: 14078000,
				Mode:      "JS8",
			},
			{
				Timestamp: time.Now().Add(-1 * time.Minute),
				From:      "K3DEP",
				To:        "N0ABC",
				Message:   "N0ABC DE K3DEP FN20",
				SNR:       -6.5,
				Frequency: 14078000,
				Mode:      "JS8",
			},
		}

		directions := []string{"TX", "RX", "TX"}
		for i, msg := range messages {
			err := store.StoreMessage(msg, directions[i], "MESSAGE")
			if err != nil {
				t.Fatalf("Failed to store message %d: %v", i+1, err)
			}
		}

		// Verify all messages stored
		count, err := store.GetMessageCount()
		if err != nil {
			t.Fatalf("Failed to get count: %v", err)
		}
		if count != 3 {
			t.Errorf("Expected 3 messages, got %d", count)
		}

		// Verify stats updated correctly
		stats, err := store.GetMessageStats()
		if err != nil {
			t.Fatalf("Failed to get stats: %v", err)
		}
		if stats.TotalMessages != 3 {
			t.Errorf("Expected total 3, got %d", stats.TotalMessages)
		}
		if stats.TotalRX != 1 {
			t.Errorf("Expected RX 1, got %d", stats.TotalRX)
		}
		if stats.TotalTX != 2 {
			t.Errorf("Expected TX 2, got %d", stats.TotalTX)
		}

		// Verify conversation tracking
		conversations, err := store.GetConversations(10)
		if err != nil {
			t.Fatalf("Failed to get conversations: %v", err)
		}
		if len(conversations) != 2 { // K3DEP and N0ABC
			t.Errorf("Expected 2 conversations, got %d", len(conversations))
		}
	})
}