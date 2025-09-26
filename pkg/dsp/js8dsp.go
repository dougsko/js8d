package dsp

/*
#cgo CFLAGS: -I${SRCDIR}/../../libjs8dsp
#cgo LDFLAGS: -L${SRCDIR}/../../libjs8dsp/build -ljs8dsp -lm -lstdc++

#include <stdlib.h>
#include "js8dsp.h"

// Callback wrapper for Go
extern void goDecodeCallback(js8dsp_decode_t* decode, void* user_data);

// C wrapper function that calls the Go callback
static void c_decode_callback(const js8dsp_decode_t* decode, void* user_data) {
    // Cast away const for the callback (Go will treat it as read-only)
    goDecodeCallback((js8dsp_decode_t*)decode, user_data);
}

// Helper function to call decode with C callback
static int decode_with_callback(const int16_t* audio_data, int samples, void* user_data) {
    return js8dsp_decode_buffer(audio_data, samples, c_decode_callback, user_data);
}
*/
import "C"

import (
	"fmt"
	"runtime"
	"unsafe"
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

// DSP represents the JS8 DSP engine
type DSP struct {
	initialized bool
}

// Global callback storage for CGO
var decodeCallbacks = make(map[uintptr]func(*DecodeResult))
var callbackCounter uintptr = 1

// NewDSP creates a new DSP instance
func NewDSP() *DSP {
	return &DSP{}
}

// Initialize initializes the DSP library
func (d *DSP) Initialize() error {
	if d.initialized {
		return nil
	}

	result := C.js8dsp_init()
	if result != 0 {
		errorMsg := C.GoString(C.js8dsp_get_error())
		return fmt.Errorf("failed to initialize JS8DSP: %s", errorMsg)
	}

	d.initialized = true
	runtime.SetFinalizer(d, (*DSP).cleanup)
	return nil
}

// cleanup is called by the finalizer
func (d *DSP) cleanup() {
	if d.initialized {
		C.js8dsp_cleanup()
		d.initialized = false
	}
}

// Close manually cleans up the DSP library
func (d *DSP) Close() {
	d.cleanup()
	runtime.SetFinalizer(d, nil)
}

// DecodeBuffer decodes audio samples and calls the callback for each decoded message
func (d *DSP) DecodeBuffer(audioData []int16, callback func(*DecodeResult)) (int, error) {
	if !d.initialized {
		return 0, fmt.Errorf("DSP not initialized")
	}

	if len(audioData) == 0 {
		return 0, fmt.Errorf("empty audio data")
	}

	// Store callback with unique ID
	callbackID := callbackCounter
	callbackCounter++
	decodeCallbacks[callbackID] = callback

	// Ensure cleanup of callback
	defer delete(decodeCallbacks, callbackID)

	// Call C function
	result := C.decode_with_callback(
		(*C.int16_t)(unsafe.Pointer(&audioData[0])),
		C.int(len(audioData)),
		unsafe.Pointer(callbackID),
	)

	if result < 0 {
		errorMsg := C.GoString(C.js8dsp_get_error())
		return 0, fmt.Errorf("decode failed: %s", errorMsg)
	}

	return int(result), nil
}

// EncodeMessage encodes a text message to audio samples
func (d *DSP) EncodeMessage(message string, mode JS8Mode) ([]int16, error) {
	if !d.initialized {
		return nil, fmt.Errorf("DSP not initialized")
	}

	// Allocate output buffer
	maxSamples := 180000 // 15 seconds at 12kHz
	audioOut := make([]int16, maxSamples)

	cMessage := C.CString(message)
	defer C.free(unsafe.Pointer(cMessage))

	result := C.js8dsp_encode_message(
		cMessage,
		C.js8dsp_mode_t(mode),
		(*C.int16_t)(unsafe.Pointer(&audioOut[0])),
		C.int(maxSamples),
	)

	if result < 0 {
		errorMsg := C.GoString(C.js8dsp_get_error())
		return nil, fmt.Errorf("encode failed: %s", errorMsg)
	}

	// Return slice with actual length
	return audioOut[:result], nil
}

// GetError returns the last error message from the DSP library
func (d *DSP) GetError() string {
	return C.GoString(C.js8dsp_get_error())
}

//export goDecodeCallback
func goDecodeCallback(decode *C.js8dsp_decode_t, userData unsafe.Pointer) {
	callbackID := uintptr(userData)
	callback, exists := decodeCallbacks[callbackID]
	if !exists {
		return
	}

	// Convert C struct to Go struct
	result := &DecodeResult{
		UTC:       int(decode.utc),
		SNR:       int(decode.snr),
		DT:        float32(decode.dt),
		Frequency: float32(decode.frequency),
		Message:   C.GoString(&decode.message[0]),
		Type:      int(decode.msg_type),
		Quality:   float32(decode.quality),
		Mode:      int(decode.mode),
	}

	callback(result)
}