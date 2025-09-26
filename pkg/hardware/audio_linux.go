//go:build linux

package hardware

import "log"

// NewPlatformAudio creates the appropriate audio implementation for Linux
func NewPlatformAudio(config PlatformAudioConfig) AudioInterface {
	alsaConfig := ALSAAudioConfig{
		InputDevice:  config.InputDevice,
		OutputDevice: config.OutputDevice,
		SampleRate:   config.SampleRate,
		BufferSize:   config.BufferSize,
		Channels:     config.Channels,
	}

	log.Printf("Audio: Attempting to initialize ALSA audio system...")
	log.Printf("Audio: Input device: '%s', Output device: '%s'", config.InputDevice, config.OutputDevice)

	// Try to create ALSA audio, fall back to mock if ALSA not available
	if audio := tryCreateALSAAudio(alsaConfig); audio != nil {
		log.Printf("Audio: Successfully initialized ALSA audio system")
		return audio
	}

	// Fallback to mock audio if ALSA is not available
	log.Printf("Audio: WARNING - ALSA audio initialization failed, falling back to mock audio")
	log.Printf("Audio: This means no real audio input/output will be available")
	log.Printf("Audio: Please check your audio device configuration and ALSA installation")

	mockConfig := MockAudioConfig{
		InputDevice:  config.InputDevice,
		OutputDevice: config.OutputDevice,
		SampleRate:   config.SampleRate,
		BufferSize:   config.BufferSize,
		Channels:     config.Channels,
	}

	mockAudio := NewMockAudio(mockConfig)
	log.Printf("Audio: Mock audio system initialized (no real audio I/O)")
	return mockAudio
}

