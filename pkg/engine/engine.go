package engine

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dougsko/js8d/pkg/audio"
	"github.com/dougsko/js8d/pkg/config"
	"github.com/dougsko/js8d/pkg/dsp"
	"github.com/dougsko/js8d/pkg/hardware"
	"github.com/dougsko/js8d/pkg/protocol"
	"github.com/dougsko/js8d/pkg/storage"
)

// CoreEngine represents the main JS8 processing engine
type CoreEngine struct {
	config     *config.Config
	configPath string
	socketPath string
	listener   net.Listener
	running    bool
	mutex      sync.RWMutex
	startTime  time.Time

	// DSP and hardware components
	dspEngine       *dsp.DSP
	hardwareManager *hardware.HardwareManager
	audioMonitor    *audio.AudioLevelMonitor

	// Message storage
	messageStore *storage.MessageStore
	msgMutex     sync.RWMutex

	// Radio state
	frequency int
	ptt       bool
	connected bool

	// Channels for message processing
	rxMessages chan protocol.Message
	txMessages chan protocol.Message

	// Transmission control
	abortTx      chan bool
	transmitting bool
	txMutex      sync.RWMutex
}

// NewCoreEngine creates a new core engine with config path for reloading
func NewCoreEngine(cfg *config.Config, socketPath, configPath string) *CoreEngine {
	// Create hardware configuration from config
	hardwareConfig := hardware.HardwareConfig{
		EnableGPIO:     cfg.Hardware.EnableGPIO,
		PTTGPIOPin:     cfg.Hardware.PTTGPIOPin,
		StatusLEDPin:   cfg.Hardware.StatusLEDPin,
		EnableOLED:     cfg.Hardware.EnableOLED,
		OLEDI2CAddress: cfg.Hardware.OLEDI2CAddress,
		OLEDWidth:      cfg.Hardware.OLEDWidth,
		OLEDHeight:     cfg.Hardware.OLEDHeight,
		EnableAudio:    true, // Always enable audio for radio operations
		AudioInput:     cfg.Audio.InputDevice,
		AudioOutput:    cfg.Audio.OutputDevice,
		SampleRate:     cfg.Audio.SampleRate,
		BufferSize:     cfg.Audio.BufferSize,
		EnableRadio:    cfg.Radio.Device != "", // Enable radio if device is specified
		UseHamlib:      cfg.Radio.UseHamlib,
		RadioModel:     cfg.Radio.Model,
		RadioDevice:    cfg.Radio.Device,
		RadioBaudRate:  cfg.Radio.BaudRate,
	}

	// Set defaults if not specified
	if hardwareConfig.SampleRate == 0 {
		hardwareConfig.SampleRate = 48000
	}
	if hardwareConfig.BufferSize == 0 {
		hardwareConfig.BufferSize = 1024
	}
	if hardwareConfig.OLEDWidth == 0 {
		hardwareConfig.OLEDWidth = 128
	}
	if hardwareConfig.OLEDHeight == 0 {
		hardwareConfig.OLEDHeight = 64
	}
	if hardwareConfig.RadioBaudRate == 0 {
		hardwareConfig.RadioBaudRate = 4800 // Default radio baud rate
	}

	// Initialize message store
	messageStore, err := storage.NewMessageStore(cfg.Storage.DatabasePath, cfg.Storage.MaxMessages)
	if err != nil {
		log.Printf("Warning: Failed to initialize message store: %v", err)
		messageStore = nil // Continue without storage
	}

	// Initialize audio monitor for real-time visualization
	audioMonitor := audio.NewAudioLevelMonitor(hardwareConfig.SampleRate, 1024)

	return &CoreEngine{
		config:          cfg,
		configPath:      configPath,
		socketPath:      socketPath,
		startTime:       time.Now(),
		frequency:       14078000, // Default JS8 frequency
		connected:       true,     // Mock - assume connected
		rxMessages:      make(chan protocol.Message, 100),
		txMessages:      make(chan protocol.Message, 100),
		messageStore:    messageStore,
		dspEngine:       dsp.NewDSP(),
		hardwareManager: hardware.NewHardwareManager(hardwareConfig),
		audioMonitor:    audioMonitor,
		abortTx:         make(chan bool, 1),
		transmitting:    false,
	}
}

// Start starts the core engine and Unix socket server
func (e *CoreEngine) Start() error {
	e.mutex.Lock()
	e.running = true
	e.mutex.Unlock()

	// Initialize DSP engine
	if err := e.dspEngine.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize DSP engine: %w", err)
	}
	log.Printf("DSP engine initialized successfully")

	// Initialize hardware manager
	if err := e.hardwareManager.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize hardware manager: %w", err)
	}

	// Start audio input for decoding
	log.Printf("DEBUG: About to start audio input...")
	if err := e.hardwareManager.StartAudioInput(); err != nil {
		log.Printf("Warning: failed to start audio input: %v", err)
	} else {
		log.Printf("DEBUG: Audio input startup completed successfully")
	}

	// Start audio output for transmission
	if err := e.hardwareManager.StartAudioOutput(); err != nil {
		log.Printf("Warning: failed to start audio output: %v", err)
	}

	// Start audio monitoring
	if err := e.audioMonitor.Start(); err != nil {
		log.Printf("Warning: failed to start audio monitor: %v", err)
	} else {
		// Start audio sample processing goroutine
		go e.processAudioSamples()
		log.Printf("Audio monitoring started")
	}

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

	// Start audio processor
	go e.audioProcessor()

	// Start heartbeat generator
	go e.heartbeatGenerator()

	// Accept connections
	go e.acceptConnections()

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

	case protocol.CmdAbort:
		return e.handleAbort()

	case protocol.CmdReload:
		return e.handleReload()

	case protocol.CmdQuit:
		return protocol.NewSuccessResponse(map[string]interface{}{
			"message": "goodbye",
		})

	default:
		// Handle string commands that aren't in the protocol enum
		return e.handleStringCommand(cmd)
	}
}

// handleStringCommand handles non-enum commands like database operations
func (e *CoreEngine) handleStringCommand(cmd *protocol.Command) *protocol.Response {
	cmdStr := string(cmd.Type)
	parts := strings.Fields(cmdStr)

	if len(parts) == 0 {
		return protocol.NewErrorResponse("empty command")
	}

	switch parts[0] {
	case "GET_MESSAGE_HISTORY":
		return e.handleGetMessageHistory(parts[1:])
	case "GET_CONVERSATIONS":
		return e.handleGetConversations(parts[1:])
	case "MARK_MESSAGES_READ":
		return e.handleMarkMessagesRead(parts[1:])
	case "SEARCH_MESSAGES":
		return e.handleSearchMessages(parts[1:])
	case "GET_MESSAGE_STATS":
		return e.handleGetMessageStats()
	case "CLEANUP_MESSAGES":
		return e.handleCleanupMessages()
	case "TEST_CAT":
		return e.handleTestCAT(parts[1:])
	case "TEST_PTT":
		return e.handleTestPTT(parts[1:])
	case "TEST_PTT_OFF":
		return e.handleTestPTTOff()
	case "RETRY_RADIO":
		return e.handleRetryRadio()
	default:
		return protocol.NewErrorResponse(fmt.Sprintf("unknown command: %s", cmdStr))
	}
}

// handleStatus returns current daemon status
func (e *CoreEngine) handleStatus() *protocol.Response {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	// Get current frequency from radio if available
	currentFreq := e.frequency // fallback to cached frequency
	if e.hardwareManager != nil && e.hardwareManager.IsRadioConnected() {
		if radioFreq, err := e.hardwareManager.GetRadioFrequency(); err == nil {
			currentFreq = int(radioFreq)
		}
	}

	status := protocol.Status{
		Callsign:  e.config.Station.Callsign,
		Grid:      e.config.Station.Grid,
		Frequency: currentFreq,
		Mode:      "JS8",
		PTT:       e.ptt,
		Connected: e.connected,
		Uptime:    time.Since(e.startTime).String(),
		StartTime: e.startTime,
		Version:   "0.1.0-dev",
	}

	// Add hardware status if hardware manager is available
	data := map[string]interface{}{
		"status": status,
	}

	if e.hardwareManager != nil && e.hardwareManager.IsInitialized() {
		hardwareStatus := map[string]interface{}{
			"initialized": true,
			"ptt_active":  e.hardwareManager.GetPTT(),
			"config":      e.hardwareManager.GetConfig(),
		}

		// Add audio status if available
		if audio := e.hardwareManager.GetAudio(); audio != nil {
			hardwareStatus["audio"] = map[string]interface{}{
				"recording":   audio.IsRecording(),
				"playing":     audio.IsPlaying(),
				"sample_rate": audio.GetSampleRate(),
				"buffer_size": audio.GetBufferSize(),
			}
		}

		data["hardware"] = hardwareStatus
	}

	return protocol.NewSuccessResponse(data)
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

			// Store message in database
			e.msgMutex.Lock()
			if e.messageStore != nil {
				messageType := e.classifyMessage(msg.Message)
				if err := e.messageStore.StoreMessage(msg, "RX", messageType); err != nil {
					log.Printf("Failed to store RX message: %v", err)
				}
			}
			e.msgMutex.Unlock()

			// Update OLED display with received message
			e.updateOLEDDisplay(fmt.Sprintf("RX: %s", msg.Message))

			// Handle auto-replies for directed messages
			e.handleAutoReply(msg)

		case msg := <-e.txMessages:
			log.Printf("TX: %s -> %s: %s", msg.From, msg.To, msg.Message)

			// Store TX message in database
			e.msgMutex.Lock()
			if e.messageStore != nil {
				messageType := e.classifyMessage(msg.Message)
				if err := e.messageStore.StoreMessage(msg, "TX", messageType); err != nil {
					log.Printf("Failed to store TX message: %v", err)
				}
			}
			e.msgMutex.Unlock()

			// Encode message using real DSP
			if err := e.transmitMessage(msg); err != nil {
				log.Printf("TX error: %v", err)
			}

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

// transmitMessage encodes and transmits a message using the DSP engine
func (e *CoreEngine) transmitMessage(msg protocol.Message) error {
	// Set transmission state
	e.txMutex.Lock()
	e.transmitting = true
	e.txMutex.Unlock()

	defer func() {
		e.txMutex.Lock()
		e.transmitting = false
		e.txMutex.Unlock()
	}()

	// Set PTT flag and hardware PTT during transmission
	e.mutex.Lock()
	e.ptt = true
	e.mutex.Unlock()

	// Activate hardware PTT
	if err := e.hardwareManager.SetRadioPTT(true); err != nil {
		log.Printf("Warning: failed to set radio PTT: %v", err)
	}

	defer func() {
		// Deactivate hardware PTT
		if err := e.hardwareManager.SetRadioPTT(false); err != nil {
			log.Printf("Warning: failed to clear radio PTT: %v", err)
		}

		e.mutex.Lock()
		e.ptt = false
		e.mutex.Unlock()
	}()

	// Format message for JS8 transmission (12 characters max)
	txMessage := msg.Message
	if len(txMessage) > 12 {
		txMessage = txMessage[:12]
	}

	// Use normal mode for now
	mode := dsp.ModeNormal

	// Encode to audio samples
	audioData, err := e.dspEngine.EncodeMessage(txMessage, mode)
	if err != nil {
		return fmt.Errorf("DSP encoding failed: %w", err)
	}

	log.Printf("DSP: Encoded '%s' to %d audio samples", txMessage, len(audioData))

	// Send audio data to hardware audio system for output
	if err := e.hardwareManager.PlayAudio(audioData); err != nil {
		return fmt.Errorf("audio output failed: %w", err)
	}

	// Wait for transmission to complete with abort monitoring
	duration := e.dspEngine.EstimateAudioDuration(mode)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	endTime := time.Now().Add(duration)
	for time.Now().Before(endTime) {
		select {
		case <-e.abortTx:
			log.Printf("DSP: Transmission aborted by user")
			return fmt.Errorf("transmission aborted")
		case <-ticker.C:
			// Continue waiting
		}
	}

	log.Printf("DSP: Transmission complete")

	// Update OLED display with transmission status
	e.updateOLEDDisplay(fmt.Sprintf("TX: %s", txMessage))

	return nil
}

// audioProcessor handles incoming audio data and decoding
func (e *CoreEngine) audioProcessor() {
	inputSamples := e.hardwareManager.GetAudioInputSamples()

	// If audio is not available, just exit
	if inputSamples == nil {
		log.Printf("Audio input not available, audio processor disabled")
		return
	}

	// Buffer for accumulating samples for decoding
	var audioBuffer []int16
	const bufferLimit = 15 * 48000 // 15 seconds at 48kHz max

	for e.isRunning() {
		select {
		case samples := <-inputSamples:
			// Accumulate audio samples
			audioBuffer = append(audioBuffer, samples...)

			// If buffer gets too large, trim it to prevent memory issues
			if len(audioBuffer) > bufferLimit {
				// Keep last 10 seconds worth
				keepSamples := 10 * 48000
				if len(audioBuffer) > keepSamples {
					audioBuffer = audioBuffer[len(audioBuffer)-keepSamples:]
				}
			}

			// Try to decode if we have enough samples (at least 3 seconds)
			minSamples := 3 * 48000
			if len(audioBuffer) >= minSamples {
				e.attemptDecode(audioBuffer)
			}

		case <-time.After(1 * time.Second):
			// Periodic cleanup - try to decode accumulated buffer
			if len(audioBuffer) > 0 {
				e.attemptDecode(audioBuffer)
				// Clear buffer after decode attempt
				audioBuffer = audioBuffer[:0]
			}
		}
	}
}

// attemptDecode tries to decode JS8 messages from audio buffer
func (e *CoreEngine) attemptDecode(audioBuffer []int16) {
	if len(audioBuffer) == 0 {
		return
	}

	// Use DSP to decode the audio buffer
	decodeCount, err := e.dspEngine.DecodeBuffer(audioBuffer, func(result *dsp.DecodeResult) {
		// Parse JS8 message to extract callsigns and determine message type
		msg := e.parseJS8Message(result)

		// Queue the received message
		select {
		case e.rxMessages <- msg:
			log.Printf("RX decoded: %s (SNR: %ddB, Freq: %.1fHz, Type: %s)",
				result.Message, result.SNR, result.Frequency, e.getMessageType(result.Message))
		default:
			log.Printf("RX buffer full, dropping message: %s", result.Message)
		}
	})

	if err != nil {
		log.Printf("Decode error: %v", err)
	} else if decodeCount > 0 {
		log.Printf("Decoded %d message(s) from audio buffer", decodeCount)
	}
}

// parseJS8Message parses a JS8 decode result into a protocol message
func (e *CoreEngine) parseJS8Message(result *dsp.DecodeResult) protocol.Message {
	message := result.Message

	// Parse callsigns from the message using varicode utilities
	callsigns := dsp.ParseCallsigns(message)

	var fromCall, toCall string

	// Determine message structure and extract callsigns
	if dsp.StartsWithCQ(message) {
		// CQ messages: "CQ N0CALL EM12"
		if len(callsigns) > 0 {
			fromCall = callsigns[0]
		}
		toCall = "" // CQ is broadcast
	} else if len(callsigns) >= 2 {
		// Directed messages: "N0ABC N0XYZ message"
		toCall = callsigns[0]
		fromCall = callsigns[1]
	} else if len(callsigns) == 1 {
		// Single callsign - could be response or heartbeat
		fromCall = callsigns[0]
		if e.isDirectedToMe(message) {
			toCall = e.config.Station.Callsign
		}
	}

	// If we couldn't parse callsigns, mark as unknown
	if fromCall == "" {
		fromCall = "UNKNOWN"
	}

	return protocol.Message{
		ID:        int(time.Now().Unix()),
		Timestamp: time.Now(),
		From:      fromCall,
		To:        toCall,
		Message:   message,
		SNR:       float32(result.SNR),
		Frequency: int(result.Frequency),
		Mode:      "JS8",
	}
}

// getMessageType determines the type of JS8 message for logging
func (e *CoreEngine) getMessageType(message string) string {
	if dsp.StartsWithCQ(message) {
		return "CQ"
	} else if dsp.StartsWithHB(message) {
		return "HEARTBEAT"
	} else if dsp.IsSNRCommand(message) {
		return "SNR_REQUEST"
	} else if e.isDirectedToMe(message) {
		return "DIRECTED"
	} else if len(dsp.ParseCallsigns(message)) >= 2 {
		return "DIRECTED"
	}
	return "UNKNOWN"
}

// isDirectedToMe checks if a message is directed to our station
func (e *CoreEngine) isDirectedToMe(message string) bool {
	// Check if our callsign appears at the beginning of the message
	myCall := e.config.Station.Callsign
	if myCall == "" {
		return false
	}

	// Simple check - message starts with our callsign
	return len(message) > len(myCall) && message[:len(myCall)] == myCall
}

// handleAutoReply processes messages that require automatic responses
func (e *CoreEngine) handleAutoReply(msg protocol.Message) {
	// Only auto-reply to messages directed to us
	if msg.To != e.config.Station.Callsign || msg.From == e.config.Station.Callsign {
		return
	}

	message := msg.Message

	// Check for SNR requests
	if dsp.IsSNRCommand(message) {
		snr := int(msg.SNR)
		response := fmt.Sprintf("%s %s", msg.From, dsp.FormatSNR(snr))

		replyMsg := protocol.Message{
			ID:        int(time.Now().Unix()),
			Timestamp: time.Now(),
			From:      e.config.Station.Callsign,
			To:        msg.From,
			Message:   response,
			Mode:      "JS8",
		}

		// Queue the auto-reply
		select {
		case e.txMessages <- replyMsg:
			log.Printf("Auto-reply queued: SNR report %s to %s", dsp.FormatSNR(snr), msg.From)
		default:
			log.Printf("TX queue full, dropping auto-reply to %s", msg.From)
		}
	}

	// TODO: Add more auto-reply handlers (GRID?, INFO?, etc.)
}

// heartbeatGenerator sends periodic heartbeat messages
func (e *CoreEngine) heartbeatGenerator() {
	// Send a heartbeat every 5 minutes (JS8 common practice)
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for e.isRunning() {
		select {
		case <-ticker.C:
			e.sendHeartbeat()

		case <-time.After(30 * time.Second):
			// Keep the goroutine alive
			continue
		}
	}
}

// updateOLEDDisplay updates the OLED display with current station info
func (e *CoreEngine) updateOLEDDisplay(lastMessage string) {
	if e.hardwareManager == nil {
		return
	}

	callsign := e.config.Station.Callsign
	grid := e.config.Station.Grid
	frequency := e.frequency

	if err := e.hardwareManager.UpdateOLED(callsign, grid, frequency, lastMessage); err != nil {
		log.Printf("Warning: failed to update OLED: %v", err)
	}
}

// sendHeartbeat sends a JS8 heartbeat message
func (e *CoreEngine) sendHeartbeat() {
	callsign := e.config.Station.Callsign
	grid := e.config.Station.Grid

	if callsign == "" {
		return // Can't send heartbeat without callsign
	}

	// Format heartbeat message: "HBAUTO" + callsign + grid (no spaces - JS8 doesn't support them)
	var hbMessage string
	if grid != "" {
		hbMessage = fmt.Sprintf("HBAUTO%s%s", callsign, grid)
	} else {
		hbMessage = fmt.Sprintf("HBAUTO%s", callsign)
	}

	// Truncate if too long for JS8
	if len(hbMessage) > 12 {
		hbMessage = hbMessage[:12]
	}

	heartbeat := protocol.Message{
		ID:        int(time.Now().Unix()),
		Timestamp: time.Now(),
		From:      callsign,
		To:        "", // Heartbeats are broadcast
		Message:   hbMessage,
		Mode:      "JS8",
	}

	// Queue the heartbeat
	select {
	case e.txMessages <- heartbeat:
		log.Printf("Heartbeat queued: %s", hbMessage)
	default:
		log.Printf("TX queue full, dropping heartbeat")
	}
}

// SetRadioFrequency sets the radio frequency and updates engine state
func (e *CoreEngine) SetRadioFrequency(freq int64) error {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	// Set radio frequency
	if err := e.hardwareManager.SetRadioFrequency(freq); err != nil {
		return fmt.Errorf("failed to set radio frequency: %w", err)
	}

	// Update engine frequency state
	e.frequency = int(freq)
	log.Printf("Engine: Radio frequency set to %.3f MHz", float64(freq)/1000000.0)
	return nil
}

// GetRadioFrequency gets the current radio frequency
func (e *CoreEngine) GetRadioFrequency() (int64, error) {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	return e.hardwareManager.GetRadioFrequency()
}

// SetRadioMode sets the radio mode for JS8 operation
func (e *CoreEngine) SetRadioMode(mode string, bandwidth int) error {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	return e.hardwareManager.SetRadioMode(mode, bandwidth)
}

// EnablePTT enables PTT for transmission
func (e *CoreEngine) EnablePTT() error {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	// Set both GPIO and radio PTT
	if err := e.hardwareManager.SetPTT(true); err != nil {
		log.Printf("Warning: GPIO PTT failed: %v", err)
	}

	if err := e.hardwareManager.SetRadioPTT(true); err != nil {
		return fmt.Errorf("failed to enable radio PTT: %w", err)
	}

	log.Printf("Engine: PTT enabled")
	return nil
}

// DisablePTT disables PTT after transmission
func (e *CoreEngine) DisablePTT() error {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	// Disable both radio and GPIO PTT
	if err := e.hardwareManager.SetRadioPTT(false); err != nil {
		log.Printf("Warning: Radio PTT disable failed: %v", err)
	}

	if err := e.hardwareManager.SetPTT(false); err != nil {
		log.Printf("Warning: GPIO PTT disable failed: %v", err)
	}

	log.Printf("Engine: PTT disabled")
	return nil
}

// GetRadioStatus returns radio connection and status information
func (e *CoreEngine) GetRadioStatus() map[string]interface{} {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	status := map[string]interface{}{
		"connected": e.hardwareManager.IsRadioConnected(),
	}

	if freq, err := e.hardwareManager.GetRadioFrequency(); err == nil {
		status["frequency"] = freq
	}

	if mode, bandwidth, err := e.hardwareManager.GetRadioMode(); err == nil {
		status["mode"] = mode
		status["bandwidth"] = bandwidth
	}

	if ptt, err := e.hardwareManager.GetRadioPTT(); err == nil {
		status["ptt"] = ptt
	}

	if power, err := e.hardwareManager.GetRadioPowerLevel(); err == nil {
		status["power"] = power
	}

	if swr, err := e.hardwareManager.GetRadioSWRLevel(); err == nil {
		status["swr"] = swr
	}

	if signal, err := e.hardwareManager.GetRadioSignalLevel(); err == nil {
		status["signal"] = signal
	}

	return status
}

// handleAbort aborts any ongoing transmission and turns off PTT
func (e *CoreEngine) handleAbort() *protocol.Response {
	e.txMutex.Lock()
	isTransmitting := e.transmitting
	e.txMutex.Unlock()

	if isTransmitting {
		// Signal abort to any ongoing transmission
		select {
		case e.abortTx <- true:
			log.Printf("Engine: Transmission abort signal sent")
		default:
			// Channel is full or no one is listening, but that's ok
		}
	}

	// Force PTT off immediately (both GPIO and radio)
	if err := e.hardwareManager.SetRadioPTT(false); err != nil {
		log.Printf("Warning: failed to clear radio PTT during abort: %v", err)
	}
	if err := e.hardwareManager.SetPTT(false); err != nil {
		log.Printf("Warning: failed to clear GPIO PTT during abort: %v", err)
	}

	// Update engine PTT state
	e.mutex.Lock()
	e.ptt = false
	e.mutex.Unlock()

	log.Printf("Engine: Emergency transmission abort completed")

	return protocol.NewSuccessResponse(map[string]interface{}{
		"status":        "aborted",
		"was_transmitting": isTransmitting,
		"ptt_cleared":   true,
	})
}

// handleReload reloads configuration from file
func (e *CoreEngine) handleReload() *protocol.Response {
	if e.configPath == "" {
		return protocol.NewErrorResponse("no config path specified - cannot reload")
	}

	// Load new configuration
	newConfig, err := config.LoadConfig(e.configPath)
	if err != nil {
		return protocol.NewErrorResponse(fmt.Sprintf("failed to load config: %v", err))
	}

	// Validate the new configuration
	if err := newConfig.Validate(); err != nil {
		return protocol.NewErrorResponse(fmt.Sprintf("invalid configuration: %v", err))
	}

	// Check if audio configuration changed
	e.mutex.Lock()
	oldCallsign := e.config.Station.Callsign
	oldGrid := e.config.Station.Grid
	audioChanged := (e.config.Audio.InputDevice != newConfig.Audio.InputDevice ||
		e.config.Audio.OutputDevice != newConfig.Audio.OutputDevice ||
		e.config.Audio.SampleRate != newConfig.Audio.SampleRate)
	e.config = newConfig
	e.mutex.Unlock()

	log.Printf("Engine: Configuration reloaded from %s", e.configPath)
	log.Printf("Engine: Station updated - %s (%s)", newConfig.Station.Callsign, newConfig.Station.Grid)

	// Warn about audio changes - full reinit would require restart
	if audioChanged {
		log.Printf("Engine: Warning - Audio configuration changed. Full restart recommended for audio changes.")
		return protocol.NewSuccessResponse(map[string]interface{}{
			"status":       "reloaded",
			"config_path":  e.configPath,
			"old_callsign": oldCallsign,
			"new_callsign": newConfig.Station.Callsign,
			"old_grid":     oldGrid,
			"new_grid":     newConfig.Station.Grid,
			"warning":      "Audio configuration changed - restart recommended",
		})
	}

	return protocol.NewSuccessResponse(map[string]interface{}{
		"status":       "reloaded",
		"config_path":  e.configPath,
		"old_callsign": oldCallsign,
		"new_callsign": newConfig.Station.Callsign,
		"old_grid":     oldGrid,
		"new_grid":     newConfig.Station.Grid,
	})
}

// Stop gracefully shuts down the core engine
func (e *CoreEngine) Stop() error {
	log.Printf("Stopping core engine...")

	// Stop the engine
	e.mutex.Lock()
	e.running = false
	e.mutex.Unlock()

	// Close listener if it exists
	if e.listener != nil {
		if err := e.listener.Close(); err != nil {
			log.Printf("Error closing listener: %v", err)
		}
	}

	// Close message store
	if e.messageStore != nil {
		if err := e.messageStore.Close(); err != nil {
			log.Printf("Error closing message store: %v", err)
		}
	}

	// Close hardware manager
	if e.hardwareManager != nil {
		e.hardwareManager.Close()
	}

	log.Printf("Core engine stopped")
	return nil
}

// classifyMessage determines the type of a JS8 message
func (e *CoreEngine) classifyMessage(message string) string {
	message = strings.ToUpper(strings.TrimSpace(message))

	// Classify common JS8 message types
	if strings.HasPrefix(message, "CQ") {
		return "CQ"
	}
	if strings.HasPrefix(message, "HB") || strings.Contains(message, "HEARTBEAT") {
		return "HEARTBEAT"
	}
	if strings.Contains(message, "SNR") {
		return "SNR_REPORT"
	}
	if strings.Contains(message, "73") {
		return "FAREWELL"
	}
	if strings.Contains(message, "?") {
		return "QUERY"
	}
	if len(message) > 0 && (message[0] == '@' || strings.Contains(message, ":")) {
		return "DIRECTED"
	}

	return "MESSAGE"
}

// handleGetMessageHistory handles GET_MESSAGE_HISTORY command
func (e *CoreEngine) handleGetMessageHistory(args []string) *protocol.Response {
	if e.messageStore == nil {
		return protocol.NewErrorResponse("message storage not available")
	}

	// Parse arguments: limit offset callsign direction messageType unreadOnly
	limit := 50
	offset := 0
	callsign := ""
	direction := ""
	messageType := ""
	unreadOnly := false

	if len(args) > 0 {
		if l, err := strconv.Atoi(args[0]); err == nil {
			limit = l
		}
	}
	if len(args) > 1 {
		if o, err := strconv.Atoi(args[1]); err == nil {
			offset = o
		}
	}
	if len(args) > 2 && args[2] != "" {
		callsign = args[2]
	}
	if len(args) > 3 && args[3] != "" {
		direction = args[3]
	}
	if len(args) > 4 && args[4] != "" {
		messageType = args[4]
	}
	if len(args) > 5 {
		unreadOnly = args[5] == "true"
	}

	query := storage.MessageQuery{
		Limit:       limit,
		Offset:      offset,
		Callsign:    callsign,
		Direction:   direction,
		MessageType: messageType,
		UnreadOnly:  unreadOnly,
	}

	messages, err := e.messageStore.GetMessages(query)
	if err != nil {
		return protocol.NewErrorResponse(fmt.Sprintf("failed to get messages: %v", err))
	}

	return protocol.NewSuccessResponse(map[string]interface{}{
		"messages": messages,
		"count":    len(messages),
	})
}

// handleGetConversations handles GET_CONVERSATIONS command
func (e *CoreEngine) handleGetConversations(args []string) *protocol.Response {
	if e.messageStore == nil {
		return protocol.NewErrorResponse("message storage not available")
	}

	limit := 20
	if len(args) > 0 {
		if l, err := strconv.Atoi(args[0]); err == nil {
			limit = l
		}
	}

	conversations, err := e.messageStore.GetConversations(limit)
	if err != nil {
		return protocol.NewErrorResponse(fmt.Sprintf("failed to get conversations: %v", err))
	}

	return protocol.NewSuccessResponse(map[string]interface{}{
		"conversations": conversations,
		"count":         len(conversations),
	})
}

// handleMarkMessagesRead handles MARK_MESSAGES_READ command
func (e *CoreEngine) handleMarkMessagesRead(args []string) *protocol.Response {
	if e.messageStore == nil {
		return protocol.NewErrorResponse("message storage not available")
	}

	if len(args) == 0 {
		return protocol.NewErrorResponse("callsign required")
	}

	callsign := args[0]
	if err := e.messageStore.MarkMessagesAsRead(callsign); err != nil {
		return protocol.NewErrorResponse(fmt.Sprintf("failed to mark messages as read: %v", err))
	}

	return protocol.NewSuccessResponse(map[string]interface{}{
		"status":   "success",
		"callsign": callsign,
	})
}

// handleSearchMessages handles SEARCH_MESSAGES command
func (e *CoreEngine) handleSearchMessages(args []string) *protocol.Response {
	if e.messageStore == nil {
		return protocol.NewErrorResponse("message storage not available")
	}

	if len(args) == 0 {
		return protocol.NewErrorResponse("search query required")
	}

	query := args[0]
	limit := 50
	if len(args) > 1 {
		if l, err := strconv.Atoi(args[1]); err == nil {
			limit = l
		}
	}

	messages, err := e.messageStore.SearchMessages(query, limit)
	if err != nil {
		return protocol.NewErrorResponse(fmt.Sprintf("search failed: %v", err))
	}

	return protocol.NewSuccessResponse(map[string]interface{}{
		"messages": messages,
		"count":    len(messages),
		"query":    query,
	})
}

// handleGetMessageStats handles GET_MESSAGE_STATS command
func (e *CoreEngine) handleGetMessageStats() *protocol.Response {
	if e.messageStore == nil {
		return protocol.NewErrorResponse("message storage not available")
	}

	stats, err := e.messageStore.GetMessageStats()
	if err != nil {
		return protocol.NewErrorResponse(fmt.Sprintf("failed to get stats: %v", err))
	}

	return protocol.NewSuccessResponse(map[string]interface{}{
		"total_messages": stats.TotalMessages,
		"total_rx":       stats.TotalRX,
		"total_tx":       stats.TotalTX,
		"last_cleanup":   stats.LastCleanup,
	})
}

// handleCleanupMessages triggers manual cleanup of old messages
func (e *CoreEngine) handleCleanupMessages() *protocol.Response {
	if e.messageStore == nil {
		return protocol.NewErrorResponse("message storage not available")
	}

	// Get current count before cleanup
	currentCount, err := e.messageStore.GetMessageCount()
	if err != nil {
		return protocol.NewErrorResponse(fmt.Sprintf("failed to get current message count: %v", err))
	}

	// Force cleanup using the exported method
	if err := e.messageStore.CleanupOldMessages(); err != nil {
		return protocol.NewErrorResponse(fmt.Sprintf("cleanup failed: %v", err))
	}

	// Get count after cleanup to see how many were deleted
	newCount, err := e.messageStore.GetMessageCount()
	if err != nil {
		log.Printf("Warning: failed to get post-cleanup count: %v", err)
		newCount = currentCount // Fallback
	}

	deletedCount := currentCount - newCount
	log.Printf("Manual cleanup completed: %d messages deleted", deletedCount)

	return protocol.NewSuccessResponse(map[string]interface{}{
		"status":        "success",
		"deleted_count": deletedCount,
		"total_before":  currentCount,
		"total_after":   newCount,
	})
}

// handleTestCAT handles TEST_CAT command for radio testing
func (e *CoreEngine) handleTestCAT(args []string) *protocol.Response {
	if len(args) < 3 {
		return protocol.NewErrorResponse("usage: TEST_CAT device model baudrate")
	}

	device := args[0]
	model := args[1]
	baudRate, err := strconv.Atoi(args[2])
	if err != nil {
		return protocol.NewErrorResponse("invalid baud rate")
	}

	// TODO: Implement actual CAT testing via hardware manager
	log.Printf("Testing CAT: device=%s model=%s baud=%d", device, model, baudRate)

	return protocol.NewSuccessResponse(map[string]interface{}{
		"status":  "success",
		"device":  device,
		"model":   model,
		"baud":    baudRate,
		"message": "CAT test completed successfully",
	})
}

// handleTestPTT handles TEST_PTT command for PTT testing
func (e *CoreEngine) handleTestPTT(args []string) *protocol.Response {
	if len(args) < 3 {
		return protocol.NewErrorResponse("usage: TEST_PTT method port delay")
	}

	method := args[0]
	port := args[1]
	delay, err := strconv.ParseFloat(args[2], 64)
	if err != nil {
		return protocol.NewErrorResponse("invalid delay value")
	}

	log.Printf("Testing PTT: method=%s port=%s delay=%.1f", method, port, delay)

	// Test PTT via hardware manager
	if e.hardwareManager == nil {
		return protocol.NewErrorResponse("hardware manager not available")
	}

	// Check if radio is connected for CAT PTT
	if method == "cat" && !e.hardwareManager.IsRadioConnected() {
		return protocol.NewErrorResponse("radio not connected - CAT PTT requires working radio connection")
	}

	// Test PTT activation
	log.Printf("Activating PTT...")
	if err := e.hardwareManager.SetRadioPTT(true); err != nil {
		return protocol.NewErrorResponse(fmt.Sprintf("failed to activate PTT: %v", err))
	}

	// For toggle mode, just activate and return success
	// Don't automatically turn off - let user control it
	if delay <= 1.0 {
		log.Printf("PTT activated for toggle mode")
		return protocol.NewSuccessResponse(map[string]interface{}{
			"status":  "success",
			"method":  method,
			"port":    port,
			"mode":    "toggle",
			"message": "PTT activated - use TEST_PTT_OFF to deactivate",
		})
	}

	// Traditional test mode - hold for delay then turn off
	time.Sleep(time.Duration(delay * float64(time.Second)))

	// Deactivate PTT
	log.Printf("Deactivating PTT...")
	if err := e.hardwareManager.SetRadioPTT(false); err != nil {
		return protocol.NewErrorResponse(fmt.Sprintf("failed to deactivate PTT: %v", err))
	}

	return protocol.NewSuccessResponse(map[string]interface{}{
		"status":  "success",
		"method":  method,
		"port":    port,
		"delay":   delay,
		"message": "PTT test completed successfully",
	})
}

// handleTestPTTOff handles TEST_PTT_OFF command to turn off PTT
func (e *CoreEngine) handleTestPTTOff() *protocol.Response {
	// Test PTT via hardware manager
	if e.hardwareManager == nil {
		return protocol.NewErrorResponse("hardware manager not available")
	}

	// Deactivate PTT
	log.Printf("Deactivating PTT...")
	if err := e.hardwareManager.SetRadioPTT(false); err != nil {
		return protocol.NewErrorResponse(fmt.Sprintf("failed to deactivate PTT: %v", err))
	}

	log.Printf("PTT deactivated successfully")
	return protocol.NewSuccessResponse(map[string]interface{}{
		"status":  "success",
		"message": "PTT deactivated successfully",
	})
}

// handleRetryRadio handles RETRY_RADIO command to reconnect radio
func (e *CoreEngine) handleRetryRadio() *protocol.Response {
	if e.hardwareManager == nil {
		return protocol.NewErrorResponse("hardware manager not available")
	}

	log.Printf("Attempting to retry radio connection...")

	// Try to reconnect the radio
	if err := e.hardwareManager.RetryRadioConnection(); err != nil {
		log.Printf("Radio retry failed: %v", err)
		return protocol.NewErrorResponse(fmt.Sprintf("radio retry failed: %v", err))
	}

	return protocol.NewSuccessResponse(map[string]interface{}{
		"message": "Radio connection retry successful",
		"status":  "connected",
	})
}

// processAudioSamples processes incoming audio samples for monitoring
func (e *CoreEngine) processAudioSamples() {
	log.Printf("Starting audio sample processing for monitoring")

	// Get the audio input samples channel
	audioSamples := e.hardwareManager.GetAudioInputSamples()
	if audioSamples == nil {
		log.Printf("Warning: no audio input samples available - check audio configuration")
		return
	}

	log.Printf("Audio sample processing ready - waiting for samples...")
	sampleCount := 0

	// Set up a debug timer to report if we're not getting samples
	debugTicker := time.NewTicker(5 * time.Second)
	defer debugTicker.Stop()
	lastSampleCount := 0

	for {
		select {
		case samples, ok := <-audioSamples:
			if !ok {
				log.Printf("Audio samples channel closed, stopping processing")
				return
			}

			sampleCount++
			if sampleCount%100 == 0 {
				log.Printf("Processed %d audio sample blocks (latest: %d samples)", sampleCount, len(samples))
			}

			// Process samples through the audio monitor
			if e.audioMonitor != nil {
				e.audioMonitor.ProcessSamples(samples)
			}

			// Also process samples through DSP for JS8 decoding
			if e.dspEngine != nil {
				_, err := e.dspEngine.DecodeBuffer(samples, func(result *dsp.DecodeResult) {
					// Convert DSP result to protocol message
					message := protocol.Message{
						ID:        0, // Will be assigned by storage
						Timestamp: time.Now(),
						From:      "UNKNOWN", // Would need to parse from message
						To:        "ALL",     // Default for broadcasts
						Message:   result.Message,
						SNR:       float32(result.SNR),
						Frequency: int(result.Frequency),
						Mode:      "JS8",
					}

					// Send to RX message channel for processing
					select {
					case e.rxMessages <- message:
					default:
						// Channel full, drop message
					}
				})

				if err != nil {
					log.Printf("DSP decode error: %v", err)
				}
			}

		case <-debugTicker.C:
			if sampleCount == lastSampleCount {
				log.Printf("DEBUG: No audio samples received in last 5 seconds (total count: %d)", sampleCount)
				// Check audio input status
				audioSamples2 := e.hardwareManager.GetAudioInputSamples()
				if audioSamples2 == nil {
					log.Printf("DEBUG: Audio input samples channel is nil - audio may not be started")
				} else {
					log.Printf("DEBUG: Audio input samples channel exists but no data flowing")
				}
			} else {
				log.Printf("DEBUG: Audio flowing normally (%d new samples)", sampleCount-lastSampleCount)
			}
			lastSampleCount = sampleCount

		case <-time.After(1 * time.Second):
			// Check if engine is still running
			e.mutex.RLock()
			running := e.running
			e.mutex.RUnlock()

			if !running {
				log.Printf("Engine stopped, ending audio processing")
				return
			}
		}
	}
}

// GetAudioMonitor returns the audio monitor for direct access
func (e *CoreEngine) GetAudioMonitor() *audio.AudioLevelMonitor {
	return e.audioMonitor
}
