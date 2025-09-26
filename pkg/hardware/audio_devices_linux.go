//go:build !darwin

package hardware

import "fmt"

// AudioDevice represents an audio device (fallback implementation)
type AudioDevice struct {
	ID       uint32 `json:"id"`
	Name     string `json:"name"`
	IsInput  bool   `json:"is_input"`
	IsOutput bool   `json:"is_output"`
}

// GetAudioDevices returns an error on non-macOS systems
func GetAudioDevices() ([]AudioDevice, error) {
	return nil, fmt.Errorf("audio device enumeration not supported on this platform")
}