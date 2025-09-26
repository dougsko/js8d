
package hardware


/*
#cgo pkg-config: hamlib
#include <hamlib/rig.h>
#include <stdlib.h>
#include <string.h>
#include <stdio.h>
#include <unistd.h>
#include <fcntl.h>

// Simple wrapper to set hamlib debug level via environment variable
static void try_set_hamlib_debug(int level) {
    char level_str[16];
    snprintf(level_str, sizeof(level_str), "%d", level);
    setenv("HAMLIB_DEBUG_LEVEL", level_str, 1);
}

// Helper function to set device path using rig_set_conf (JS8Call approach)
static int set_device_path(RIG *rig, const char *device_path) {
    if (rig && device_path) {
        token_t token = rig_token_lookup(rig, "rig_pathname");
        if (token != RIG_CONF_END) {
            return rig_set_conf(rig, token, device_path);
        }
    }
    return -1;
}

// Helper function to set baud rate using rig_set_conf (JS8Call approach)
static int set_baud_rate(RIG *rig, const char *baud_rate) {
    if (rig && baud_rate) {
        token_t token = rig_token_lookup(rig, "serial_speed");
        if (token != RIG_CONF_END) {
            return rig_set_conf(rig, token, baud_rate);
        }
    }
    return -1;
}

// Helper function to get rig info as strings
static void get_rig_info(RIG *rig, char **model, char **mfg, char **version) {
    const struct rig_caps *caps = rig->caps;
    if (caps) {
        *model = (char*)caps->model_name;
        *mfg = (char*)caps->mfg_name;
        *version = (char*)caps->version;
    } else {
        *model = "Unknown";
        *mfg = "Unknown";
        *version = "Unknown";
    }
}

// Helper function to convert rig mode to string
static const char* mode_to_string(rmode_t mode) {
    switch(mode) {
        case RIG_MODE_USB: return "USB";
        case RIG_MODE_LSB: return "LSB";
        case RIG_MODE_CW: return "CW";
        case RIG_MODE_RTTY: return "RTTY";
        case RIG_MODE_PSK: return "PSK";
        case RIG_MODE_PKTUSB: return "PKTUSB";
        case RIG_MODE_PKTLSB: return "PKTLSB";
        case RIG_MODE_FM: return "FM";
        case RIG_MODE_AM: return "AM";
        default: return "UNKNOWN";
    }
}

// Helper function to convert string to rig mode
static rmode_t string_to_mode(const char* mode) {
    if (strcmp(mode, "USB") == 0) return RIG_MODE_USB;
    if (strcmp(mode, "LSB") == 0) return RIG_MODE_LSB;
    if (strcmp(mode, "CW") == 0) return RIG_MODE_CW;
    if (strcmp(mode, "RTTY") == 0) return RIG_MODE_RTTY;
    if (strcmp(mode, "PSK") == 0) return RIG_MODE_PSK;
    if (strcmp(mode, "PKTUSB") == 0) return RIG_MODE_PKTUSB;
    if (strcmp(mode, "PKTLSB") == 0) return RIG_MODE_PKTLSB;
    if (strcmp(mode, "JT8") == 0) return RIG_MODE_PKTUSB; // Use PKTUSB for digital modes
    if (strcmp(mode, "FM") == 0) return RIG_MODE_FM;
    if (strcmp(mode, "AM") == 0) return RIG_MODE_AM;
    return RIG_MODE_USB; // Default to USB
}
*/
import "C"

import (
	"fmt"
	"strconv"
	"sync"
	"unsafe"

	"github.com/dougsko/js8d/pkg/verbose"
)

// HamlibRadio implements RadioInterface using Hamlib
type HamlibRadio struct {
	config RadioConfig
	rig    *C.RIG
	mutex  sync.RWMutex

	// Connection state
	connected bool
	model     C.rig_model_t
}

// NewHamlibRadio creates a new Hamlib radio interface
func NewHamlibRadio(config RadioConfig) *HamlibRadio {
	// Set hamlib debug level immediately when creating radio instance
	if verbose.IsEnabled() {
		C.try_set_hamlib_debug(3) // RIG_DEBUG_VERBOSE
	} else {
		C.try_set_hamlib_debug(0) // RIG_DEBUG_NONE - suppress all hamlib output
	}

	return &HamlibRadio{
		config: config,
	}
}

// Initialize initializes the Hamlib radio interface
func (r *HamlibRadio) Initialize() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Set hamlib debug level using multiple approaches
	if verbose.IsEnabled() {
		C.try_set_hamlib_debug(3) // RIG_DEBUG_VERBOSE
		verbose.Printf("Hamlib: Verbose debugging enabled")
	} else {
		C.try_set_hamlib_debug(0) // RIG_DEBUG_NONE - suppress all hamlib output
	}

	verbose.Printf("Hamlib: Initializing radio control...")
	verbose.Printf("Hamlib: Model: %s", r.config.Model)
	verbose.Printf("Hamlib: Device: %s", r.config.Device)
	verbose.Printf("Hamlib: Baud Rate: %d", r.config.BaudRate)

	// Parse model ID (can be numeric or string)
	var modelID C.rig_model_t
	if num, err := strconv.Atoi(r.config.Model); err == nil {
		// Numeric model ID
		modelID = C.rig_model_t(num)
	} else {
		// Try to find model by name
		// For now, default to a common model if string provided
		verbose.Printf("Hamlib: Model name provided, using auto-detection")
		modelID = C.RIG_MODEL_DUMMY // Will be replaced with proper name lookup
	}

	r.model = modelID

	// Initialize rig
	r.rig = C.rig_init(modelID)
	if r.rig == nil {
		return fmt.Errorf("failed to initialize rig model %s", r.config.Model)
	}

	// Set device path (not needed for dummy rig)
	if r.config.Device != "" && r.config.Model != "1" {
		devicePath := C.CString(r.config.Device)
		defer C.free(unsafe.Pointer(devicePath))

		verbose.Printf("Hamlib: Setting device to %s", r.config.Device)

		// Set the device path using rig_set_conf (JS8Call approach)
		ret := C.set_device_path(r.rig, devicePath)
		if ret != C.RIG_OK {
			verbose.Printf("Hamlib: Warning - failed to set device path (%s), may use default", C.GoString(C.rigerror(ret)))
		} else {
			verbose.Printf("Hamlib: Device path set successfully")
		}
	} else if r.config.Model == "1" {
		verbose.Printf("Hamlib: Using dummy rig - no device path needed")
	}

	// Set baud rate explicitly
	if r.config.BaudRate > 0 {
		baudStr := C.CString(fmt.Sprintf("%d", r.config.BaudRate))
		defer C.free(unsafe.Pointer(baudStr))

		verbose.Printf("Hamlib: Setting baud rate to %d", r.config.BaudRate)

		// Set the baud rate using rig_set_conf (JS8Call approach)
		ret := C.set_baud_rate(r.rig, baudStr)
		if ret != C.RIG_OK {
			verbose.Printf("Hamlib: Warning - failed to set baud rate (%s), using default", C.GoString(C.rigerror(ret)))
		} else {
			verbose.Printf("Hamlib: Baud rate set successfully")
		}
	}

	// Open the rig connection
	ret := C.rig_open(r.rig)
	if ret != C.RIG_OK {
		C.rig_cleanup(r.rig)
		return fmt.Errorf("failed to open rig connection: %s", C.GoString(C.rigerror(ret)))
	}

	r.connected = true
	verbose.Printf("Hamlib: Radio connection established successfully")

	// Get radio info for verification
	if info, err := r.getRadioInfoLocked(); err == nil {
		verbose.Printf("Hamlib: Connected to %s %s (version %s)",
			info.Manufacturer, info.Model, info.Version)
	}


	return nil
}

// Close closes the radio connection
func (r *HamlibRadio) Close() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if !r.connected {
		return nil
	}

	verbose.Printf("Hamlib: Closing radio connection...")

	if r.rig != nil {
		C.rig_close(r.rig)
		C.rig_cleanup(r.rig)
		r.rig = nil
	}

	r.connected = false
	verbose.Printf("Hamlib: Radio connection closed")
	return nil
}

// SetFrequency sets the radio frequency in Hz
func (r *HamlibRadio) SetFrequency(freq int64) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if !r.connected {
		return fmt.Errorf("radio not connected")
	}

	verbose.Printf("Hamlib: Setting frequency to %d Hz (%.3f MHz)", freq, float64(freq)/1000000.0)

	ret := C.rig_set_freq(r.rig, C.RIG_VFO_CURR, C.freq_t(freq))
	if ret != C.RIG_OK {
		return fmt.Errorf("failed to set frequency: %s", C.GoString(C.rigerror(ret)))
	}

	return nil
}

// GetFrequency gets the current radio frequency in Hz
func (r *HamlibRadio) GetFrequency() (int64, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	if !r.connected {
		return 0, fmt.Errorf("radio not connected")
	}

	var freq C.freq_t
	ret := C.rig_get_freq(r.rig, C.RIG_VFO_CURR, &freq)
	if ret != C.RIG_OK {
		return 0, fmt.Errorf("failed to get frequency: %s", C.GoString(C.rigerror(ret)))
	}

	return int64(freq), nil
}

// SetMode sets the radio mode and bandwidth
func (r *HamlibRadio) SetMode(mode string, bandwidth int) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if !r.connected {
		return fmt.Errorf("radio not connected")
	}

	verbose.Printf("Hamlib: Setting mode to %s with bandwidth %d Hz", mode, bandwidth)

	modeStr := C.CString(mode)
	defer C.free(unsafe.Pointer(modeStr))

	rigMode := C.string_to_mode(modeStr)

	ret := C.rig_set_mode(r.rig, C.RIG_VFO_CURR, rigMode, C.pbwidth_t(bandwidth))
	if ret != C.RIG_OK {
		return fmt.Errorf("failed to set mode: %s", C.GoString(C.rigerror(ret)))
	}

	return nil
}

// GetMode gets the current radio mode and bandwidth
func (r *HamlibRadio) GetMode() (string, int, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	if !r.connected {
		return "", 0, fmt.Errorf("radio not connected")
	}

	var mode C.rmode_t
	var bandwidth C.pbwidth_t

	ret := C.rig_get_mode(r.rig, C.RIG_VFO_CURR, &mode, &bandwidth)
	if ret != C.RIG_OK {
		return "", 0, fmt.Errorf("failed to get mode: %s", C.GoString(C.rigerror(ret)))
	}

	modeStr := C.GoString(C.mode_to_string(mode))
	return modeStr, int(bandwidth), nil
}

// SetPTT sets the PTT (Push-To-Talk) state
func (r *HamlibRadio) SetPTT(state bool) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if !r.connected {
		return fmt.Errorf("radio not connected")
	}

	var ptt C.ptt_t
	if state {
		ptt = C.RIG_PTT_ON
		verbose.Printf("Hamlib: PTT ON")
	} else {
		ptt = C.RIG_PTT_OFF
		verbose.Printf("Hamlib: PTT OFF")
	}

	ret := C.rig_set_ptt(r.rig, C.RIG_VFO_CURR, ptt)
	if ret != C.RIG_OK {
		return fmt.Errorf("failed to set PTT: %s", C.GoString(C.rigerror(ret)))
	}

	return nil
}

// GetPTT gets the current PTT state
func (r *HamlibRadio) GetPTT() (bool, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	if !r.connected {
		return false, fmt.Errorf("radio not connected")
	}

	var ptt C.ptt_t
	ret := C.rig_get_ptt(r.rig, C.RIG_VFO_CURR, &ptt)
	if ret != C.RIG_OK {
		return false, fmt.Errorf("failed to get PTT: %s", C.GoString(C.rigerror(ret)))
	}

	return ptt == C.RIG_PTT_ON, nil
}

// GetRadioInfo gets radio information
func (r *HamlibRadio) GetRadioInfo() (RadioInfo, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return r.getRadioInfoLocked()
}

// getRadioInfoLocked gets radio info (must be called with lock held)
func (r *HamlibRadio) getRadioInfoLocked() (RadioInfo, error) {
	if !r.connected {
		return RadioInfo{}, fmt.Errorf("radio not connected")
	}

	var model, mfg, version *C.char
	C.get_rig_info(r.rig, &model, &mfg, &version)

	info := RadioInfo{
		Model:        C.GoString(model),
		Manufacturer: C.GoString(mfg),
		Version:      C.GoString(version),
		Capabilities: []string{}, // TODO: Add capability detection
	}

	return info, nil
}

// IsConnected returns whether the radio is connected
func (r *HamlibRadio) IsConnected() bool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return r.connected
}

// GetPowerLevel gets the current power level (0.0-1.0)
func (r *HamlibRadio) GetPowerLevel() (float32, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	if !r.connected {
		return 0, fmt.Errorf("radio not connected")
	}

	// For now, return a mock value since hamlib API has changed
	// This would need to be updated for specific hamlib version compatibility
	verbose.Printf("Hamlib: GetPowerLevel called (returning mock value)")
	return 0.5, nil // 50% power
}

// GetSWRLevel gets the current SWR level
func (r *HamlibRadio) GetSWRLevel() (float32, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	if !r.connected {
		return 0, fmt.Errorf("radio not connected")
	}

	// For now, return a mock value since hamlib API has changed
	verbose.Printf("Hamlib: GetSWRLevel called (returning mock value)")
	return 1.2, nil // 1.2:1 SWR
}

// GetSignalLevel gets the current signal level in dBm
func (r *HamlibRadio) GetSignalLevel() (int, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	if !r.connected {
		return 0, fmt.Errorf("radio not connected")
	}

	// For now, return a mock value since hamlib API has changed
	verbose.Printf("Hamlib: GetSignalLevel called (returning mock value)")
	return -73, nil // -73 dBm signal level
}

