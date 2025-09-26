//go:build darwin

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

