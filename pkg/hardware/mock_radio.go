package hardware

import (
	"fmt"
	"log"
	"sync"
)

// MockRadio implements RadioInterface for testing
type MockRadio struct {
	config RadioConfig
	mutex  sync.RWMutex

	// Mock state
	connected bool
	frequency int64
	mode      string
	bandwidth int
	ptt       bool
	power     float32
	swr       float32
	signal    int
}

// NewMockRadio creates a new mock radio interface
func NewMockRadio(config RadioConfig) *MockRadio {
	return &MockRadio{
		config:    config,
		frequency: Band20m_JS8, // Default to 20m JS8 frequency
		mode:      ModeUSB,
		bandwidth: JS8Bandwidth,
		power:     0.5, // 50% power
		swr:       1.2, // Good SWR
		signal:    -10, // -10 dBm signal level
	}
}

// Initialize initializes the mock radio
func (r *MockRadio) Initialize() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	log.Printf("MockRadio: Initializing radio control (mock)...")
	log.Printf("MockRadio: Model: %s", r.config.Model)
	log.Printf("MockRadio: Device: %s", r.config.Device)
	log.Printf("MockRadio: Baud Rate: %d", r.config.BaudRate)

	r.connected = true
	log.Printf("MockRadio: Mock radio connection established")
	log.Printf("MockRadio: Frequency: %.3f MHz", float64(r.frequency)/1000000.0)
	log.Printf("MockRadio: Mode: %s", r.mode)

	return nil
}

// Close closes the mock radio connection
func (r *MockRadio) Close() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if !r.connected {
		return nil
	}

	log.Printf("MockRadio: Closing radio connection (mock)")
	r.connected = false
	return nil
}

// SetFrequency sets the mock radio frequency
func (r *MockRadio) SetFrequency(freq int64) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if !r.connected {
		return fmt.Errorf("radio not connected")
	}

	log.Printf("MockRadio: Setting frequency to %d Hz (%.3f MHz)", freq, float64(freq)/1000000.0)
	r.frequency = freq
	return nil
}

// GetFrequency gets the mock radio frequency
func (r *MockRadio) GetFrequency() (int64, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	if !r.connected {
		return 0, fmt.Errorf("radio not connected")
	}

	return r.frequency, nil
}

// SetMode sets the mock radio mode
func (r *MockRadio) SetMode(mode string, bandwidth int) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if !r.connected {
		return fmt.Errorf("radio not connected")
	}

	log.Printf("MockRadio: Setting mode to %s with bandwidth %d Hz", mode, bandwidth)
	r.mode = mode
	r.bandwidth = bandwidth
	return nil
}

// GetMode gets the mock radio mode
func (r *MockRadio) GetMode() (string, int, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	if !r.connected {
		return "", 0, fmt.Errorf("radio not connected")
	}

	return r.mode, r.bandwidth, nil
}

// SetPTT sets the mock PTT state
func (r *MockRadio) SetPTT(state bool) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if !r.connected {
		return fmt.Errorf("radio not connected")
	}

	if state != r.ptt {
		if state {
			log.Printf("MockRadio: PTT ON")
		} else {
			log.Printf("MockRadio: PTT OFF")
		}
		r.ptt = state
	}

	return nil
}

// GetPTT gets the mock PTT state
func (r *MockRadio) GetPTT() (bool, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	if !r.connected {
		return false, fmt.Errorf("radio not connected")
	}

	return r.ptt, nil
}

// GetRadioInfo gets mock radio information
func (r *MockRadio) GetRadioInfo() (RadioInfo, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	if !r.connected {
		return RadioInfo{}, fmt.Errorf("radio not connected")
	}

	info := RadioInfo{
		Model:        r.config.Model + " (Mock)",
		Manufacturer: "MockRadio Inc.",
		Version:      "1.0.0-mock",
		Capabilities: []string{
			"Frequency Control",
			"Mode Control",
			"PTT Control",
			"Power Level",
			"SWR Monitoring",
			"Signal Level",
		},
	}

	return info, nil
}

// IsConnected returns mock connection state
func (r *MockRadio) IsConnected() bool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return r.connected
}

// GetPowerLevel gets mock power level
func (r *MockRadio) GetPowerLevel() (float32, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	if !r.connected {
		return 0, fmt.Errorf("radio not connected")
	}

	return r.power, nil
}

// GetSWRLevel gets mock SWR level
func (r *MockRadio) GetSWRLevel() (float32, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	if !r.connected {
		return 0, fmt.Errorf("radio not connected")
	}

	return r.swr, nil
}

// GetSignalLevel gets mock signal level
func (r *MockRadio) GetSignalLevel() (int, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	if !r.connected {
		return 0, fmt.Errorf("radio not connected")
	}

	return r.signal, nil
}