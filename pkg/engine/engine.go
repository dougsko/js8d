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
	"github.com/js8call/js8d/pkg/dsp"
	"github.com/js8call/js8d/pkg/hardware"
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

	// DSP and hardware components
	dspEngine       *dsp.DSP
	hardwareManager *hardware.HardwareManager

	// Message storage
	messages []protocol.Message
	msgMutex sync.RWMutex

	// Radio state
	frequency int
	ptt       bool
	connected bool

	// Channels for message processing
	rxMessages chan protocol.Message
	txMessages chan protocol.Message
}

// NewCoreEngine creates a new core engine
func NewCoreEngine(cfg *config.Config, socketPath string) *CoreEngine {
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

	return &CoreEngine{
		config:          cfg,
		socketPath:      socketPath,
		startTime:       time.Now(),
		frequency:       14078000, // Default JS8 frequency
		connected:       true,     // Mock - assume connected
		rxMessages:      make(chan protocol.Message, 100),
		txMessages:      make(chan protocol.Message, 100),
		messages:        make([]protocol.Message, 0),
		dspEngine:       dsp.NewDSP(),
		hardwareManager: hardware.NewHardwareManager(hardwareConfig),
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
	if err := e.hardwareManager.StartAudioInput(); err != nil {
		log.Printf("Warning: failed to start audio input: %v", err)
	}

	// Start audio output for transmission
	if err := e.hardwareManager.StartAudioOutput(); err != nil {
		log.Printf("Warning: failed to start audio output: %v", err)
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

// Stop stops the core engine
func (e *CoreEngine) Stop() error {
	e.mutex.Lock()
	e.running = false
	e.mutex.Unlock()

	if e.listener != nil {
		e.listener.Close()
	}

	// Clean up hardware manager
	if e.hardwareManager != nil {
		e.hardwareManager.Close()
	}

	// Clean up DSP engine
	if e.dspEngine != nil {
		e.dspEngine.Close()
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

			// Store message
			e.msgMutex.Lock()
			e.messages = append(e.messages, msg)
			e.msgMutex.Unlock()

			// Update OLED display with received message
			e.updateOLEDDisplay(fmt.Sprintf("RX: %s", msg.Message))

			// Handle auto-replies for directed messages
			e.handleAutoReply(msg)

		case msg := <-e.txMessages:
			log.Printf("TX: %s -> %s: %s", msg.From, msg.To, msg.Message)

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
	// Set PTT flag and hardware PTT during transmission
	e.mutex.Lock()
	e.ptt = true
	e.mutex.Unlock()

	// Activate hardware PTT
	if err := e.hardwareManager.SetPTT(true); err != nil {
		log.Printf("Warning: failed to set PTT: %v", err)
	}

	defer func() {
		// Deactivate hardware PTT
		if err := e.hardwareManager.SetPTT(false); err != nil {
			log.Printf("Warning: failed to clear PTT: %v", err)
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

	// Wait for transmission to complete
	duration := e.dspEngine.EstimateAudioDuration(mode)
	time.Sleep(duration)

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

	// Format heartbeat message: "HB AUTO callsign grid"
	var hbMessage string
	if grid != "" {
		hbMessage = fmt.Sprintf("HB AUTO %s %s", callsign, grid)
	} else {
		hbMessage = fmt.Sprintf("HB AUTO %s", callsign)
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
