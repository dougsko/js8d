// +build linux

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
	return NewALSAAudio(alsaConfig)
}

// PlatformAudioConfig represents cross-platform audio configuration
type PlatformAudioConfig struct {
	InputDevice  string
	OutputDevice string
	SampleRate   int
	BufferSize   int
	Channels     int
}