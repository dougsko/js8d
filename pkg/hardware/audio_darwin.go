// +build darwin

package hardware

// NewPlatformAudio creates the appropriate audio implementation for macOS
func NewPlatformAudio(config PlatformAudioConfig) AudioInterface {
	coreAudioConfig := CoreAudioConfig{
		InputDevice:  config.InputDevice,
		OutputDevice: config.OutputDevice,
		SampleRate:   config.SampleRate,
		BufferSize:   config.BufferSize,
		Channels:     config.Channels,
	}
	return NewCoreAudio(coreAudioConfig)
}

// PlatformAudioConfig represents cross-platform audio configuration
type PlatformAudioConfig struct {
	InputDevice  string
	OutputDevice string
	SampleRate   int
	BufferSize   int
	Channels     int
}