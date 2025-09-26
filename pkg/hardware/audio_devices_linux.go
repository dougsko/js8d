//go:build !darwin && !linux

package hardware

import "fmt"

// GetAudioDevices returns an error on unsupported platforms
func GetAudioDevices() ([]AudioDevice, error) {
	return nil, fmt.Errorf("audio device enumeration not supported on this platform")
}