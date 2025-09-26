package audio

import (
	"math"
	"sync"
	"time"

	"github.com/mjibson/go-dsp/fft"
)

// AudioLevelData represents real-time audio level measurements
type AudioLevelData struct {
	Timestamp int64   `json:"timestamp"`
	RMSLevel  float32 `json:"rms"`      // RMS level in dB
	PeakLevel float32 `json:"peak"`     // Peak level in dB
	Clipping  bool    `json:"clipping"` // True if clipping detected
}

// SpectrumData represents FFT spectrum analysis
type SpectrumData struct {
	Timestamp  int64     `json:"timestamp"`
	SampleRate int       `json:"sample_rate"`
	Spectrum   []float32 `json:"spectrum"` // Magnitude spectrum in dB
	FreqStep   float32   `json:"freq_step"` // Frequency per bin in Hz
}

// AudioVisualizationData combines level and spectrum data
type AudioVisualizationData struct {
	AudioLevelData
	SpectrumData
}

// AudioLevelMonitor processes audio samples for real-time visualization
type AudioLevelMonitor struct {
	mutex sync.RWMutex

	// Configuration
	sampleRate   int
	fftSize      int
	updateRate   time.Duration

	// Current measurements
	currentRMS   float32
	currentPeak  float32
	peakHold     float32
	peakHoldTime time.Time
	isClipping   bool

	// Spectrum analysis
	spectrum     []float32
	spectrumTime time.Time

	// Buffers
	sampleBuffer []int16
	fftBuffer    []complex128
	window       []float64

	// Statistics
	sampleCount  int64
	clipCount    int64

	// Control
	running bool
	stopChan chan struct{}
}

// NewAudioLevelMonitor creates a new audio level monitor
func NewAudioLevelMonitor(sampleRate, fftSize int) *AudioLevelMonitor {
	monitor := &AudioLevelMonitor{
		sampleRate: sampleRate,
		fftSize:    fftSize,
		updateRate: 50 * time.Millisecond, // 20Hz update rate
		spectrum:   make([]float32, fftSize/2),
		fftBuffer:  make([]complex128, fftSize),
		window:     makeHannWindow(fftSize),
		stopChan:   make(chan struct{}),
	}

	return monitor
}

// makeHannWindow creates a Hann window function for FFT
func makeHannWindow(size int) []float64 {
	window := make([]float64, size)
	for i := 0; i < size; i++ {
		window[i] = 0.5 * (1.0 - math.Cos(2.0*math.Pi*float64(i)/float64(size-1)))
	}
	return window
}

// ProcessSamples processes a buffer of audio samples
func (m *AudioLevelMonitor) ProcessSamples(samples []int16) {
	if len(samples) == 0 {
		return
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Calculate RMS and peak levels
	m.calculateLevels(samples)

	// Update spectrum if we have enough samples
	m.sampleBuffer = append(m.sampleBuffer, samples...)
	if len(m.sampleBuffer) >= m.fftSize {
		m.calculateSpectrum()
		// Keep only the newest samples
		if len(m.sampleBuffer) > m.fftSize {
			copy(m.sampleBuffer, m.sampleBuffer[len(m.sampleBuffer)-m.fftSize:])
			m.sampleBuffer = m.sampleBuffer[:m.fftSize]
		}
	}

	m.sampleCount += int64(len(samples))
}

// calculateLevels computes RMS and peak levels from samples
func (m *AudioLevelMonitor) calculateLevels(samples []int16) {
	if len(samples) == 0 {
		return
	}

	var sumSquares float64
	var peak int16
	clipping := false

	for _, sample := range samples {
		// Track peak
		if sample < 0 {
			sample = -sample
		}
		if sample > peak {
			peak = sample
		}

		// Check for clipping (close to max value)
		if sample >= 32000 { // ~98% of max int16
			clipping = true
			m.clipCount++
		}

		// Sum for RMS calculation
		sumSquares += float64(sample) * float64(sample)
	}

	// Calculate RMS in dB
	rms := math.Sqrt(sumSquares / float64(len(samples)))
	if rms > 0 {
		m.currentRMS = float32(20.0 * math.Log10(rms/32768.0))
	} else {
		m.currentRMS = -100.0 // Very quiet
	}

	// Calculate peak in dB
	if peak > 0 {
		peakDB := float32(20.0 * math.Log10(float64(peak)/32768.0))
		m.currentPeak = peakDB

		// Update peak hold
		now := time.Now()
		if peakDB > m.peakHold || now.Sub(m.peakHoldTime) > 2*time.Second {
			m.peakHold = peakDB
			m.peakHoldTime = now
		}
	} else {
		m.currentPeak = -100.0
	}

	m.isClipping = clipping
}

// calculateSpectrum performs FFT analysis on accumulated samples
func (m *AudioLevelMonitor) calculateSpectrum() {
	if len(m.sampleBuffer) < m.fftSize {
		return
	}

	// Convert samples to complex with windowing
	for i := 0; i < m.fftSize; i++ {
		sample := float64(m.sampleBuffer[i]) / 32768.0 // Normalize to [-1, 1]
		windowed := sample * m.window[i]
		m.fftBuffer[i] = complex(windowed, 0)
	}

	// Perform FFT
	fftResult := fft.FFT(m.fftBuffer)

	// Calculate magnitude spectrum (only positive frequencies)
	for i := 0; i < len(m.spectrum); i++ {
		magnitude := math.Sqrt(real(fftResult[i])*real(fftResult[i]) +
							   imag(fftResult[i])*imag(fftResult[i]))

		// Convert to dB
		if magnitude > 0 {
			m.spectrum[i] = float32(20.0 * math.Log10(magnitude))
		} else {
			m.spectrum[i] = -100.0
		}
	}

	m.spectrumTime = time.Now()
}

// GetCurrentLevels returns the current audio levels
func (m *AudioLevelMonitor) GetCurrentLevels() AudioLevelData {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return AudioLevelData{
		Timestamp: time.Now().UnixMilli(),
		RMSLevel:  m.currentRMS,
		PeakLevel: m.currentPeak,
		Clipping:  m.isClipping,
	}
}

// GetCurrentSpectrum returns the current spectrum data
func (m *AudioLevelMonitor) GetCurrentSpectrum() SpectrumData {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// Copy spectrum to avoid race conditions
	spectrum := make([]float32, len(m.spectrum))
	copy(spectrum, m.spectrum)

	freqStep := float32(m.sampleRate) / float32(m.fftSize)

	return SpectrumData{
		Timestamp:  m.spectrumTime.UnixMilli(),
		SampleRate: m.sampleRate,
		Spectrum:   spectrum,
		FreqStep:   freqStep,
	}
}

// GetVisualizationData returns combined audio data for visualization
func (m *AudioLevelMonitor) GetVisualizationData() AudioVisualizationData {
	levels := m.GetCurrentLevels()
	spectrum := m.GetCurrentSpectrum()

	return AudioVisualizationData{
		AudioLevelData: levels,
		SpectrumData:   spectrum,
	}
}

// GetStatistics returns monitoring statistics
func (m *AudioLevelMonitor) GetStatistics() map[string]interface{} {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	clipRate := float64(0)
	if m.sampleCount > 0 {
		clipRate = float64(m.clipCount) / float64(m.sampleCount) * 100.0
	}

	return map[string]interface{}{
		"sample_count":    m.sampleCount,
		"clip_count":      m.clipCount,
		"clip_rate_pct":   clipRate,
		"peak_hold_db":    m.peakHold,
		"sample_rate":     m.sampleRate,
		"fft_size":        m.fftSize,
		"buffer_samples":  len(m.sampleBuffer),
	}
}

// Start begins monitoring (currently just marks as running)
func (m *AudioLevelMonitor) Start() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.running = true
	return nil
}

// Stop stops monitoring
func (m *AudioLevelMonitor) Stop() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.running = false
	close(m.stopChan)
}

// IsRunning returns whether monitoring is active
func (m *AudioLevelMonitor) IsRunning() bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.running
}