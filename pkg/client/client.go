package client

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/js8call/js8d/pkg/protocol"
)

// SocketClient represents a client connection to the core engine
type SocketClient struct {
	socketPath string
	timeout    time.Duration
}

// NewSocketClient creates a new socket client
func NewSocketClient(socketPath string) *SocketClient {
	return &SocketClient{
		socketPath: socketPath,
		timeout:    5 * time.Second,
	}
}

// SendCommand sends a command and returns the response
func (c *SocketClient) SendCommand(cmd string) (*protocol.Response, error) {
	// Connect to Unix socket
	conn, err := net.DialTimeout("unix", c.socketPath, c.timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to socket: %w", err)
	}
	defer conn.Close()

	// Set read/write timeout
	conn.SetDeadline(time.Now().Add(c.timeout))

	// Send command
	_, err = conn.Write([]byte(cmd + "\n"))
	if err != nil {
		return nil, fmt.Errorf("send error: %w", err)
	}

	// Read response
	scanner := bufio.NewScanner(conn)
	if !scanner.Scan() {
		return nil, fmt.Errorf("no response received")
	}

	responseText := scanner.Text()
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read error: %w", err)
	}

	// Parse JSON response
	var response protocol.Response
	if err := json.Unmarshal([]byte(responseText), &response); err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	return &response, nil
}

// GetStatus gets the current daemon status
func (c *SocketClient) GetStatus() (*protocol.Status, error) {
	resp, err := c.SendCommand("STATUS")
	if err != nil {
		return nil, err
	}

	if !resp.Success {
		return nil, fmt.Errorf("status error: %s", resp.Error)
	}

	// Extract status from response
	statusData, ok := resp.Data["status"]
	if !ok {
		return nil, fmt.Errorf("status not found in response")
	}

	// Convert to JSON and back to parse properly
	statusJSON, _ := json.Marshal(statusData)
	var status protocol.Status
	if err := json.Unmarshal(statusJSON, &status); err != nil {
		return nil, fmt.Errorf("failed to parse status: %w", err)
	}

	return &status, nil
}

// GetMessages gets recent messages
func (c *SocketClient) GetMessages(limit int) ([]protocol.Message, error) {
	cmd := "MESSAGES"
	if limit > 0 {
		cmd = fmt.Sprintf("MESSAGES:%d", limit)
	}

	resp, err := c.SendCommand(cmd)
	if err != nil {
		return nil, err
	}

	if !resp.Success {
		return nil, fmt.Errorf("messages error: %s", resp.Error)
	}

	// Extract messages from response
	messagesData, ok := resp.Data["messages"]
	if !ok {
		return []protocol.Message{}, nil
	}

	// Convert to JSON and back to parse properly
	messagesJSON, _ := json.Marshal(messagesData)
	var messages []protocol.Message
	if err := json.Unmarshal(messagesJSON, &messages); err != nil {
		return nil, fmt.Errorf("failed to parse messages: %w", err)
	}

	return messages, nil
}

// SendMessage sends a message
func (c *SocketClient) SendMessage(to, messageText string) (*protocol.Message, error) {
	cmd := fmt.Sprintf("SEND:%s %s", to, messageText)
	if to == "" {
		cmd = fmt.Sprintf("SEND:%s", messageText)
	}

	resp, err := c.SendCommand(cmd)
	if err != nil {
		return nil, err
	}

	if !resp.Success {
		return nil, fmt.Errorf("send error: %s", resp.Error)
	}

	// Extract message from response
	messageData, ok := resp.Data["message"]
	if !ok {
		return nil, fmt.Errorf("message not found in response")
	}

	// Convert to JSON and back to parse properly
	messageJSON, _ := json.Marshal(messageData)
	var msg protocol.Message
	if err := json.Unmarshal(messageJSON, &msg); err != nil {
		return nil, fmt.Errorf("failed to parse message: %w", err)
	}

	return &msg, nil
}

// GetRadioStatus gets radio status
func (c *SocketClient) GetRadioStatus() (map[string]interface{}, error) {
	resp, err := c.SendCommand("RADIO")
	if err != nil {
		return nil, err
	}

	if !resp.Success {
		return nil, fmt.Errorf("radio error: %s", resp.Error)
	}

	return resp.Data, nil
}

// SetFrequency sets the radio frequency
func (c *SocketClient) SetFrequency(frequency int) error {
	cmd := fmt.Sprintf("FREQUENCY:%d", frequency)

	resp, err := c.SendCommand(cmd)
	if err != nil {
		return err
	}

	if !resp.Success {
		return fmt.Errorf("frequency error: %s", resp.Error)
	}

	return nil
}

// Ping tests the connection
func (c *SocketClient) Ping() error {
	resp, err := c.SendCommand("PING")
	if err != nil {
		return err
	}

	if !resp.Success {
		return fmt.Errorf("ping error: %s", resp.Error)
	}

	return nil
}

// IsConnected tests if the daemon is reachable
func (c *SocketClient) IsConnected() bool {
	return c.Ping() == nil
}

// AbortTransmission aborts any ongoing transmission and turns off PTT
func (c *SocketClient) AbortTransmission() error {
	resp, err := c.SendCommand("ABORT")
	if err != nil {
		return err
	}

	if !resp.Success {
		return fmt.Errorf("abort error: %s", resp.Error)
	}

	return nil
}
