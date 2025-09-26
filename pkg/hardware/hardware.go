package hardware

import (
	"fmt"
	"log"
	"sync"
)

// HardwareConfig represents hardware configuration
type HardwareConfig struct {
	EnableGPIO     bool
	PTTGPIOPin     int
	StatusLEDPin   int
	EnableOLED     bool
	OLEDI2CAddress int
	OLEDWidth      int
	OLEDHeight     int
	EnableAudio    bool
	AudioInput     string
	AudioOutput    string
	SampleRate     int
	BufferSize     int
	EnableRadio    bool
	RadioModel     string
	RadioDevice    string
	RadioBaudRate  int
}

// HardwareManager manages all hardware interfaces
type HardwareManager struct {
	config HardwareConfig
	mutex  sync.RWMutex

	// Hardware interfaces
	gpio      GPIOInterface
	oled      OLEDInterface
	audio     AudioInterface
	radio     RadioInterface
	pttActive bool

	// State
	initialized bool
}

// GPIOInterface defines GPIO operations
type GPIOInterface interface {
	Initialize() error
	Close() error
	SetPin(pin int, value bool) error
	GetPin(pin int) (bool, error)
}

// OLEDInterface defines OLED display operations
type OLEDInterface interface {
	Initialize() error
	Close() error
	Clear() error
	WriteLine(line int, text string) error
	Display() error
	GetWidth() int
	GetHeight() int
}

// AudioInterface defines audio I/O operations
type AudioInterface interface {
	Initialize() error
	Close() error
	StartInput() error
	StopInput() error
	StartOutput() error
	StopOutput() error
	PlayAudio(samples []int16) error
	GetInputSamples() <-chan []int16
	GetSampleRate() int
	GetBufferSize() int
	IsRecording() bool
	IsPlaying() bool
}

// NewHardwareManager creates a new hardware manager
func NewHardwareManager(config HardwareConfig) *HardwareManager {
	return &HardwareManager{
		config: config,
	}
}

// Initialize initializes all hardware interfaces
func (h *HardwareManager) Initialize() error {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if h.initialized {
		return nil
	}

	log.Printf("Hardware: Initializing hardware manager...")

	// Initialize GPIO if enabled
	if h.config.EnableGPIO {
		log.Printf("Hardware: Initializing GPIO...")

		// Use mock GPIO for now - will be replaced with real implementation
		h.gpio = NewMockGPIO()
		if err := h.gpio.Initialize(); err != nil {
			return fmt.Errorf("failed to initialize GPIO: %w", err)
		}
		log.Printf("Hardware: GPIO initialized (PTT pin: %d, LED pin: %d)",
			h.config.PTTGPIOPin, h.config.StatusLEDPin)
	}

	// Initialize OLED if enabled
	if h.config.EnableOLED {
		log.Printf("Hardware: Initializing OLED...")

		// Use mock OLED for now - will be replaced with real implementation
		h.oled = NewMockOLED(h.config.OLEDWidth, h.config.OLEDHeight)
		if err := h.oled.Initialize(); err != nil {
			return fmt.Errorf("failed to initialize OLED: %w", err)
		}
		log.Printf("Hardware: OLED initialized (%dx%d at I2C 0x%02x)",
			h.config.OLEDWidth, h.config.OLEDHeight, h.config.OLEDI2CAddress)
	}

	// Initialize Audio if enabled
	if h.config.EnableAudio {
		log.Printf("Hardware: Initializing Audio...")

		// Use platform-specific audio implementation
		audioConfig := PlatformAudioConfig{
			InputDevice:  h.config.AudioInput,
			OutputDevice: h.config.AudioOutput,
			SampleRate:   h.config.SampleRate,
			BufferSize:   h.config.BufferSize,
			Channels:     1, // Mono for radio
		}
		h.audio = NewPlatformAudio(audioConfig)
		if err := h.audio.Initialize(); err != nil {
			return fmt.Errorf("failed to initialize audio: %w", err)
		}
		log.Printf("Hardware: Audio initialized (%s -> %s, %d Hz)",
			h.config.AudioInput, h.config.AudioOutput, h.config.SampleRate)
	}

	// Initialize Radio if enabled
	if h.config.EnableRadio {
		log.Printf("Hardware: Initializing Radio...")

		// Use Hamlib for radio control
		radioConfig := RadioConfig{
			Model:    h.config.RadioModel,
			Device:   h.config.RadioDevice,
			BaudRate: h.config.RadioBaudRate,
			Enabled:  true,
		}

		// Use mock radio for testing - Hamlib CGO needs compatibility fixes
		h.radio = NewMockRadio(radioConfig)
		if err := h.radio.Initialize(); err != nil {
			return fmt.Errorf("failed to initialize radio: %w", err)
		}
		log.Printf("Hardware: Radio initialized (%s on %s)",
			h.config.RadioModel, h.config.RadioDevice)
	}

	h.initialized = true
	log.Printf("Hardware: Hardware manager initialized successfully")
	return nil
}

// Close shuts down all hardware interfaces
func (h *HardwareManager) Close() error {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if !h.initialized {
		return nil
	}

	log.Printf("Hardware: Shutting down hardware manager...")

	// Turn off PTT if active
	if h.pttActive {
		h.setPTTLocked(false)
	}

	// Close Radio
	if h.radio != nil {
		if err := h.radio.Close(); err != nil {
			log.Printf("Hardware: Error closing Radio: %v", err)
		}
	}

	// Close Audio
	if h.audio != nil {
		if err := h.audio.Close(); err != nil {
			log.Printf("Hardware: Error closing Audio: %v", err)
		}
	}

	// Close OLED
	if h.oled != nil {
		if err := h.oled.Close(); err != nil {
			log.Printf("Hardware: Error closing OLED: %v", err)
		}
	}

	// Close GPIO
	if h.gpio != nil {
		if err := h.gpio.Close(); err != nil {
			log.Printf("Hardware: Error closing GPIO: %v", err)
		}
	}

	h.initialized = false
	log.Printf("Hardware: Hardware manager shut down")
	return nil
}

// SetPTT controls the PTT (Push-To-Talk) output
func (h *HardwareManager) SetPTT(active bool) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	return h.setPTTLocked(active)
}

// setPTTLocked sets PTT state (must be called with lock held)
func (h *HardwareManager) setPTTLocked(active bool) error {
	if !h.initialized || !h.config.EnableGPIO || h.gpio == nil {
		// Just log for mock mode
		if h.pttActive != active {
			log.Printf("Hardware: PTT %s (mock)", map[bool]string{true: "ON", false: "OFF"}[active])
		}
		h.pttActive = active
		return nil
	}

	if h.pttActive != active {
		if err := h.gpio.SetPin(h.config.PTTGPIOPin, active); err != nil {
			return fmt.Errorf("failed to set PTT: %w", err)
		}
		h.pttActive = active
		log.Printf("Hardware: PTT %s (GPIO pin %d)",
			map[bool]string{true: "ON", false: "OFF"}[active], h.config.PTTGPIOPin)
	}

	return nil
}

// GetPTT returns the current PTT state
func (h *HardwareManager) GetPTT() bool {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return h.pttActive
}

// SetStatusLED controls the status LED
func (h *HardwareManager) SetStatusLED(active bool) error {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	if !h.initialized || !h.config.EnableGPIO || h.gpio == nil {
		// Just log for mock mode
		log.Printf("Hardware: Status LED %s (mock)", map[bool]string{true: "ON", false: "OFF"}[active])
		return nil
	}

	if err := h.gpio.SetPin(h.config.StatusLEDPin, active); err != nil {
		return fmt.Errorf("failed to set status LED: %w", err)
	}

	log.Printf("Hardware: Status LED %s (GPIO pin %d)",
		map[bool]string{true: "ON", false: "OFF"}[active], h.config.StatusLEDPin)
	return nil
}

// UpdateOLED updates the OLED display with station information
func (h *HardwareManager) UpdateOLED(callsign, grid string, frequency int, lastMessage string) error {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	if !h.initialized || !h.config.EnableOLED || h.oled == nil {
		// Just log for mock mode
		log.Printf("Hardware: OLED update (mock): %s %s | %.3f MHz | %s",
			callsign, grid, float64(frequency)/1000000.0, lastMessage)
		return nil
	}

	// Clear display
	if err := h.oled.Clear(); err != nil {
		return fmt.Errorf("failed to clear OLED: %w", err)
	}

	// Line 1: Station info
	line1 := fmt.Sprintf("%s %s", callsign, grid)
	if err := h.oled.WriteLine(0, line1); err != nil {
		return fmt.Errorf("failed to write OLED line 1: %w", err)
	}

	// Line 2: Frequency
	line2 := fmt.Sprintf("%.3f MHz", float64(frequency)/1000000.0)
	if err := h.oled.WriteLine(1, line2); err != nil {
		return fmt.Errorf("failed to write OLED line 2: %w", err)
	}

	// Line 3: Last message (truncated)
	line3 := lastMessage
	if len(line3) > 20 { // Typical OLED width constraint
		line3 = line3[:17] + "..."
	}
	if err := h.oled.WriteLine(2, line3); err != nil {
		return fmt.Errorf("failed to write OLED line 3: %w", err)
	}

	// Update display
	if err := h.oled.Display(); err != nil {
		return fmt.Errorf("failed to update OLED display: %w", err)
	}

	return nil
}

// IsInitialized returns whether hardware is initialized
func (h *HardwareManager) IsInitialized() bool {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return h.initialized
}

// GetConfig returns the hardware configuration
func (h *HardwareManager) GetConfig() HardwareConfig {
	return h.config
}

// StartAudioInput starts audio input capture
func (h *HardwareManager) StartAudioInput() error {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	if !h.initialized || !h.config.EnableAudio || h.audio == nil {
		return fmt.Errorf("audio not initialized")
	}

	return h.audio.StartInput()
}

// StopAudioInput stops audio input capture
func (h *HardwareManager) StopAudioInput() error {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	if !h.initialized || !h.config.EnableAudio || h.audio == nil {
		return fmt.Errorf("audio not initialized")
	}

	return h.audio.StopInput()
}

// StartAudioOutput starts audio output
func (h *HardwareManager) StartAudioOutput() error {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	if !h.initialized || !h.config.EnableAudio || h.audio == nil {
		return fmt.Errorf("audio not initialized")
	}

	return h.audio.StartOutput()
}

// StopAudioOutput stops audio output
func (h *HardwareManager) StopAudioOutput() error {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	if !h.initialized || !h.config.EnableAudio || h.audio == nil {
		return fmt.Errorf("audio not initialized")
	}

	return h.audio.StopOutput()
}

// PlayAudio plays audio samples
func (h *HardwareManager) PlayAudio(samples []int16) error {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	if !h.initialized || !h.config.EnableAudio || h.audio == nil {
		return fmt.Errorf("audio not initialized")
	}

	return h.audio.PlayAudio(samples)
}

// GetAudioInputSamples returns the audio input samples channel
func (h *HardwareManager) GetAudioInputSamples() <-chan []int16 {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	if !h.initialized || !h.config.EnableAudio || h.audio == nil {
		return nil
	}

	return h.audio.GetInputSamples()
}

// GetAudio returns the audio interface for direct access
func (h *HardwareManager) GetAudio() AudioInterface {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return h.audio
}

// GetRadio returns the radio interface for direct access
func (h *HardwareManager) GetRadio() RadioInterface {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return h.radio
}

// SetRadioFrequency sets the radio frequency
func (h *HardwareManager) SetRadioFrequency(freq int64) error {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	if !h.initialized || !h.config.EnableRadio || h.radio == nil {
		return fmt.Errorf("radio not initialized")
	}

	return h.radio.SetFrequency(freq)
}

// GetRadioFrequency gets the current radio frequency
func (h *HardwareManager) GetRadioFrequency() (int64, error) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	if !h.initialized || !h.config.EnableRadio || h.radio == nil {
		return 0, fmt.Errorf("radio not initialized")
	}

	return h.radio.GetFrequency()
}

// SetRadioMode sets the radio mode and bandwidth
func (h *HardwareManager) SetRadioMode(mode string, bandwidth int) error {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	if !h.initialized || !h.config.EnableRadio || h.radio == nil {
		return fmt.Errorf("radio not initialized")
	}

	return h.radio.SetMode(mode, bandwidth)
}

// GetRadioMode gets the current radio mode and bandwidth
func (h *HardwareManager) GetRadioMode() (string, int, error) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	if !h.initialized || !h.config.EnableRadio || h.radio == nil {
		return "", 0, fmt.Errorf("radio not initialized")
	}

	return h.radio.GetMode()
}

// SetRadioPTT sets the radio PTT state
func (h *HardwareManager) SetRadioPTT(state bool) error {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	if !h.initialized || !h.config.EnableRadio || h.radio == nil {
		return fmt.Errorf("radio not initialized")
	}

	return h.radio.SetPTT(state)
}

// GetRadioPTT gets the current radio PTT state
func (h *HardwareManager) GetRadioPTT() (bool, error) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	if !h.initialized || !h.config.EnableRadio || h.radio == nil {
		return false, fmt.Errorf("radio not initialized")
	}

	return h.radio.GetPTT()
}

// GetRadioInfo gets radio information
func (h *HardwareManager) GetRadioInfo() (RadioInfo, error) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	if !h.initialized || !h.config.EnableRadio || h.radio == nil {
		return RadioInfo{}, fmt.Errorf("radio not initialized")
	}

	return h.radio.GetRadioInfo()
}

// IsRadioConnected returns whether the radio is connected
func (h *HardwareManager) IsRadioConnected() bool {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	if !h.initialized || !h.config.EnableRadio || h.radio == nil {
		return false
	}

	return h.radio.IsConnected()
}

// GetRadioPowerLevel gets the radio power level
func (h *HardwareManager) GetRadioPowerLevel() (float32, error) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	if !h.initialized || !h.config.EnableRadio || h.radio == nil {
		return 0, fmt.Errorf("radio not initialized")
	}

	return h.radio.GetPowerLevel()
}

// GetRadioSWRLevel gets the radio SWR level
func (h *HardwareManager) GetRadioSWRLevel() (float32, error) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	if !h.initialized || !h.config.EnableRadio || h.radio == nil {
		return 0, fmt.Errorf("radio not initialized")
	}

	return h.radio.GetSWRLevel()
}

// GetRadioSignalLevel gets the radio signal level
func (h *HardwareManager) GetRadioSignalLevel() (int, error) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	if !h.initialized || !h.config.EnableRadio || h.radio == nil {
		return 0, fmt.Errorf("radio not initialized")
	}

	return h.radio.GetSignalLevel()
}

// PlatformAudioConfig represents cross-platform audio configuration
type PlatformAudioConfig struct {
	InputDevice  string
	OutputDevice string
	SampleRate   int
	BufferSize   int
	Channels     int
}