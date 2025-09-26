// +build !darwin,!linux

package hardware

// NewPlatformAudio creates a mock audio implementation for unsupported platforms
func NewPlatformAudio(config PlatformAudioConfig) AudioInterface {
	mockConfig := MockAudioConfig{
		InputDevice:  config.InputDevice,
		OutputDevice: config.OutputDevice,
		SampleRate:   config.SampleRate,
		BufferSize:   config.BufferSize,
		Channels:     config.Channels,
	}
	return NewMockAudio(mockConfig)
}

// PlatformAudioConfig represents cross-platform audio configuration
type PlatformAudioConfig struct {
	InputDevice  string
	OutputDevice string
	SampleRate   int
	BufferSize   int
	Channels     int
}