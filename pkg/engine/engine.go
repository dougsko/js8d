package engine

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"sync"
	"time"

	"github.com/js8call/js8d/pkg/config"
	"github.com/js8call/js8d/pkg/protocol"
)

// CoreEngine represents the main JS8 processing engine
type CoreEngine struct {
	config     *config.Config
	socketPath string
	listener   net.Listener
	running    bool
	mutex      sync.RWMutex
	startTime  time.Time

	// Message storage (mock for now)
	messages []protocol.Message
	msgMutex sync.RWMutex

	// Radio state (mock for now)
	frequency int
	ptt       bool
	connected bool

	// Channels for message processing
	rxMessages chan protocol.Message
	txMessages chan protocol.Message
}

// NewCoreEngine creates a new core engine
func NewCoreEngine(cfg *config.Config, socketPath string) *CoreEngine {
	return &CoreEngine{
		config:     cfg,
		socketPath: socketPath,
		startTime:  time.Now(),
		frequency:  14078000, // Default JS8 frequency
		connected:  true,     // Mock - assume connected
		rxMessages: make(chan protocol.Message, 100),
		txMessages: make(chan protocol.Message, 100),
		messages:   make([]protocol.Message, 0),
	}
}

// Start starts the core engine and Unix socket server
func (e *CoreEngine) Start() error {
	e.mutex.Lock()
	e.running = true
	e.mutex.Unlock()

	// Remove existing socket file
	os.Remove(e.socketPath)

	// Create Unix domain socket
	listener, err := net.Listen("unix", e.socketPath)
	if err != nil {
		return fmt.Errorf("failed to create Unix socket: %w", err)
	}
	e.listener = listener

	// Set socket permissions (readable/writable by owner and group)
	if err := os.Chmod(e.socketPath, 0660); err != nil {
		log.Printf("Warning: failed to set socket permissions: %v", err)
	}

	log.Printf("Core engine listening on %s", e.socketPath)

	// Start message processor
	go e.messageProcessor()

	// Accept connections
	go e.acceptConnections()

	return nil
}

// Stop stops the core engine
func (e *CoreEngine) Stop() error {
	e.mutex.Lock()
	e.running = false
	e.mutex.Unlock()

	if e.listener != nil {
		e.listener.Close()
	}

	// Clean up socket file
	os.Remove(e.socketPath)

	return nil
}

// acceptConnections accepts and handles socket connections
func (e *CoreEngine) acceptConnections() {
	for e.isRunning() {
		conn, err := e.listener.Accept()
		if err != nil {
			if e.isRunning() {
				log.Printf("Socket accept error: %v", err)
			}
			continue
		}

		go e.handleConnection(conn)
	}
}

// handleConnection handles a single socket connection
func (e *CoreEngine) handleConnection(conn net.Conn) {
	defer conn.Close()

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		// Parse command
		cmd, err := protocol.ParseCommand(line)
		if err != nil {
			response := protocol.NewErrorResponse(fmt.Sprintf("parse error: %v", err))
			conn.Write([]byte(response.String() + "\n"))
			continue
		}

		// Handle command
		response := e.handleCommand(cmd)
		conn.Write([]byte(response.String() + "\n"))

		// Close connection after QUIT command
		if cmd.Type == protocol.CmdQuit {
			break
		}
	}
}

// handleCommand processes a single command
func (e *CoreEngine) handleCommand(cmd *protocol.Command) *protocol.Response {
	switch cmd.Type {
	case protocol.CmdStatus:
		return e.handleStatus()

	case protocol.CmdMessages:
		return e.handleMessages(cmd)

	case protocol.CmdSend:
		return e.handleSend(cmd)

	case protocol.CmdFrequency:
		return e.handleFrequency(cmd)

	case protocol.CmdRadio:
		return e.handleRadio()

	case protocol.CmdPing:
		return protocol.NewSuccessResponse(map[string]interface{}{
			"pong": time.Now().Unix(),
		})

	case protocol.CmdQuit:
		return protocol.NewSuccessResponse(map[string]interface{}{
			"message": "goodbye",
		})

	default:
		return protocol.NewErrorResponse(fmt.Sprintf("unknown command: %s", cmd.Type))
	}
}

// handleStatus returns current daemon status
func (e *CoreEngine) handleStatus() *protocol.Response {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	status := protocol.Status{
		Callsign:  e.config.Station.Callsign,
		Grid:      e.config.Station.Grid,
		Frequency: e.frequency,
		Mode:      "JS8",
		PTT:       e.ptt,
		Connected: e.connected,
		Uptime:    time.Since(e.startTime).String(),
		StartTime: e.startTime,
		Version:   "0.1.0-dev",
	}

	return protocol.NewSuccessResponse(map[string]interface{}{
		"status": status,
	})
}

// handleMessages returns message history
func (e *CoreEngine) handleMessages(cmd *protocol.Command) *protocol.Response {
	e.msgMutex.RLock()
	defer e.msgMutex.RUnlock()

	// For now, return mock messages
	messages := []protocol.Message{
		{
			ID:        1,
			Timestamp: time.Now().Add(-5 * time.Minute),
			From:      "N0ABC",
			To:        e.config.Station.Callsign,
			Message:   "Hello from the mountains!",
			SNR:       -12.5,
			Frequency: 14078000,
			Mode:      "JS8",
		},
		{
			ID:        2,
			Timestamp: time.Now().Add(-2 * time.Minute),
			From:      "N0DEF",
			To:        "",
			Message:   "CQ CQ DE N0DEF N0DEF K",
			SNR:       -8.2,
			Frequency: 14078500,
			Mode:      "JS8",
		},
	}

	return protocol.NewSuccessResponse(map[string]interface{}{
		"messages": messages,
		"count":    len(messages),
	})
}

// handleSend queues a message for transmission
func (e *CoreEngine) handleSend(cmd *protocol.Command) *protocol.Response {
	to, _ := cmd.Args["to"].(string)
	message, _ := cmd.Args["message"].(string)

	if message == "" {
		return protocol.NewErrorResponse("message cannot be empty")
	}

	msg := protocol.Message{
		ID:        int(time.Now().Unix()),
		Timestamp: time.Now(),
		From:      e.config.Station.Callsign,
		To:        to,
		Message:   message,
		Mode:      "JS8",
	}

	// Queue for transmission
	select {
	case e.txMessages <- msg:
		log.Printf("TX queued: %s -> %s: %s", msg.From, msg.To, msg.Message)
		return protocol.NewSuccessResponse(map[string]interface{}{
			"status":  "queued",
			"message": msg,
		})
	default:
		return protocol.NewErrorResponse("transmit queue full")
	}
}

// handleFrequency sets the radio frequency
func (e *CoreEngine) handleFrequency(cmd *protocol.Command) *protocol.Response {
	// TODO: Implement actual radio control
	freqStr, _ := cmd.Args["frequency"].(string)

	// For now, just acknowledge
	return protocol.NewSuccessResponse(map[string]interface{}{
		"status":    "ok",
		"frequency": freqStr,
	})
}

// handleRadio returns radio status
func (e *CoreEngine) handleRadio() *protocol.Response {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	return protocol.NewSuccessResponse(map[string]interface{}{
		"frequency": e.frequency,
		"mode":      "USB",
		"ptt":       e.ptt,
		"connected": e.connected,
		"model":     e.config.Radio.Model,
		"device":    e.config.Radio.Device,
	})
}

// messageProcessor handles incoming and outgoing messages
func (e *CoreEngine) messageProcessor() {
	for e.isRunning() {
		select {
		case msg := <-e.rxMessages:
			log.Printf("RX: %s -> %s: %s (SNR: %.1fdB)", msg.From, msg.To, msg.Message, msg.SNR)

			// Store message
			e.msgMutex.Lock()
			e.messages = append(e.messages, msg)
			e.msgMutex.Unlock()

		case msg := <-e.txMessages:
			log.Printf("TX: %s -> %s: %s", msg.From, msg.To, msg.Message)

			// TODO: Actual transmission via DSP/audio
			// For now, just simulate transmission delay
			time.Sleep(100 * time.Millisecond)

		case <-time.After(1 * time.Second):
			// Periodic processing (keep-alive, etc.)
			continue
		}
	}
}

// isRunning checks if the engine is running
func (e *CoreEngine) isRunning() bool {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	return e.running
}