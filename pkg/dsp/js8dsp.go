package dsp

import (
	"fmt"
	"time"
)

// JS8Mode represents JS8 submodes
type JS8Mode int

const (
	ModeNormal JS8Mode = 0
	ModeFast   JS8Mode = 1
	ModeTurbo  JS8Mode = 2
	ModeSlow   JS8Mode = 4
	ModeUltra  JS8Mode = 8
)

// DecodeResult represents a decoded JS8 message
type DecodeResult struct {
	UTC       int     `json:"utc"`
	SNR       int     `json:"snr"`
	DT        float32 `json:"dt"`
	Frequency float32 `json:"frequency"`
	Message   string  `json:"message"`
	Type      int     `json:"type"`
	Quality   float32 `json:"quality"`
	Mode      int     `json:"mode"`
}

// DSP represents the pure Go JS8 DSP engine
type DSP struct {
	encoder    *JS8Encoder
	sampleRate int
}

// NewDSP creates a new pure Go DSP instance
func NewDSP() *DSP {
	return &DSP{
		encoder:    NewJS8Encoder(),
		sampleRate: 12000, // Default JS8 sample rate
	}
}

// Initialize initializes the DSP library (pure Go - always succeeds)
func (d *DSP) Initialize() error {
	// Pure Go implementation needs no initialization
	return nil
}

// Close cleans up the DSP library (pure Go - nothing to clean up)
func (d *DSP) Close() {
	// Pure Go implementation needs no cleanup
}

// SetSampleRate sets the audio sample rate
func (d *DSP) SetSampleRate(rate int) {
	d.sampleRate = rate
}

// GetSampleRate returns the current sample rate
func (d *DSP) GetSampleRate() int {
	return d.sampleRate
}

// DecodeBuffer decodes audio samples and calls the callback for each decoded message
// Currently returns mock data - real decoding not yet implemented
func (d *DSP) DecodeBuffer(audioData []int16, callback func(*DecodeResult)) (int, error) {
	if len(audioData) == 0 {
		return 0, fmt.Errorf("empty audio data")
	}

	if callback == nil {
		return 0, fmt.Errorf("callback function required")
	}

	// TODO: Implement real JS8 decoding
	// For now, return mock data to maintain API compatibility

	// Simulate finding a decode in the audio
	if len(audioData) > 100000 { // Only "decode" longer audio buffers
		result := &DecodeResult{
			UTC:       int(time.Now().Unix()),
			SNR:       15,
			DT:        0.2,
			Frequency: 1500.0,
			Message:   "CQ TEST DE N0CALL",
			Type:      0,
			Quality:   0.85,
			Mode:      int(ModeNormal),
		}

		callback(result)
		return 1, nil
	}

	return 0, nil // No messages found
}

// EncodeMessage encodes a text message to audio samples using pure Go
func (d *DSP) EncodeMessage(message string, mode JS8Mode) ([]int16, error) {
	// Validate message length and pad if necessary
	if len(message) == 0 {
		return nil, fmt.Errorf("empty message")
	}

	var paddedMessage string
	var err error

	if len(message) <= 12 {
		// Pad short messages with dashes
		paddedMessage, err = PadMessage(message, '-')
		if err != nil {
			return nil, fmt.Errorf("message padding failed: %w", err)
		}
	} else {
		return nil, fmt.Errorf("message too long (max 12 characters)")
	}

	// Use pure Go encoder
	audio, err := d.encoder.EncodeToAudio(paddedMessage, int(mode), d.sampleRate)
	if err != nil {
		return nil, fmt.Errorf("encoding failed: %w", err)
	}

	return audio, nil
}

// GetError returns the last error message (pure Go - not needed)
func (d *DSP) GetError() string {
	return "" // Pure Go version doesn't maintain global error state
}

// ValidateJS8Message validates that a message contains only valid JS8 characters
func (d *DSP) ValidateJS8Message(message string) error {
	return ValidateMessage(message)
}

// GetJS8Alphabet returns the valid JS8 alphabet
func (d *DSP) GetJS8Alphabet() string {
	return js8Alphabet
}

// EstimateAudioDuration estimates the duration of encoded audio for a given mode
func (d *DSP) EstimateAudioDuration(mode JS8Mode) time.Duration {
	switch mode {
	case ModeNormal:
		return 15 * time.Second
	case ModeFast:
		return 10 * time.Second
	case ModeTurbo:
		return 6 * time.Second
	case ModeSlow:
		return 30 * time.Second
	case ModeUltra:
		return 60 * time.Second
	default:
		return 15 * time.Second
	}
}

// GetToneCount returns the number of tones for a given mode
func (d *DSP) GetToneCount(mode JS8Mode) int {
	switch mode {
	case ModeNormal:
		return 79
	case ModeFast:
		return 40 // Approximate
	case ModeTurbo:
		return 21 // Approximate
	case ModeSlow:
		return 158 // Approximate
	case ModeUltra:
		return 316 // Approximate
	default:
		return 79
	}
}
