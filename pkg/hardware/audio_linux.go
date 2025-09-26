//go:build linux

package hardware

// NewPlatformAudio creates the appropriate audio implementation for Linux
func NewPlatformAudio(config PlatformAudioConfig) AudioInterface {
	alsaConfig := ALSAAudioConfig{
		InputDevice:  config.InputDevice,
		OutputDevice: config.OutputDevice,
		SampleRate:   config.SampleRate,
		BufferSize:   config.BufferSize,
		Channels:     config.Channels,
	}

	// Try to create ALSA audio, fall back to mock if ALSA not available
	if audio := tryCreateALSAAudio(alsaConfig); audio != nil {
		return audio
	}

	// Fallback to mock audio if ALSA is not available
	mockConfig := MockAudioConfig{
		InputDevice:  config.InputDevice,
		OutputDevice: config.OutputDevice,
		SampleRate:   config.SampleRate,
		BufferSize:   config.BufferSize,
		Channels:     config.Channels,
	}
	return NewMockAudio(mockConfig)
}

