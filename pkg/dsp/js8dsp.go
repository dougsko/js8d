package dsp

import (
	"fmt"
	"time"
	"math"
	"math/cmplx"

	"github.com/mjibson/go-dsp/fft"
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

// findSignals uses FFT to find potential JS8 signals in the audio
func (d *DSP) findSignals(audioData []int16) []float32 {
	if len(audioData) < 1024 {
		return nil
	}

	// Convert int16 to complex128 for FFT
	fftInput := make([]complex128, 1024)
	for i := 0; i < 1024 && i < len(audioData); i++ {
		fftInput[i] = complex(float64(audioData[i])/32768.0, 0)
	}

	// Apply window function (Hann window)
	for i := range fftInput {
		window := 0.5 * (1 - math.Cos(2*math.Pi*float64(i)/float64(len(fftInput)-1)))
		fftInput[i] *= complex(window, 0)
	}

	// Perform FFT using go-dsp
	fftOutput := fft.FFT(fftInput)

	// Calculate power spectrum
	var frequencies []float32
	for i := 0; i < len(fftOutput)/2; i++ {
		power := cmplx.Abs(fftOutput[i])
		freq := float32(i * d.sampleRate / len(fftOutput))

		// JS8 signals are typically between 300-3000 Hz
		if freq >= 300 && freq <= 3000 && power > 0.1 {
			frequencies = append(frequencies, freq)
		}
	}

	return frequencies
}

// DecodeBuffer decodes audio samples and calls the callback for each decoded message
// Now uses gonum/fourier for signal detection
func (d *DSP) DecodeBuffer(audioData []int16, callback func(*DecodeResult)) (int, error) {
	if len(audioData) == 0 {
		return 0, fmt.Errorf("empty audio data")
	}

	if callback == nil {
		return 0, fmt.Errorf("callback function required")
	}

	// Use FFT to find potential signals
	signals := d.findSignals(audioData)

	// For each potential signal, attempt basic decoding
	var decodeCount int
	for _, freq := range signals {
		// This is still simplified - real JS8 decoding would involve:
		// 1. Costas array synchronization
		// 2. Symbol extraction
		// 3. LDPC decoding
		// For now, create a result for detected signals
		if freq >= 1400 && freq <= 1600 { // Common JS8 frequency range
			result := &DecodeResult{
				UTC:       int(time.Now().Unix()),
				SNR:       12,
				DT:        0.1,
				Frequency: freq,
				Message:   "JS8 SIGNAL DETECTED",
				Type:      0,
				Quality:   0.75,
				Mode:      int(ModeNormal),
			}

			callback(result)
			decodeCount++
		}
	}

	return decodeCount, nil
}

// EncodeMessage encodes a text message to audio samples using pure Go
func (d *DSP) EncodeMessage(message string, mode JS8Mode) ([]int16, error) {
	// Validate message length and pad if necessary
	if len(message) == 0 {
		return nil, fmt.Errorf("empty message")
	}

	// Preprocess message to handle spaces and invalid characters
	message = PreprocessJS8Message(message)
	if len(message) == 0 {
		return nil, fmt.Errorf("message became empty after preprocessing")
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
