package protocol

import (
	"encoding/json"
	"strings"
	"time"
)

// Command represents a command sent to the core engine
type Command struct {
	Type string                 `json:"type"`
	Args map[string]interface{} `json:"args,omitempty"`
}

// Response represents a response from the core engine
type Response struct {
	Success bool                   `json:"success"`
	Data    map[string]interface{} `json:"data,omitempty"`
	Error   string                 `json:"error,omitempty"`
}

// Message represents a JS8 message
type Message struct {
	ID        int       `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	From      string    `json:"from"`
	To        string    `json:"to"`
	Message   string    `json:"message"`
	SNR       float32   `json:"snr"`
	Frequency int       `json:"frequency"`
	Mode      string    `json:"mode"`
}

// Status represents the current daemon status
type Status struct {
	Callsign   string    `json:"callsign"`
	Grid       string    `json:"grid"`
	Frequency  int       `json:"frequency"`
	Mode       string    `json:"mode"`
	PTT        bool      `json:"ptt"`
	Connected  bool      `json:"connected"`
	Uptime     string    `json:"uptime"`
	StartTime  time.Time `json:"start_time"`
	Version    string    `json:"version"`
}

// ParseCommand parses a text command into a Command struct
func ParseCommand(text string) (*Command, error) {
	text = strings.TrimSpace(text)
	parts := strings.SplitN(text, ":", 2)

	cmd := &Command{
		Type: strings.ToUpper(parts[0]),
		Args: make(map[string]interface{}),
	}

	if len(parts) > 1 {
		args := parts[1]

		switch cmd.Type {
		case "SEND":
			// SEND:N0CALL Hello world
			sendParts := strings.SplitN(args, " ", 2)
			if len(sendParts) >= 2 {
				cmd.Args["to"] = sendParts[0]
				cmd.Args["message"] = sendParts[1]
			} else {
				cmd.Args["to"] = ""
				cmd.Args["message"] = args
			}

		case "MESSAGES":
			// MESSAGES:10 or MESSAGES:since:123
			if strings.Contains(args, "since:") {
				sinceParts := strings.Split(args, "since:")
				if len(sinceParts) > 1 {
					cmd.Args["since"] = sinceParts[1]
				}
			} else {
				cmd.Args["limit"] = args
			}

		case "FREQUENCY":
			// FREQUENCY:14078000
			cmd.Args["frequency"] = args

		case "CONFIG":
			// CONFIG:set:key:value or CONFIG:get:key
			configParts := strings.SplitN(args, ":", 3)
			if len(configParts) >= 1 {
				cmd.Args["action"] = configParts[0]
			}
			if len(configParts) >= 2 {
				cmd.Args["key"] = configParts[1]
			}
			if len(configParts) >= 3 {
				cmd.Args["value"] = configParts[2]
			}
		}
	}

	return cmd, nil
}

// FormatResponse converts a Response to JSON string
func (r *Response) String() string {
	data, _ := json.Marshal(r)
	return string(data)
}

// NewSuccessResponse creates a successful response
func NewSuccessResponse(data map[string]interface{}) *Response {
	return &Response{
		Success: true,
		Data:    data,
	}
}

// NewErrorResponse creates an error response
func NewErrorResponse(err string) *Response {
	return &Response{
		Success: false,
		Error:   err,
	}
}

// Protocol commands
const (
	CmdStatus    = "STATUS"
	CmdMessages  = "MESSAGES"
	CmdSend      = "SEND"
	CmdFrequency = "FREQUENCY"
	CmdConfig    = "CONFIG"
	CmdQuit      = "QUIT"
	CmdPing      = "PING"
	CmdRadio     = "RADIO"
	CmdAudio     = "AUDIO"
)