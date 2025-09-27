package dsp

/*
#cgo CFLAGS: -I/home/doug/bin/js8d/libjs8dsp/include
#cgo LDFLAGS: -L/home/doug/bin/js8d/libjs8dsp/build -ljs8dsp -lstdc++ -lm
#include "js8dsp.h"
#include <stdlib.h>
*/
import "C"
import (
	"fmt"
	"time"
	"unsafe"
)

// CppDSP represents the C++ DSP engine with real JS8Call algorithms
type CppDSP struct {
	handle     C.js8dsp_handle_t
	sampleRate int
}

// NewCppDSP creates a new C++ DSP instance
func NewCppDSP() *CppDSP {
	return &CppDSP{
		sampleRate: 48000, // Default to 48kHz
	}
}

// Initialize initializes the C++ DSP library
func (d *CppDSP) Initialize() error {
	d.handle = C.js8dsp_init(C.int(d.sampleRate), C.JS8DSP_MODE_NORMAL)
	if d.handle == nil {
		return fmt.Errorf("failed to initialize JS8DSP library")
	}
	return nil
}

// Close cleans up the C++ DSP library
func (d *CppDSP) Close() {
	if d.handle != nil {
		C.js8dsp_cleanup(d.handle)
		d.handle = nil
	}
}

// SetSampleRate sets the audio sample rate
func (d *CppDSP) SetSampleRate(rate int) {
	d.sampleRate = rate
	// If already initialized, need to reinitialize with new rate
	if d.handle != nil {
		d.Close()
		d.Initialize()
	}
}

// GetSampleRate returns the current sample rate
func (d *CppDSP) GetSampleRate() int {
	return d.sampleRate
}

// DecodeBuffer decodes audio samples and calls the callback for each decoded message
func (d *CppDSP) DecodeBuffer(audioData []int16, callback func(*DecodeResult)) (int, error) {
	if d.handle == nil {
		return 0, fmt.Errorf("DSP not initialized")
	}

	if len(audioData) == 0 {
		return 0, fmt.Errorf("empty audio data")
	}

	if callback == nil {
		return 0, fmt.Errorf("callback function required")
	}

	// Convert int16 to float32 for C++ API
	floatData := make([]float32, len(audioData))
	for i, sample := range audioData {
		floatData[i] = float32(sample) / 32768.0
	}

	// Prepare output buffer for decoded messages
	const maxMessages = 10
	messages := make([]C.js8dsp_decoded_message_t, maxMessages)

	// Call C++ decode function
	decodeCount := C.js8dsp_decode_buffer(
		d.handle,
		(*C.float)(unsafe.Pointer(&floatData[0])),
		C.size_t(len(floatData)),
		&messages[0],
		C.int(maxMessages),
	)

	if decodeCount < 0 {
		errorMsg := C.js8dsp_get_error(d.handle)
		if errorMsg != nil {
			return 0, fmt.Errorf("decode error: %s", C.GoString(errorMsg))
		}
		return 0, fmt.Errorf("decode failed with code %d", int(decodeCount))
	}

	// Convert C messages to Go and call callback
	for i := 0; i < int(decodeCount); i++ {
		msg := &messages[i]
		result := &DecodeResult{
			UTC:       int(time.Now().Unix()),
			SNR:       int(msg.snr),
			DT:        0.0, // Not used in current implementation
			Frequency: float32(msg.freq_offset),
			Message:   C.GoString(&msg.message[0]),
			Type:      0,
			Quality:   float32(msg.confidence) / 100.0,
			Mode:      int(ModeNormal),
		}
		callback(result)
	}

	return int(decodeCount), nil
}

// EncodeMessage encodes a text message to audio samples using C++ DSP
func (d *CppDSP) EncodeMessage(message string, mode JS8Mode) ([]int16, error) {
	if d.handle == nil {
		return nil, fmt.Errorf("DSP not initialized")
	}

	if len(message) == 0 {
		return nil, fmt.Errorf("empty message")
	}

	// Get required buffer size
	cMessage := C.CString(message)
	defer C.free(unsafe.Pointer(cMessage))

	bufferSize := C.js8dsp_get_encode_buffer_size(d.handle, cMessage)
	if bufferSize < 0 {
		errorMsg := C.js8dsp_get_error(d.handle)
		if errorMsg != nil {
			return nil, fmt.Errorf("encode buffer size error: %s", C.GoString(errorMsg))
		}
		return nil, fmt.Errorf("failed to get encode buffer size: %d", int(bufferSize))
	}

	// Allocate output buffer
	floatBuffer := make([]float32, int(bufferSize))

	// Encode message
	samplesGenerated := C.js8dsp_encode_message(
		d.handle,
		cMessage,
		(*C.float)(unsafe.Pointer(&floatBuffer[0])),
		C.size_t(len(floatBuffer)),
	)

	if samplesGenerated < 0 {
		errorMsg := C.js8dsp_get_error(d.handle)
		if errorMsg != nil {
			return nil, fmt.Errorf("encode error: %s", C.GoString(errorMsg))
		}
		return nil, fmt.Errorf("encode failed with code %d", int(samplesGenerated))
	}

	// Convert float32 to int16
	audioSamples := make([]int16, int(samplesGenerated))
	for i := 0; i < int(samplesGenerated); i++ {
		// Scale and clamp to int16 range
		sample := floatBuffer[i] * 32767.0
		if sample > 32767.0 {
			sample = 32767.0
		} else if sample < -32768.0 {
			sample = -32768.0
		}
		audioSamples[i] = int16(sample)
	}

	return audioSamples, nil
}

// GetError returns the last error message from the C++ library
func (d *CppDSP) GetError() string {
	if d.handle == nil {
		return "DSP not initialized"
	}

	errorMsg := C.js8dsp_get_error(d.handle)
	if errorMsg == nil {
		return ""
	}
	return C.GoString(errorMsg)
}

// ValidateJS8Message validates that a message contains only valid JS8 characters
func (d *CppDSP) ValidateJS8Message(message string) error {
	return ValidateMessage(message)
}

// GetJS8Alphabet returns the valid JS8 alphabet
func (d *CppDSP) GetJS8Alphabet() string {
	return js8Alphabet
}

// EstimateAudioDuration estimates the duration of encoded audio for a given mode
func (d *CppDSP) EstimateAudioDuration(mode JS8Mode) time.Duration {
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
func (d *CppDSP) GetToneCount(mode JS8Mode) int {
	switch mode {
	case ModeNormal:
		return 79
	case ModeFast:
		return 40
	case ModeTurbo:
		return 21
	case ModeSlow:
		return 158
	case ModeUltra:
		return 316
	default:
		return 79
	}
}

// GetStats returns decoder statistics
func (d *CppDSP) GetStats() (totalDecoded, totalErrors uint32, err error) {
	if d.handle == nil {
		return 0, 0, fmt.Errorf("DSP not initialized")
	}

	var decoded, errors C.uint32_t
	result := C.js8dsp_get_stats(d.handle, &decoded, &errors)

	if result != C.JS8DSP_OK {
		return 0, 0, fmt.Errorf("failed to get stats")
	}

	return uint32(decoded), uint32(errors), nil
}