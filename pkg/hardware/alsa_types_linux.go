//go:build linux

package hardware

// ALSAAudioConfig represents ALSA audio configuration
type ALSAAudioConfig struct {
	InputDevice  string
	OutputDevice string
	SampleRate   int
	BufferSize   int
	Channels     int
}

// Fallback tryCreateALSAAudio when ALSA is not available
// This will be overridden by the real implementation if ALSA CGO builds successfully
var tryCreateALSAAudio = func(ALSAAudioConfig) AudioInterface {
	return nil // ALSA not available
}