//go:build linux

package hardware

import (
	"fmt"
	"log"
	"sync"
	"time"
	"unsafe"
)

/*
#cgo pkg-config: alsa
#include <alsa/asoundlib.h>
#include <stdlib.h>

// Helper function to get error string
static const char* alsa_strerror_wrapper(int err) {
    return snd_strerror(err);
}
*/
import "C"


// ALSAAudio implements real ALSA audio I/O
type ALSAAudio struct {
	config ALSAAudioConfig

	// ALSA handles
	inputHandle  *C.snd_pcm_t
	outputHandle *C.snd_pcm_t

	// State
	recording bool
	playing   bool
	mutex     sync.RWMutex

	// Channels for audio data
	inputSamples  chan []int16
	outputSamples chan []int16

	// Worker control
	stopChan chan struct{}
}

// Override the fallback function with real ALSA implementation
func init() {
	tryCreateALSAAudio = func(config ALSAAudioConfig) AudioInterface {
		audio := NewALSAAudio(config)
		// Test if ALSA is actually available by trying to initialize
		if err := audio.Initialize(); err != nil {
			audio.Close()
			return nil
		}
		return audio
	}
}

// NewALSAAudio creates a new ALSA audio interface
func NewALSAAudio(config ALSAAudioConfig) *ALSAAudio {
	// Set defaults
	if config.SampleRate == 0 {
		config.SampleRate = 48000
	}
	if config.BufferSize == 0 {
		config.BufferSize = 1024
	}
	if config.Channels == 0 {
		config.Channels = 1 // Mono for radio applications
	}

	return &ALSAAudio{
		config:        config,
		inputSamples:  make(chan []int16, 10),
		outputSamples: make(chan []int16, 10),
		stopChan:      make(chan struct{}),
	}
}

// Initialize initializes the ALSA audio system
func (a *ALSAAudio) Initialize() error {
	log.Printf("ALSA: Initializing audio system...")
	log.Printf("ALSA: Input device: %s", a.config.InputDevice)
	log.Printf("ALSA: Output device: %s", a.config.OutputDevice)
	log.Printf("ALSA: Sample rate: %d Hz", a.config.SampleRate)
	log.Printf("ALSA: Buffer size: %d samples", a.config.BufferSize)

	// Initialize input device
	if a.config.InputDevice != "" {
		if err := a.initializeInput(); err != nil {
			return fmt.Errorf("failed to initialize input: %w", err)
		}
	}

	// Initialize output device
	if a.config.OutputDevice != "" {
		if err := a.initializeOutput(); err != nil {
			return fmt.Errorf("failed to initialize output: %w", err)
		}
	}

	log.Printf("ALSA: Audio system initialized successfully")
	return nil
}

// initializeInput initializes ALSA input device
func (a *ALSAAudio) initializeInput() error {
	log.Printf("ALSA: Setting up input device: %s", a.config.InputDevice)

	// Open PCM device for recording
	deviceName := C.CString(a.config.InputDevice)
	defer C.free(unsafe.Pointer(deviceName))

	ret := C.snd_pcm_open(&a.inputHandle, deviceName, C.SND_PCM_STREAM_CAPTURE, 0)
	if ret < 0 {
		return fmt.Errorf("unable to open input device %s: %s",
			a.config.InputDevice, C.GoString(C.alsa_strerror_wrapper(ret)))
	}

	// Configure hardware parameters
	if err := a.configureHardwareParams(a.inputHandle, "input"); err != nil {
		C.snd_pcm_close(a.inputHandle)
		return err
	}

	log.Printf("ALSA: Input device configured successfully")
	return nil
}

// initializeOutput initializes ALSA output device
func (a *ALSAAudio) initializeOutput() error {
	log.Printf("ALSA: Setting up output device: %s", a.config.OutputDevice)

	// Open PCM device for playback
	deviceName := C.CString(a.config.OutputDevice)
	defer C.free(unsafe.Pointer(deviceName))

	ret := C.snd_pcm_open(&a.outputHandle, deviceName, C.SND_PCM_STREAM_PLAYBACK, 0)
	if ret < 0 {
		return fmt.Errorf("unable to open output device %s: %s",
			a.config.OutputDevice, C.GoString(C.alsa_strerror_wrapper(ret)))
	}

	// Configure hardware parameters
	if err := a.configureHardwareParams(a.outputHandle, "output"); err != nil {
		C.snd_pcm_close(a.outputHandle)
		return err
	}

	log.Printf("ALSA: Output device configured successfully")
	return nil
}

// configureHardwareParams configures ALSA hardware parameters
func (a *ALSAAudio) configureHardwareParams(handle *C.snd_pcm_t, deviceType string) error {
	var params *C.snd_pcm_hw_params_t

	// Allocate parameters structure
	C.snd_pcm_hw_params_alloca(&params)

	// Initialize parameters with full configuration space
	ret := C.snd_pcm_hw_params_any(handle, params)
	if ret < 0 {
		return fmt.Errorf("unable to initialize hw params for %s: %s",
			deviceType, C.GoString(C.alsa_strerror_wrapper(ret)))
	}

	// Set access type to interleaved
	ret = C.snd_pcm_hw_params_set_access(handle, params, C.SND_PCM_ACCESS_RW_INTERLEAVED)
	if ret < 0 {
		return fmt.Errorf("unable to set access type for %s: %s",
			deviceType, C.GoString(C.alsa_strerror_wrapper(ret)))
	}

	// Set sample format to 16-bit signed little endian
	ret = C.snd_pcm_hw_params_set_format(handle, params, C.SND_PCM_FORMAT_S16_LE)
	if ret < 0 {
		return fmt.Errorf("unable to set format for %s: %s",
			deviceType, C.GoString(C.alsa_strerror_wrapper(ret)))
	}

	// Set number of channels
	ret = C.snd_pcm_hw_params_set_channels(handle, params, C.uint(a.config.Channels))
	if ret < 0 {
		return fmt.Errorf("unable to set channels for %s: %s",
			deviceType, C.GoString(C.alsa_strerror_wrapper(ret)))
	}

	// Set sample rate
	sampleRate := C.uint(a.config.SampleRate)
	ret = C.snd_pcm_hw_params_set_rate_near(handle, params, &sampleRate, nil)
	if ret < 0 {
		return fmt.Errorf("unable to set sample rate for %s: %s",
			deviceType, C.GoString(C.alsa_strerror_wrapper(ret)))
	}

	// Set buffer size
	bufferSize := C.snd_pcm_uframes_t(a.config.BufferSize)
	ret = C.snd_pcm_hw_params_set_buffer_size_near(handle, params, &bufferSize)
	if ret < 0 {
		return fmt.Errorf("unable to set buffer size for %s: %s",
			deviceType, C.GoString(C.alsa_strerror_wrapper(ret)))
	}

	// Apply parameters
	ret = C.snd_pcm_hw_params(handle, params)
	if ret < 0 {
		return fmt.Errorf("unable to set hw parameters for %s: %s",
			deviceType, C.GoString(C.alsa_strerror_wrapper(ret)))
	}

	log.Printf("ALSA: %s configured - %d Hz, %d channels, %d buffer",
		deviceType, int(sampleRate), a.config.Channels, int(bufferSize))
	return nil
}

// StartInput starts audio input capture
func (a *ALSAAudio) StartInput() error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if a.recording {
		return fmt.Errorf("audio input already started")
	}

	if a.inputHandle == nil {
		return fmt.Errorf("input device not initialized")
	}

	a.recording = true
	go a.inputWorker()

	log.Printf("ALSA: Audio input started")
	return nil
}

// StopInput stops audio input capture
func (a *ALSAAudio) StopInput() error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	a.recording = false
	log.Printf("ALSA: Audio input stopped")
	return nil
}

// StartOutput starts audio output
func (a *ALSAAudio) StartOutput() error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if a.playing {
		return fmt.Errorf("audio output already started")
	}

	if a.outputHandle == nil {
		return fmt.Errorf("output device not initialized")
	}

	a.playing = true
	go a.outputWorker()

	log.Printf("ALSA: Audio output started")
	return nil
}

// StopOutput stops audio output
func (a *ALSAAudio) StopOutput() error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	a.playing = false
	log.Printf("ALSA: Audio output stopped")
	return nil
}

// PlayAudio queues audio samples for output
func (a *ALSAAudio) PlayAudio(samples []int16) error {
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
func (a *ALSAAudio) GetInputSamples() <-chan []int16 {
	return a.inputSamples
}

// Close shuts down the ALSA audio system
func (a *ALSAAudio) Close() error {
	// Stop workers
	close(a.stopChan)

	// Stop input/output
	a.StopInput()
	a.StopOutput()

	// Close ALSA handles
	if a.inputHandle != nil {
		C.snd_pcm_close(a.inputHandle)
		a.inputHandle = nil
	}

	if a.outputHandle != nil {
		C.snd_pcm_close(a.outputHandle)
		a.outputHandle = nil
	}

	// Close channels
	close(a.inputSamples)
	close(a.outputSamples)

	log.Printf("ALSA: Audio system closed")
	return nil
}

// inputWorker captures audio from ALSA input device
func (a *ALSAAudio) inputWorker() {
	buffer := make([]int16, a.config.BufferSize*a.config.Channels)

	for a.isRecording() {
		ret := C.snd_pcm_readi(a.inputHandle,
			unsafe.Pointer(&buffer[0]),
			C.snd_pcm_uframes_t(a.config.BufferSize))

		if ret < 0 {
			// Handle underrun
			if ret == -C.EPIPE {
				log.Printf("ALSA: Input underrun, recovering...")
				C.snd_pcm_prepare(a.inputHandle)
				continue
			}
			log.Printf("ALSA: Input error: %s", C.GoString(C.alsa_strerror_wrapper(C.int(ret))))
			continue
		}

		// Copy samples to avoid race conditions
		samples := make([]int16, ret*C.snd_pcm_sframes_t(a.config.Channels))
		copy(samples, buffer[:ret*C.snd_pcm_sframes_t(a.config.Channels)])

		// Send to channel
		select {
		case a.inputSamples <- samples:
		default:
			// Drop samples if buffer full
		}
	}
}

// outputWorker plays audio to ALSA output device
func (a *ALSAAudio) outputWorker() {
	for a.isPlaying() {
		select {
		case samples := <-a.outputSamples:
			ret := C.snd_pcm_writei(a.outputHandle,
				unsafe.Pointer(&samples[0]),
				C.snd_pcm_uframes_t(len(samples)/a.config.Channels))

			if ret < 0 {
				// Handle underrun
				if ret == -C.EPIPE {
					log.Printf("ALSA: Output underrun, recovering...")
					C.snd_pcm_prepare(a.outputHandle)
					continue
				}
				log.Printf("ALSA: Output error: %s", C.GoString(C.alsa_strerror_wrapper(C.int(ret))))
				continue
			}

			log.Printf("ALSA: Played %d samples", len(samples))

		case <-a.stopChan:
			return

		case <-time.After(100 * time.Millisecond):
			// Keep the worker alive
			continue
		}
	}
}

// isRecording checks if audio input is active
func (a *ALSAAudio) isRecording() bool {
	a.mutex.RLock()
	defer a.mutex.RUnlock()
	return a.recording
}

// isPlaying checks if audio output is active
func (a *ALSAAudio) isPlaying() bool {
	a.mutex.RLock()
	defer a.mutex.RUnlock()
	return a.playing
}

// GetSampleRate returns the current sample rate
func (a *ALSAAudio) GetSampleRate() int {
	return a.config.SampleRate
}

// GetBufferSize returns the current buffer size
func (a *ALSAAudio) GetBufferSize() int {
	return a.config.BufferSize
}

// IsRecording returns whether audio input is active
func (a *ALSAAudio) IsRecording() bool {
	return a.isRecording()
}

// IsPlaying returns whether audio output is active
func (a *ALSAAudio) IsPlaying() bool {
	return a.isPlaying()
}