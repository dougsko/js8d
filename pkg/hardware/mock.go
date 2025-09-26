package hardware

import (
	"fmt"
	"log"
	"sync"
)

// MockGPIO implements GPIOInterface for testing
type MockGPIO struct {
	pins map[int]bool
	mu   sync.RWMutex
}

// NewMockGPIO creates a new mock GPIO interface
func NewMockGPIO() *MockGPIO {
	return &MockGPIO{
		pins: make(map[int]bool),
	}
}

// Initialize initializes the mock GPIO
func (g *MockGPIO) Initialize() error {
	log.Printf("MockGPIO: Initialized")
	return nil
}

// Close closes the mock GPIO
func (g *MockGPIO) Close() error {
	log.Printf("MockGPIO: Closed")
	return nil
}

// SetPin sets a GPIO pin value
func (g *MockGPIO) SetPin(pin int, value bool) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.pins[pin] = value
	log.Printf("MockGPIO: Pin %d set to %t", pin, value)
	return nil
}

// GetPin gets a GPIO pin value
func (g *MockGPIO) GetPin(pin int) (bool, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	value := g.pins[pin]
	return value, nil
}

// MockOLED implements OLEDInterface for testing
type MockOLED struct {
	width  int
	height int
	lines  map[int]string
	mu     sync.RWMutex
}

// NewMockOLED creates a new mock OLED interface
func NewMockOLED(width, height int) *MockOLED {
	return &MockOLED{
		width:  width,
		height: height,
		lines:  make(map[int]string),
	}
}

// Initialize initializes the mock OLED
func (o *MockOLED) Initialize() error {
	log.Printf("MockOLED: Initialized (%dx%d)", o.width, o.height)
	return nil
}

// Close closes the mock OLED
func (o *MockOLED) Close() error {
	log.Printf("MockOLED: Closed")
	return nil
}

// Clear clears the mock OLED display
func (o *MockOLED) Clear() error {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.lines = make(map[int]string)
	return nil
}

// WriteLine writes a line to the mock OLED
func (o *MockOLED) WriteLine(line int, text string) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if line < 0 || line >= o.height/8 { // Assuming 8 pixels per line
		return fmt.Errorf("line %d out of range", line)
	}

	o.lines[line] = text
	return nil
}

// Display updates the mock OLED display
func (o *MockOLED) Display() error {
	o.mu.RLock()
	defer o.mu.RUnlock()

	log.Printf("MockOLED: Display updated:")
	for i := 0; i < o.height/8; i++ {
		if text, exists := o.lines[i]; exists && text != "" {
			log.Printf("MockOLED:   Line %d: %s", i, text)
		}
	}
	return nil
}

// GetWidth returns the mock OLED width
func (o *MockOLED) GetWidth() int {
	return o.width
}

// GetHeight returns the mock OLED height
func (o *MockOLED) GetHeight() int {
	return o.height
}

// MockAudio implements AudioInterface for testing
type MockAudio struct {
	config       MockAudioConfig
	recording    bool
	playing      bool
	mutex        sync.RWMutex
	inputSamples chan []int16
	stopChan     chan struct{}
}

// MockAudioConfig represents mock audio configuration
type MockAudioConfig struct {
	InputDevice  string
	OutputDevice string
	SampleRate   int
	BufferSize   int
	Channels     int
}

// NewMockAudio creates a new mock audio interface
func NewMockAudio(config MockAudioConfig) *MockAudio {
	if config.SampleRate == 0 {
		config.SampleRate = 48000
	}
	if config.BufferSize == 0 {
		config.BufferSize = 1024
	}
	if config.Channels == 0 {
		config.Channels = 1
	}

	return &MockAudio{
		config:       config,
		inputSamples: make(chan []int16, 10),
		stopChan:     make(chan struct{}),
	}
}

// Initialize initializes the mock audio system
func (a *MockAudio) Initialize() error {
	log.Printf("MockAudio: Initialized - %d Hz, %d channels, %d buffer",
		a.config.SampleRate, a.config.Channels, a.config.BufferSize)
	return nil
}

// Close shuts down the mock audio system
func (a *MockAudio) Close() error {
	close(a.stopChan)
	a.StopInput()
	a.StopOutput()
	close(a.inputSamples)
	log.Printf("MockAudio: Closed")
	return nil
}

// StartInput starts mock audio input
func (a *MockAudio) StartInput() error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if a.recording {
		return fmt.Errorf("audio input already started")
	}

	a.recording = true
	log.Printf("MockAudio: Input started")
	return nil
}

// StopInput stops mock audio input
func (a *MockAudio) StopInput() error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	a.recording = false
	log.Printf("MockAudio: Input stopped")
	return nil
}

// StartOutput starts mock audio output
func (a *MockAudio) StartOutput() error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if a.playing {
		return fmt.Errorf("audio output already started")
	}

	a.playing = true
	log.Printf("MockAudio: Output started")
	return nil
}

// StopOutput stops mock audio output
func (a *MockAudio) StopOutput() error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	a.playing = false
	log.Printf("MockAudio: Output stopped")
	return nil
}

// PlayAudio simulates playing audio samples
func (a *MockAudio) PlayAudio(samples []int16) error {
	if !a.IsPlaying() {
		return fmt.Errorf("audio output not started")
	}

	log.Printf("MockAudio: Playing %d samples", len(samples))
	return nil
}

// GetInputSamples returns mock input samples channel
func (a *MockAudio) GetInputSamples() <-chan []int16 {
	return a.inputSamples
}

// GetSampleRate returns mock sample rate
func (a *MockAudio) GetSampleRate() int {
	return a.config.SampleRate
}

// GetBufferSize returns mock buffer size
func (a *MockAudio) GetBufferSize() int {
	return a.config.BufferSize
}

// IsRecording returns mock recording state
func (a *MockAudio) IsRecording() bool {
	a.mutex.RLock()
	defer a.mutex.RUnlock()
	return a.recording
}

// IsPlaying returns mock playing state
func (a *MockAudio) IsPlaying() bool {
	a.mutex.RLock()
	defer a.mutex.RUnlock()
	return a.playing
}