package storage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dougsko/js8d/pkg/protocol"
)

func setupTestStore(t *testing.T) (*MessageStore, func()) {
	tempDir, err := os.MkdirTemp("", "js8d-queries-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	dbPath := filepath.Join(tempDir, "queries_test.db")
	store, err := NewMessageStore(dbPath, 1000)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	cleanup := func() {
		store.Close()
		os.RemoveAll(tempDir)
	}

	return store, cleanup
}

func seedTestMessages(t *testing.T, store *MessageStore) {
	baseTime := time.Now().Add(-10 * time.Minute)
	messages := []struct {
		msg       protocol.Message
		direction string
		msgType   string
	}{
		{
			msg: protocol.Message{
				Timestamp: baseTime.Add(1 * time.Minute),
				From:      "K3DEP",
				To:        "",
				Message:   "CQ CQ DE K3DEP FN20",
				SNR:       -8.5,
				Frequency: 14078000,
				Mode:      "JS8",
			},
			direction: "TX",
			msgType:   "CQ",
		},
		{
			msg: protocol.Message{
				Timestamp: baseTime.Add(2 * time.Minute),
				From:      "N0ABC",
				To:        "K3DEP",
				Message:   "K3DEP DE N0ABC EM12",
				SNR:       -12.0,
				Frequency: 14078000,
				Mode:      "JS8",
			},
			direction: "RX",
			msgType:   "REPLY",
		},
		{
			msg: protocol.Message{
				Timestamp: baseTime.Add(3 * time.Minute),
				From:      "K3DEP",
				To:        "N0ABC",
				Message:   "N0ABC DE K3DEP Thanks for the contact",
				SNR:       -6.5,
				Frequency: 14078000,
				Mode:      "JS8",
			},
			direction: "TX",
			msgType:   "MESSAGE",
		},
		{
			msg: protocol.Message{
				Timestamp: baseTime.Add(4 * time.Minute),
				From:      "W1XYZ",
				To:        "",
				Message:   "CQ CQ DE W1XYZ W1XYZ K",
				SNR:       -15.2,
				Frequency: 14078000,
				Mode:      "JS8",
			},
			direction: "RX",
			msgType:   "CQ",
		},
		{
			msg: protocol.Message{
				Timestamp: baseTime.Add(5 * time.Minute),
				From:      "N0ABC",
				To:        "K3DEP",
				Message:   "K3DEP DE N0ABC 73",
				SNR:       -10.8,
				Frequency: 14078000,
				Mode:      "JS8",
			},
			direction: "RX",
			msgType:   "MESSAGE",
		},
	}

	for i, data := range messages {
		err := store.StoreMessage(data.msg, data.direction, data.msgType)
		if err != nil {
			t.Fatalf("Failed to seed message %d: %v", i+1, err)
		}
	}
}

func TestGetMessages(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	seedTestMessages(t, store)

	t.Run("Get All Messages", func(t *testing.T) {
		query := MessageQuery{}
		messages, err := store.GetMessages(query)
		if err != nil {
			t.Fatalf("Failed to get messages: %v", err)
		}

		if len(messages) != 5 {
			t.Errorf("Expected 5 messages, got %d", len(messages))
		}

		// Should be ordered by timestamp DESC (newest first)
		for i := 1; i < len(messages); i++ {
			if messages[i].Timestamp.After(messages[i-1].Timestamp) {
				t.Error("Messages not ordered by timestamp DESC")
			}
		}
	})

	t.Run("Get Messages with Limit", func(t *testing.T) {
		query := MessageQuery{Limit: 3}
		messages, err := store.GetMessages(query)
		if err != nil {
			t.Fatalf("Failed to get messages: %v", err)
		}

		if len(messages) != 3 {
			t.Errorf("Expected 3 messages, got %d", len(messages))
		}
	})

	t.Run("Get Messages with Limit and Offset", func(t *testing.T) {
		query := MessageQuery{Limit: 2, Offset: 2}
		messages, err := store.GetMessages(query)
		if err != nil {
			t.Fatalf("Failed to get messages: %v", err)
		}

		if len(messages) != 2 {
			t.Errorf("Expected 2 messages, got %d", len(messages))
		}
	})

	t.Run("Get Messages Since Time", func(t *testing.T) {
		since := time.Now().Add(-7 * time.Minute)
		query := MessageQuery{Since: &since}
		messages, err := store.GetMessages(query)
		if err != nil {
			t.Fatalf("Failed to get messages since: %v", err)
		}

		// Should get the messages more recent than 7 minutes ago
		// Based on seed data: messages at -5, -4, -3 minutes (3 messages)
		// But timing might be slightly off, so check for >= 2
		if len(messages) < 2 {
			t.Errorf("Expected at least 2 messages since time, got %d", len(messages))
		}

		for _, msg := range messages {
			if msg.Timestamp.Before(since) {
				t.Errorf("Message timestamp %v is before since time %v", msg.Timestamp, since)
			}
		}
	})

	t.Run("Get Messages Until Time", func(t *testing.T) {
		until := time.Now().Add(-7 * time.Minute)
		query := MessageQuery{Until: &until}
		messages, err := store.GetMessages(query)
		if err != nil {
			t.Fatalf("Failed to get messages until: %v", err)
		}

		// Should get the oldest messages (before 7 minutes ago)
		// Based on seed data: messages at -10, -9, -8 minutes, so timing flexible
		if len(messages) < 2 {
			t.Errorf("Expected at least 2 messages until time, got %d", len(messages))
		}

		for _, msg := range messages {
			if msg.Timestamp.After(until) {
				t.Errorf("Message timestamp %v is after until time %v", msg.Timestamp, until)
			}
		}
	})

	t.Run("Get Messages by Callsign", func(t *testing.T) {
		query := MessageQuery{Callsign: "N0ABC"}
		messages, err := store.GetMessages(query)
		if err != nil {
			t.Fatalf("Failed to get messages by callsign: %v", err)
		}

		// N0ABC appears in 3 messages: 2 from N0ABC, 1 to N0ABC
		if len(messages) != 3 {
			t.Errorf("Expected 3 messages for N0ABC, got %d", len(messages))
		}

		for _, msg := range messages {
			if msg.From != "N0ABC" && msg.To != "N0ABC" {
				t.Errorf("Message doesn't involve N0ABC: from=%s, to=%s", msg.From, msg.To)
			}
		}
	})

	t.Run("Get Messages by Direction", func(t *testing.T) {
		query := MessageQuery{Direction: "RX"}
		messages, err := store.GetMessages(query)
		if err != nil {
			t.Fatalf("Failed to get RX messages: %v", err)
		}

		if len(messages) != 3 {
			t.Errorf("Expected 3 RX messages, got %d", len(messages))
		}
	})

	t.Run("Get Messages by Type", func(t *testing.T) {
		query := MessageQuery{MessageType: "CQ"}
		messages, err := store.GetMessages(query)
		if err != nil {
			t.Fatalf("Failed to get CQ messages: %v", err)
		}

		if len(messages) != 2 {
			t.Errorf("Expected 2 CQ messages, got %d", len(messages))
		}
	})

	t.Run("Complex Query", func(t *testing.T) {
		since := time.Now().Add(-8 * time.Minute)
		query := MessageQuery{
			Since:     &since,
			Direction: "TX",
			Limit:     10,
		}
		messages, err := store.GetMessages(query)
		if err != nil {
			t.Fatalf("Failed to get complex query: %v", err)
		}

		// Should get 1 TX message since the time
		if len(messages) != 1 {
			t.Errorf("Expected 1 message for complex query, got %d", len(messages))
		}

		if messages[0].From != "K3DEP" {
			t.Errorf("Expected message from K3DEP, got %s", messages[0].From)
		}
	})
}

func TestGetConversations(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	seedTestMessages(t, store)

	t.Run("Get All Conversations", func(t *testing.T) {
		conversations, err := store.GetConversations(0)
		if err != nil {
			t.Fatalf("Failed to get conversations: %v", err)
		}

		// Should have 3 conversations: K3DEP, N0ABC, W1XYZ
		if len(conversations) != 3 {
			t.Errorf("Expected 3 conversations, got %d", len(conversations))
		}

		// Should be ordered by last message time DESC
		for i := 1; i < len(conversations); i++ {
			if conversations[i].LastMessageTime.After(conversations[i-1].LastMessageTime) {
				t.Error("Conversations not ordered by last message time DESC")
			}
		}
	})

	t.Run("Get Limited Conversations", func(t *testing.T) {
		conversations, err := store.GetConversations(2)
		if err != nil {
			t.Fatalf("Failed to get limited conversations: %v", err)
		}

		if len(conversations) != 2 {
			t.Errorf("Expected 2 conversations, got %d", len(conversations))
		}
	})

	t.Run("Conversation Content", func(t *testing.T) {
		conversations, err := store.GetConversations(0)
		if err != nil {
			t.Fatalf("Failed to get conversations: %v", err)
		}

		// Find N0ABC conversation
		var n0abcConv *ConversationSummary
		for _, conv := range conversations {
			if conv.Callsign == "N0ABC" {
				n0abcConv = &conv
				break
			}
		}

		if n0abcConv == nil {
			t.Fatal("N0ABC conversation not found")
		}

		if n0abcConv.TotalMessages != 3 {
			t.Errorf("Expected 3 total messages for N0ABC, got %d", n0abcConv.TotalMessages)
		}

		if n0abcConv.UnreadCount != 2 {
			t.Errorf("Expected 2 unread messages for N0ABC, got %d", n0abcConv.UnreadCount)
		}

		if !strings.Contains(n0abcConv.LastMessageText, "73") {
			t.Errorf("Expected last message to contain '73', got: %s", n0abcConv.LastMessageText)
		}
	})
}

func TestMessagesByCallsign(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	seedTestMessages(t, store)

	t.Run("Get Messages for Specific Callsign", func(t *testing.T) {
		messages, err := store.GetMessagesByCallsign("K3DEP", 10, 0)
		if err != nil {
			t.Fatalf("Failed to get messages for K3DEP: %v", err)
		}

		// K3DEP appears in 4 messages: 2 from K3DEP, 2 to K3DEP
		if len(messages) != 4 {
			t.Errorf("Expected 4 messages for K3DEP, got %d", len(messages))
		}

		for _, msg := range messages {
			if msg.From != "K3DEP" && msg.To != "K3DEP" {
				t.Errorf("Message doesn't involve K3DEP: from=%s, to=%s", msg.From, msg.To)
			}
		}
	})

	t.Run("Get Messages with Pagination", func(t *testing.T) {
		messages, err := store.GetMessagesByCallsign("K3DEP", 2, 1)
		if err != nil {
			t.Fatalf("Failed to get paginated messages: %v", err)
		}

		if len(messages) != 2 {
			t.Errorf("Expected 2 messages with pagination, got %d", len(messages))
		}
	})
}

func TestRecentMessages(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	seedTestMessages(t, store)

	messages, err := store.GetRecentMessages(3)
	if err != nil {
		t.Fatalf("Failed to get recent messages: %v", err)
	}

	if len(messages) != 3 {
		t.Errorf("Expected 3 recent messages, got %d", len(messages))
	}

	// Should be the 3 newest messages
	expectedCallsigns := []string{"N0ABC", "W1XYZ", "K3DEP"}
	for i, msg := range messages {
		if msg.From != expectedCallsigns[i] {
			t.Errorf("Expected message %d from %s, got %s", i+1, expectedCallsigns[i], msg.From)
		}
	}
}

func TestUnreadMessages(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	seedTestMessages(t, store)

	t.Run("Get All Unread Messages", func(t *testing.T) {
		messages, err := store.GetUnreadMessages()
		if err != nil {
			t.Fatalf("Failed to get unread messages: %v", err)
		}

		// All messages should be unread initially
		if len(messages) != 5 {
			t.Errorf("Expected 5 unread messages, got %d", len(messages))
		}
	})

	t.Run("Mark Messages as Read", func(t *testing.T) {
		err := store.MarkMessagesAsRead("N0ABC")
		if err != nil {
			t.Fatalf("Failed to mark messages as read: %v", err)
		}

		// Check unread count
		count, err := store.GetUnreadCount()
		if err != nil {
			t.Fatalf("Failed to get unread count: %v", err)
		}

		// Should have 2 unread messages left (5 total - 3 involving N0ABC but only 2 RX)
		if count != 2 {
			t.Errorf("Expected 2 unread messages after marking N0ABC as read, got %d", count)
		}

		// Check conversation unread count reset
		conversations, err := store.GetConversations(0)
		if err != nil {
			t.Fatalf("Failed to get conversations: %v", err)
		}

		for _, conv := range conversations {
			if conv.Callsign == "N0ABC" && conv.UnreadCount != 0 {
				t.Errorf("Expected N0ABC unread count 0, got %d", conv.UnreadCount)
			}
		}
	})
}

func TestSearchMessages(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	seedTestMessages(t, store)

	t.Run("Search by Message Content", func(t *testing.T) {
		messages, err := store.SearchMessages("CQ", 10)
		if err != nil {
			t.Fatalf("Failed to search messages: %v", err)
		}

		if len(messages) != 2 {
			t.Errorf("Expected 2 messages containing 'CQ', got %d", len(messages))
		}

		for _, msg := range messages {
			if !strings.Contains(strings.ToUpper(msg.Message), "CQ") {
				t.Errorf("Message doesn't contain 'CQ': %s", msg.Message)
			}
		}
	})

	t.Run("Search with Limit", func(t *testing.T) {
		messages, err := store.SearchMessages("DE", 2)
		if err != nil {
			t.Fatalf("Failed to search with limit: %v", err)
		}

		if len(messages) != 2 {
			t.Errorf("Expected 2 messages with limit, got %d", len(messages))
		}
	})

	t.Run("Search No Results", func(t *testing.T) {
		messages, err := store.SearchMessages("NONEXISTENT", 10)
		if err != nil {
			t.Fatalf("Failed to search for nonexistent term: %v", err)
		}

		if len(messages) != 0 {
			t.Errorf("Expected 0 messages for nonexistent search, got %d", len(messages))
		}
	})

	t.Run("Case Insensitive Search", func(t *testing.T) {
		messages, err := store.SearchMessages("thanks", 10)
		if err != nil {
			t.Fatalf("Failed to search case insensitive: %v", err)
		}

		if len(messages) != 1 {
			t.Errorf("Expected 1 message containing 'thanks', got %d", len(messages))
		}
	})
}

func TestMessageStats(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	seedTestMessages(t, store)

	stats, err := store.GetMessageStats()
	if err != nil {
		t.Fatalf("Failed to get message stats: %v", err)
	}

	if stats.TotalMessages != 5 {
		t.Errorf("Expected total messages 5, got %d", stats.TotalMessages)
	}

	if stats.TotalRX != 3 {
		t.Errorf("Expected total RX 3, got %d", stats.TotalRX)
	}

	if stats.TotalTX != 2 {
		t.Errorf("Expected total TX 2, got %d", stats.TotalTX)
	}
}

func TestMessageCounts(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	seedTestMessages(t, store)

	t.Run("Get Total Message Count", func(t *testing.T) {
		count, err := store.GetMessageCount()
		if err != nil {
			t.Fatalf("Failed to get message count: %v", err)
		}

		if count != 5 {
			t.Errorf("Expected message count 5, got %d", count)
		}
	})

	t.Run("Get Unread Count", func(t *testing.T) {
		count, err := store.GetUnreadCount()
		if err != nil {
			t.Fatalf("Failed to get unread count: %v", err)
		}

		if count != 5 {
			t.Errorf("Expected unread count 5, got %d", count)
		}
	})

	t.Run("Unread Count After Marking Read", func(t *testing.T) {
		err := store.MarkMessagesAsRead("K3DEP")
		if err != nil {
			t.Fatalf("Failed to mark messages as read: %v", err)
		}

		count, err := store.GetUnreadCount()
		if err != nil {
			t.Fatalf("Failed to get unread count after marking read: %v", err)
		}

		// Should have 1 unread message left (only W1XYZ)
		if count != 1 {
			t.Errorf("Expected unread count 1 after marking K3DEP as read, got %d", count)
		}
	})
}

func TestQueryIntegration(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	seedTestMessages(t, store)

	t.Run("Complete Query Workflow", func(t *testing.T) {
		// Get all conversations
		conversations, err := store.GetConversations(0)
		if err != nil {
			t.Fatalf("Failed to get conversations: %v", err)
		}

		if len(conversations) != 3 {
			t.Errorf("Expected 3 conversations, got %d", len(conversations))
		}

		// Get messages for most active conversation
		mostActive := conversations[0]
		messages, err := store.GetMessagesByCallsign(mostActive.Callsign, 10, 0)
		if err != nil {
			t.Fatalf("Failed to get messages for most active: %v", err)
		}

		if len(messages) < 1 {
			t.Error("Expected at least 1 message for most active conversation")
		}

		// Mark as read
		err = store.MarkMessagesAsRead(mostActive.Callsign)
		if err != nil {
			t.Fatalf("Failed to mark as read: %v", err)
		}

		// Verify unread count decreased
		newCount, err := store.GetUnreadCount()
		if err != nil {
			t.Fatalf("Failed to get new unread count: %v", err)
		}

		if newCount >= 5 {
			t.Errorf("Expected unread count to decrease from 5, got %d", newCount)
		}

		// Search for specific content
		searchResults, err := store.SearchMessages("73", 10)
		if err != nil {
			t.Fatalf("Failed to search: %v", err)
		}

		if len(searchResults) != 1 {
			t.Errorf("Expected 1 search result for '73', got %d", len(searchResults))
		}
	})
}