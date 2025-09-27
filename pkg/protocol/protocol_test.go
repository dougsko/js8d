package protocol

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestParseCommand(t *testing.T) {
	t.Run("STATUS Command", func(t *testing.T) {
		cmd, err := ParseCommand("STATUS")
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if cmd.Type != "STATUS" {
			t.Errorf("Expected type STATUS, got %s", cmd.Type)
		}
		if len(cmd.Args) != 0 {
			t.Errorf("Expected no args for STATUS, got %d", len(cmd.Args))
		}
	})

	t.Run("SEND Command with To and Message", func(t *testing.T) {
		cmd, err := ParseCommand("SEND:N0CALL Hello world test")
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if cmd.Type != "SEND" {
			t.Errorf("Expected type SEND, got %s", cmd.Type)
		}
		if cmd.Args["to"] != "N0CALL" {
			t.Errorf("Expected to N0CALL, got %v", cmd.Args["to"])
		}
		if cmd.Args["message"] != "Hello world test" {
			t.Errorf("Expected message 'Hello world test', got %v", cmd.Args["message"])
		}
	})

	t.Run("SEND Command Message Only", func(t *testing.T) {
		cmd, err := ParseCommand("SEND:CQ CQ DE K3DEP")
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if cmd.Type != "SEND" {
			t.Errorf("Expected type SEND, got %s", cmd.Type)
		}
		// The parsing logic treats the first word as "to" if there are multiple words
		// For broadcast messages like CQ, this behavior is expected
		if cmd.Args["to"] != "CQ" {
			t.Errorf("Expected to field 'CQ', got %v", cmd.Args["to"])
		}
		if cmd.Args["message"] != "CQ DE K3DEP" {
			t.Errorf("Expected message 'CQ DE K3DEP', got %v", cmd.Args["message"])
		}
	})

	t.Run("MESSAGES Command with Limit", func(t *testing.T) {
		cmd, err := ParseCommand("MESSAGES:20")
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if cmd.Type != "MESSAGES" {
			t.Errorf("Expected type MESSAGES, got %s", cmd.Type)
		}
		if cmd.Args["limit"] != "20" {
			t.Errorf("Expected limit 20, got %v", cmd.Args["limit"])
		}
	})

	t.Run("MESSAGES Command with Since", func(t *testing.T) {
		cmd, err := ParseCommand("MESSAGES:since:123456789")
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if cmd.Type != "MESSAGES" {
			t.Errorf("Expected type MESSAGES, got %s", cmd.Type)
		}
		if cmd.Args["since"] != "123456789" {
			t.Errorf("Expected since 123456789, got %v", cmd.Args["since"])
		}
	})

	t.Run("FREQUENCY Command", func(t *testing.T) {
		cmd, err := ParseCommand("FREQUENCY:14078000")
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if cmd.Type != "FREQUENCY" {
			t.Errorf("Expected type FREQUENCY, got %s", cmd.Type)
		}
		if cmd.Args["frequency"] != "14078000" {
			t.Errorf("Expected frequency 14078000, got %v", cmd.Args["frequency"])
		}
	})

	t.Run("CONFIG Command Set", func(t *testing.T) {
		cmd, err := ParseCommand("CONFIG:set:callsign:K3DEP")
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if cmd.Type != "CONFIG" {
			t.Errorf("Expected type CONFIG, got %s", cmd.Type)
		}
		if cmd.Args["action"] != "set" {
			t.Errorf("Expected action set, got %v", cmd.Args["action"])
		}
		if cmd.Args["key"] != "callsign" {
			t.Errorf("Expected key callsign, got %v", cmd.Args["key"])
		}
		if cmd.Args["value"] != "K3DEP" {
			t.Errorf("Expected value K3DEP, got %v", cmd.Args["value"])
		}
	})

	t.Run("CONFIG Command Get", func(t *testing.T) {
		cmd, err := ParseCommand("CONFIG:get:callsign")
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if cmd.Type != "CONFIG" {
			t.Errorf("Expected type CONFIG, got %s", cmd.Type)
		}
		if cmd.Args["action"] != "get" {
			t.Errorf("Expected action get, got %v", cmd.Args["action"])
		}
		if cmd.Args["key"] != "callsign" {
			t.Errorf("Expected key callsign, got %v", cmd.Args["key"])
		}
		if _, exists := cmd.Args["value"]; exists {
			t.Errorf("Expected no value for get command, got %v", cmd.Args["value"])
		}
	})

	t.Run("Simple Commands", func(t *testing.T) {
		commands := []string{"QUIT", "PING", "RADIO", "AUDIO", "ABORT", "RELOAD"}
		for _, cmdText := range commands {
			t.Run(cmdText, func(t *testing.T) {
				cmd, err := ParseCommand(cmdText)
				if err != nil {
					t.Fatalf("Expected no error for %s, got: %v", cmdText, err)
				}
				if cmd.Type != cmdText {
					t.Errorf("Expected type %s, got %s", cmdText, cmd.Type)
				}
				if len(cmd.Args) != 0 {
					t.Errorf("Expected no args for %s, got %d", cmdText, len(cmd.Args))
				}
			})
		}
	})

	t.Run("Case Insensitive", func(t *testing.T) {
		cmd, err := ParseCommand("status")
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if cmd.Type != "STATUS" {
			t.Errorf("Expected uppercase STATUS, got %s", cmd.Type)
		}
	})

	t.Run("Whitespace Handling", func(t *testing.T) {
		cmd, err := ParseCommand("  PING  ")
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if cmd.Type != "PING" {
			t.Errorf("Expected type PING, got %s", cmd.Type)
		}
	})

	t.Run("Unknown Command", func(t *testing.T) {
		cmd, err := ParseCommand("UNKNOWN:test")
		if err != nil {
			t.Fatalf("Expected no error for unknown command, got: %v", err)
		}
		if cmd.Type != "UNKNOWN" {
			t.Errorf("Expected type UNKNOWN, got %s", cmd.Type)
		}
		// Unknown commands should not parse args specially
		if len(cmd.Args) != 0 {
			t.Errorf("Expected no args for unknown command, got %d", len(cmd.Args))
		}
	})

	t.Run("Empty Command", func(t *testing.T) {
		cmd, err := ParseCommand("")
		if err != nil {
			t.Fatalf("Expected no error for empty command, got: %v", err)
		}
		if cmd.Type != "" {
			t.Errorf("Expected empty type, got %s", cmd.Type)
		}
	})
}

func TestResponse(t *testing.T) {
	t.Run("Success Response JSON", func(t *testing.T) {
		data := map[string]interface{}{
			"callsign":  "K3DEP",
			"frequency": 14078000,
			"connected": true,
		}
		resp := NewSuccessResponse(data)

		if !resp.Success {
			t.Error("Expected success to be true")
		}
		if resp.Error != "" {
			t.Errorf("Expected no error, got %s", resp.Error)
		}
		if resp.Data["callsign"] != "K3DEP" {
			t.Errorf("Expected callsign K3DEP, got %v", resp.Data["callsign"])
		}

		jsonStr := resp.String()
		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
			t.Fatalf("Failed to parse JSON: %v", err)
		}

		if parsed["success"] != true {
			t.Error("Expected success true in JSON")
		}
		if parsed["data"] == nil {
			t.Error("Expected data in JSON")
		}
	})

	t.Run("Error Response JSON", func(t *testing.T) {
		resp := NewErrorResponse("invalid command")

		if resp.Success {
			t.Error("Expected success to be false")
		}
		if resp.Error != "invalid command" {
			t.Errorf("Expected error 'invalid command', got %s", resp.Error)
		}
		if resp.Data != nil {
			t.Errorf("Expected no data for error response, got %v", resp.Data)
		}

		jsonStr := resp.String()
		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
			t.Fatalf("Failed to parse JSON: %v", err)
		}

		if parsed["success"] != false {
			t.Error("Expected success false in JSON")
		}
		if parsed["error"] != "invalid command" {
			t.Errorf("Expected error in JSON, got %v", parsed["error"])
		}
	})

	t.Run("Empty Success Response", func(t *testing.T) {
		resp := NewSuccessResponse(nil)
		jsonStr := resp.String()

		// Should still be valid JSON
		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
			t.Fatalf("Failed to parse JSON: %v", err)
		}

		if parsed["success"] != true {
			t.Error("Expected success true in JSON")
		}
	})

	t.Run("Response with Complex Data", func(t *testing.T) {
		data := map[string]interface{}{
			"messages": []map[string]interface{}{
				{
					"id":        1,
					"from":      "N0ABC",
					"message":   "Hello world",
					"timestamp": "2023-01-01T12:00:00Z",
				},
				{
					"id":        2,
					"from":      "K3DEF",
					"message":   "73",
					"timestamp": "2023-01-01T12:01:00Z",
				},
			},
			"count": 2,
		}
		resp := NewSuccessResponse(data)
		jsonStr := resp.String()

		// Should be valid JSON with nested structures
		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
			t.Fatalf("Failed to parse JSON: %v", err)
		}

		dataField := parsed["data"].(map[string]interface{})
		if dataField["count"] != float64(2) { // JSON numbers become float64
			t.Errorf("Expected count 2, got %v", dataField["count"])
		}
	})
}

func TestMessage(t *testing.T) {
	t.Run("Message JSON Serialization", func(t *testing.T) {
		timestamp := time.Now()
		msg := Message{
			ID:        123,
			Timestamp: timestamp,
			From:      "K3DEP",
			To:        "N0ABC",
			Message:   "Hello test",
			SNR:       -12.5,
			Frequency: 14078000,
			Mode:      "JS8",
		}

		data, err := json.Marshal(msg)
		if err != nil {
			t.Fatalf("Failed to marshal message: %v", err)
		}

		var parsed Message
		if err := json.Unmarshal(data, &parsed); err != nil {
			t.Fatalf("Failed to unmarshal message: %v", err)
		}

		if parsed.ID != 123 {
			t.Errorf("Expected ID 123, got %d", parsed.ID)
		}
		if parsed.From != "K3DEP" {
			t.Errorf("Expected from K3DEP, got %s", parsed.From)
		}
		if parsed.To != "N0ABC" {
			t.Errorf("Expected to N0ABC, got %s", parsed.To)
		}
		if parsed.Message != "Hello test" {
			t.Errorf("Expected message 'Hello test', got %s", parsed.Message)
		}
		if parsed.SNR != -12.5 {
			t.Errorf("Expected SNR -12.5, got %f", parsed.SNR)
		}
		if parsed.Frequency != 14078000 {
			t.Errorf("Expected frequency 14078000, got %d", parsed.Frequency)
		}
		if parsed.Mode != "JS8" {
			t.Errorf("Expected mode JS8, got %s", parsed.Mode)
		}
	})
}

func TestStatus(t *testing.T) {
	t.Run("Status JSON Serialization", func(t *testing.T) {
		startTime := time.Now()
		status := Status{
			Callsign:  "K3DEP",
			Grid:      "FN20",
			Frequency: 14078000,
			Mode:      "JS8",
			PTT:       false,
			Connected: true,
			Uptime:    "1h30m",
			StartTime: startTime,
			Version:   "0.1.0",
		}

		data, err := json.Marshal(status)
		if err != nil {
			t.Fatalf("Failed to marshal status: %v", err)
		}

		var parsed Status
		if err := json.Unmarshal(data, &parsed); err != nil {
			t.Fatalf("Failed to unmarshal status: %v", err)
		}

		if parsed.Callsign != "K3DEP" {
			t.Errorf("Expected callsign K3DEP, got %s", parsed.Callsign)
		}
		if parsed.Grid != "FN20" {
			t.Errorf("Expected grid FN20, got %s", parsed.Grid)
		}
		if parsed.Frequency != 14078000 {
			t.Errorf("Expected frequency 14078000, got %d", parsed.Frequency)
		}
		if parsed.PTT != false {
			t.Errorf("Expected PTT false, got %t", parsed.PTT)
		}
		if parsed.Connected != true {
			t.Errorf("Expected connected true, got %t", parsed.Connected)
		}
	})
}

func TestConstants(t *testing.T) {
	// Test that all command constants are defined
	expectedCommands := []string{
		"STATUS", "MESSAGES", "SEND", "FREQUENCY", "CONFIG",
		"QUIT", "PING", "RADIO", "AUDIO", "ABORT", "RELOAD",
	}

	constants := map[string]string{
		"STATUS":    CmdStatus,
		"MESSAGES":  CmdMessages,
		"SEND":      CmdSend,
		"FREQUENCY": CmdFrequency,
		"CONFIG":    CmdConfig,
		"QUIT":      CmdQuit,
		"PING":      CmdPing,
		"RADIO":     CmdRadio,
		"AUDIO":     CmdAudio,
		"ABORT":     CmdAbort,
		"RELOAD":    CmdReload,
	}

	for _, expected := range expectedCommands {
		if constant, exists := constants[expected]; !exists {
			t.Errorf("Missing constant for command %s", expected)
		} else if constant != expected {
			t.Errorf("Expected constant %s to equal %s, got %s", expected, expected, constant)
		}
	}
}

func TestProtocolIntegration(t *testing.T) {
	// Test a complete protocol flow: parse command -> generate response -> serialize
	t.Run("Complete Flow", func(t *testing.T) {
		// Parse a command
		cmd, err := ParseCommand("SEND:N0ABC Test message from integration test")
		if err != nil {
			t.Fatalf("Failed to parse command: %v", err)
		}

		// Simulate processing and create response
		responseData := map[string]interface{}{
			"status":  "queued",
			"message": map[string]interface{}{
				"id":      456,
				"from":    "K3DEP",
				"to":      cmd.Args["to"],
				"message": cmd.Args["message"],
				"mode":    "JS8",
			},
		}
		resp := NewSuccessResponse(responseData)

		// Serialize response
		jsonStr := resp.String()

		// Verify the complete flow
		if !strings.Contains(jsonStr, "queued") {
			t.Error("Expected 'queued' in response JSON")
		}
		if !strings.Contains(jsonStr, "N0ABC") {
			t.Error("Expected 'N0ABC' in response JSON")
		}
		if !strings.Contains(jsonStr, "Test message from integration test") {
			t.Error("Expected test message in response JSON")
		}

		// Verify it's valid JSON
		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
			t.Fatalf("Response is not valid JSON: %v", err)
		}
	})

	t.Run("Error Flow", func(t *testing.T) {
		// Test error response flow
		resp := NewErrorResponse("command parsing failed: invalid syntax")
		jsonStr := resp.String()

		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
			t.Fatalf("Error response is not valid JSON: %v", err)
		}

		if parsed["success"] != false {
			t.Error("Expected success false for error response")
		}
		if !strings.Contains(parsed["error"].(string), "command parsing failed") {
			t.Error("Expected error message in response")
		}
	})
}