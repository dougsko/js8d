package audio

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// AudioConfig represents audio system configuration
type AudioConfig struct {
	InputDevice  string
	OutputDevice string
	SampleRate   int
	BufferSize   int
}

// AudioSystem manages audio input/output for JS8 operations
type AudioSystem struct {
	config     AudioConfig
	sampleRate int
	bufferSize int

	// Audio state
	recording bool
	playing   bool
	mutex     sync.RWMutex

	// Channels for audio data
	inputSamples  chan []int16
	outputSamples chan []int16

	// Mock audio state (for now)
	mockInput  bool
	mockOutput bool
}

// NewAudioSystem creates a new audio system
func NewAudioSystem(config AudioConfig) *AudioSystem {
	return &AudioSystem{
		config:        config,
		sampleRate:    config.SampleRate,
		bufferSize:    config.BufferSize,
		inputSamples:  make(chan []int16, 10),
		outputSamples: make(chan []int16, 10),
		mockInput:     true, // Start with mock until real drivers implemented
		mockOutput:    true,
	}
}

// Initialize initializes the audio system
func (a *AudioSystem) Initialize() error {
	log.Printf("Initializing audio system...")
	log.Printf("Input device: %s", a.config.InputDevice)
	log.Printf("Output device: %s", a.config.OutputDevice)
	log.Printf("Sample rate: %d Hz", a.sampleRate)
	log.Printf("Buffer size: %d samples", a.bufferSize)

	// TODO: Initialize ALSA or other audio system
	// For now, use mock implementation

	if a.mockInput {
		log.Printf("Using mock audio input")
	}

	if a.mockOutput {
		log.Printf("Using mock audio output")
	}

	return nil
}

// StartInput starts audio input capture
func (a *AudioSystem) StartInput() error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if a.recording {
		return fmt.Errorf("audio input already started")
	}

	a.recording = true

	if a.mockInput {
		go a.mockInputWorker()
	} else {
		// TODO: Start real audio input
		return fmt.Errorf("real audio input not implemented yet")
	}

	log.Printf("Audio input started")
	return nil
}

// StopInput stops audio input capture
func (a *AudioSystem) StopInput() error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	a.recording = false
	log.Printf("Audio input stopped")
	return nil
}

// StartOutput starts audio output
func (a *AudioSystem) StartOutput() error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if a.playing {
		return fmt.Errorf("audio output already started")
	}

	a.playing = true

	if a.mockOutput {
		go a.mockOutputWorker()
	} else {
		// TODO: Start real audio output
		return fmt.Errorf("real audio output not implemented yet")
	}

	log.Printf("Audio output started")
	return nil
}

// StopOutput stops audio output
func (a *AudioSystem) StopOutput() error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	a.playing = false
	log.Printf("Audio output stopped")
	return nil
}

// PlayAudio queues audio samples for output
func (a *AudioSystem) PlayAudio(samples []int16) error {
	if !a.isPlaying() {
		return fmt.Errorf("audio output not started")
	}

	select {
	case a.outputSamples <- samples:
		return nil
	default:
		return fmt.Errorf("audio output buffer full")
	}
}

// GetInputSamples returns a channel for receiving input audio samples
func (a *AudioSystem) GetInputSamples() <-chan []int16 {
	return a.inputSamples
}

// Close shuts down the audio system
func (a *AudioSystem) Close() error {
	a.StopInput()
	a.StopOutput()

	// Close channels
	close(a.inputSamples)
	close(a.outputSamples)

	log.Printf("Audio system closed")
	return nil
}

// mockInputWorker simulates audio input with background noise
func (a *AudioSystem) mockInputWorker() {
	ticker := time.NewTicker(time.Duration(a.bufferSize*1000/a.sampleRate) * time.Millisecond)
	defer ticker.Stop()

	for a.isRecording() {
		select {
		case <-ticker.C:
			// Generate mock background noise
			samples := make([]int16, a.bufferSize)
			for i := range samples {
				// Very quiet random noise
				samples[i] = int16((time.Now().UnixNano() % 200) - 100)
			}

			select {
			case a.inputSamples <- samples:
			default:
				// Drop samples if buffer full
			}
		}
	}
}

// mockOutputWorker simulates audio output by consuming samples
func (a *AudioSystem) mockOutputWorker() {
	for a.isPlaying() {
		select {
		case samples := <-a.outputSamples:
			// Simulate playing audio by consuming samples
			duration := time.Duration(len(samples)*1000/a.sampleRate) * time.Millisecond
			log.Printf("Mock audio: Playing %d samples (%.1fms)", len(samples), float64(duration)/float64(time.Millisecond))
			time.Sleep(duration)

		case <-time.After(100 * time.Millisecond):
			// Keep the worker alive even when no audio
			continue
		}
	}
}

// isRecording checks if audio input is active
func (a *AudioSystem) isRecording() bool {
	a.mutex.RLock()
	defer a.mutex.RUnlock()
	return a.recording
}

// isPlaying checks if audio output is active
func (a *AudioSystem) isPlaying() bool {
	a.mutex.RLock()
	defer a.mutex.RUnlock()
	return a.playing
}

// GetSampleRate returns the current sample rate
func (a *AudioSystem) GetSampleRate() int {
	return a.sampleRate
}

// GetBufferSize returns the current buffer size
func (a *AudioSystem) GetBufferSize() int {
	return a.bufferSize
}

// IsRecording returns whether audio input is active
func (a *AudioSystem) IsRecording() bool {
	return a.isRecording()
}

// IsPlaying returns whether audio output is active
func (a *AudioSystem) IsPlaying() bool {
	return a.isPlaying()
}
