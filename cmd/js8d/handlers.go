package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"gopkg.in/yaml.v2"

	"github.com/dougsko/js8d/pkg/hardware"
)

// handleHome serves the main web interface
func (d *JS8Daemon) handleHome(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", gin.H{
		"callsign": d.config.Station.Callsign,
		"grid":     d.config.Station.Grid,
		"version":  Version,
	})
}

// handleGetStatus returns daemon status via socket
func (d *JS8Daemon) handleGetStatus(c *gin.Context) {
	status, err := d.socketClient.GetStatus()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    "running",
		"version":   Version,
		"callsign":  status.Callsign,
		"grid":      status.Grid,
		"uptime":    status.Uptime,
		"frequency": status.Frequency,
		"mode":      status.Mode,
		"ptt":       status.PTT,
		"connected": status.Connected,
	})
}

// handleGetMessages returns recent messages via socket
func (d *JS8Daemon) handleGetMessages(c *gin.Context) {
	// Parse limit parameter
	limitStr := c.DefaultQuery("limit", "50")
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 50
	}

	messages, err := d.socketClient.GetMessages(limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"messages": messages,
		"count":    len(messages),
	})
}

// handleSendMessage queues a message for transmission via socket
func (d *JS8Daemon) handleSendMessage(c *gin.Context) {
	var req struct {
		To      string `json:"to"`
		Message string `json:"message" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	message, err := d.socketClient.SendMessage(req.To, req.Message)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "queued",
		"message": message,
	})
}

// handleGetRadio returns radio status via socket
func (d *JS8Daemon) handleGetRadio(c *gin.Context) {
	radioStatus, err := d.socketClient.GetRadioStatus()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, radioStatus)
}

// handleSetFrequency sets the radio frequency via socket
func (d *JS8Daemon) handleSetFrequency(c *gin.Context) {
	var req struct {
		Frequency int `json:"frequency" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := d.socketClient.SetFrequency(req.Frequency); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"frequency": req.Frequency,
	})
}

// handleAbortTransmission aborts any ongoing transmission and turns off PTT
func (d *JS8Daemon) handleAbortTransmission(c *gin.Context) {
	if err := d.socketClient.AbortTransmission(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "aborted",
	})
}

// handleSettings serves the settings page
func (d *JS8Daemon) handleSettings(c *gin.Context) {
	c.HTML(http.StatusOK, "settings.html", gin.H{
		"version": Version,
	})
}

// handleGetConfig returns the current configuration
func (d *JS8Daemon) handleGetConfig(c *gin.Context) {
	// Marshal to YAML then unmarshal to JSON via map to ensure
	// field names match the YAML structure and JSON compatibility
	yamlData, err := yaml.Marshal(d.config)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("failed to marshal config: %v", err),
		})
		return
	}

	// Unmarshal YAML to interface{} then convert to JSON-compatible map
	var yamlConfig interface{}
	if err := yaml.Unmarshal(yamlData, &yamlConfig); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("failed to unmarshal config: %v", err),
		})
		return
	}

	// Convert map[interface{}]interface{} to map[string]interface{} recursively
	configMap := convertYamlToJson(yamlConfig)

	c.JSON(http.StatusOK, configMap)
}

// convertYamlToJson converts YAML map[interface{}]interface{} to JSON-compatible map[string]interface{}
func convertYamlToJson(i interface{}) interface{} {
	switch x := i.(type) {
	case map[interface{}]interface{}:
		m2 := map[string]interface{}{}
		for k, v := range x {
			m2[k.(string)] = convertYamlToJson(v)
		}
		return m2
	case []interface{}:
		for i, v := range x {
			x[i] = convertYamlToJson(v)
		}
	}
	return i
}

// deepMerge recursively merges source map into destination map
func deepMerge(dst, src map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	// First copy all dst values
	for k, v := range dst {
		result[k] = v
	}

	// Then merge src values
	for k, v := range src {
		if srcMap, srcOk := v.(map[string]interface{}); srcOk {
			if dstMap, dstOk := result[k].(map[string]interface{}); dstOk {
				// Both are maps, merge recursively
				result[k] = deepMerge(dstMap, srcMap)
			} else {
				// Destination is not a map, replace with source
				result[k] = v
			}
		} else {
			// Source is not a map, replace destination
			result[k] = v
		}
	}

	return result
}

// handleSaveConfig saves the configuration to file
func (d *JS8Daemon) handleSaveConfig(c *gin.Context) {
	var newConfig map[string]interface{}
	if err := c.ShouldBindJSON(&newConfig); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get current configuration and convert to map format
	yamlData, err := yaml.Marshal(d.config)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("failed to marshal current config: %v", err),
		})
		return
	}

	var currentConfig interface{}
	if err := yaml.Unmarshal(yamlData, &currentConfig); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("failed to unmarshal current config: %v", err),
		})
		return
	}

	// Convert to JSON-compatible format
	currentConfigMap := convertYamlToJson(currentConfig).(map[string]interface{})

	// Merge new configuration into current configuration
	mergedConfig := deepMerge(currentConfigMap, newConfig)

	// Validate audio device configuration if present
	if audioConfig, exists := newConfig["Audio"]; exists {
		if audioMap, ok := audioConfig.(map[string]interface{}); ok {
			warnings := []string{}

			// Validate input device if specified
			if inputDevice, exists := audioMap["InputDevice"]; exists {
				if deviceStr, ok := inputDevice.(string); ok && deviceStr != "" && deviceStr != "default" {
					if err := validateAudioDevice(deviceStr, "input"); err != nil {
						warning := fmt.Sprintf("Input device '%s' validation failed: %v", deviceStr, err)
						log.Printf("Audio validation warning: %s", warning)
						warnings = append(warnings, warning)
					}
				}
			}

			// Validate output device if specified
			if outputDevice, exists := audioMap["OutputDevice"]; exists {
				if deviceStr, ok := outputDevice.(string); ok && deviceStr != "" && deviceStr != "default" {
					if err := validateAudioDevice(deviceStr, "output"); err != nil {
						warning := fmt.Sprintf("Output device '%s' validation failed: %v", deviceStr, err)
						log.Printf("Audio validation warning: %s", warning)
						warnings = append(warnings, warning)
					}
				}
			}

			// Return warnings if any validation failed
			if len(warnings) > 0 {
				c.JSON(http.StatusOK, gin.H{
					"status":   "saved_with_warnings",
					"warnings": warnings,
				})
				// Continue with save despite warnings
			}
		}
	}

	// Convert merged config to YAML and save to file
	yamlData, err = yaml.Marshal(mergedConfig)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("failed to marshal config: %v", err),
		})
		return
	}

	// Determine config file path from daemon's loaded config
	configPath := d.configPath
	if configPath == "" {
		// Fallback if config path not available
		if len(os.Args) > 2 && os.Args[1] == "-config" {
			configPath = os.Args[2]
		} else {
			configPath = "/tmp/claude/config.yaml"
		}
	}

	// Write to file
	if err := os.WriteFile(configPath, yamlData, 0644); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("failed to write config file: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "saved",
		"path":   configPath,
	})
}

// handleReloadConfig triggers daemon to reload configuration
func (d *JS8Daemon) handleReloadConfig(c *gin.Context) {
	// Send reload command to core engine via socket
	resp, err := d.socketClient.SendCommand("RELOAD")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("failed to send reload command: %v", err),
		})
		return
	}

	if !resp.Success {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": resp.Error,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "reloaded",
	})
}

// handleRetryRadioConnection attempts to reconnect the radio after configuration changes
func (d *JS8Daemon) handleRetryRadioConnection(c *gin.Context) {
	// Send retry radio connection command to core engine via socket
	resp, err := d.socketClient.SendCommand("RETRY_RADIO")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("failed to send retry radio command: %v", err),
		})
		return
	}

	if !resp.Success {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": resp.Error,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Radio connection retry completed",
		"details": resp.Data,
	})
}

// handleGetAudioDevices returns available audio devices
func (d *JS8Daemon) handleGetAudioDevices(c *gin.Context) {
	// Try to get real audio devices on macOS and Linux
	if runtime.GOOS == "darwin" || runtime.GOOS == "linux" {
		devices, err := getAvailableAudioDevices()
		if err != nil {
			log.Printf("Warning: Failed to enumerate audio devices: %v", err)
		} else {
			// Separate input and output devices with deduplication
			inputDeviceSet := make(map[string]bool)
			outputDeviceSet := make(map[string]bool)

			for _, device := range devices {
				// Trim any trailing spaces from device names
				deviceName := strings.TrimSpace(device.Name)
				log.Printf("Device: %s (input:%v, output:%v)", deviceName, device.IsInput, device.IsOutput)

				if device.IsInput {
					inputDeviceSet[deviceName] = true
				}
				if device.IsOutput {
					outputDeviceSet[deviceName] = true
				}
			}

			// Convert sets to slices
			inputDevices := make([]string, 0, len(inputDeviceSet))
			for deviceName := range inputDeviceSet {
				inputDevices = append(inputDevices, deviceName)
			}

			outputDevices := make([]string, 0, len(outputDeviceSet))
			for deviceName := range outputDeviceSet {
				outputDevices = append(outputDevices, deviceName)
			}

			log.Printf("Final lists - Input devices: %d, Output devices: %d", len(inputDevices), len(outputDevices))

			c.JSON(http.StatusOK, gin.H{
				"input_devices":  inputDevices,
				"output_devices": outputDevices,
			})
			return
		}
	}

	// Fallback to static device list if enumeration fails or on other platforms
	devices := gin.H{
		"input_devices": []string{
			"Built-in Microphone",
			"USB Audio Device",
			"IC-7300",
			"External Microphone",
		},
		"output_devices": []string{
			"Built-in Output",
			"USB Audio Device",
			"IC-7300",
			"External Speakers",
		},
	}

	c.JSON(http.StatusOK, devices)
}

// getAvailableAudioDevices wraps the audio device enumeration
func getAvailableAudioDevices() ([]AudioDeviceInfo, error) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		return nil, fmt.Errorf("audio device enumeration only supported on macOS and Linux")
	}

	// Call the audio device enumeration from the hardware package
	log.Printf("Attempting to enumerate audio devices...")
	devices, err := hardware.GetAudioDevices()
	if err != nil {
		log.Printf("Audio device enumeration failed: %v", err)
		return nil, fmt.Errorf("failed to enumerate audio devices: %v", err)
	}

	log.Printf("Found %d audio devices", len(devices))

	// Convert hardware.AudioDevice to AudioDeviceInfo
	result := make([]AudioDeviceInfo, len(devices))
	for i, device := range devices {
		result[i] = AudioDeviceInfo{
			ID:       device.ID,
			Name:     device.Name,
			IsInput:  device.IsInput,
			IsOutput: device.IsOutput,
		}
	}

	return result, nil
}

// AudioDeviceInfo represents an audio device for the handler
type AudioDeviceInfo struct {
	ID       uint32 `json:"id"`
	Name     string `json:"name"`
	IsInput  bool   `json:"is_input"`
	IsOutput bool   `json:"is_output"`
}

// validateAudioDevice validates that an audio device can be opened and used
func validateAudioDevice(deviceName, deviceType string) error {
	// For now, we'll do basic validation - check if device file exists on Linux
	if runtime.GOOS == "linux" {
		// ALSA device names like "hw:0,0" or "plughw:0,0"
		if strings.HasPrefix(deviceName, "hw:") || strings.HasPrefix(deviceName, "plughw:") {
			// Parse ALSA device name format hw:card,device
			parts := strings.Split(strings.TrimPrefix(strings.TrimPrefix(deviceName, "plughw:"), "hw:"), ",")
			if len(parts) >= 1 {
				cardNum := parts[0]
				// Check if the ALSA card exists
				cardPath := fmt.Sprintf("/proc/asound/card%s", cardNum)
				if _, err := os.Stat(cardPath); err != nil {
					return fmt.Errorf("ALSA card %s not found", cardNum)
				}

				// Check if device node exists
				devicePath := fmt.Sprintf("/dev/snd/controlC%s", cardNum)
				if _, err := os.Stat(devicePath); err != nil {
					return fmt.Errorf("ALSA device node %s not accessible", devicePath)
				}

				log.Printf("Audio device validation: %s device '%s' appears valid", deviceType, deviceName)
				return nil
			}
		}

		// For other device names, just log a warning
		log.Printf("Audio device validation: Cannot validate non-standard device name '%s'", deviceName)
		return nil
	}

	// On macOS, we could validate against the available devices list
	if runtime.GOOS == "darwin" {
		devices, err := getAvailableAudioDevices()
		if err != nil {
			log.Printf("Audio device validation: Cannot enumerate devices for validation: %v", err)
			return nil // Don't fail validation if we can't enumerate
		}

		// Check if the device name exists in the available devices
		for _, device := range devices {
			if strings.TrimSpace(device.Name) == deviceName {
				// Check if device supports the required direction
				if deviceType == "input" && !device.IsInput {
					return fmt.Errorf("device '%s' does not support input", deviceName)
				}
				if deviceType == "output" && !device.IsOutput {
					return fmt.Errorf("device '%s' does not support output", deviceName)
				}
				log.Printf("Audio device validation: %s device '%s' validated successfully", deviceType, deviceName)
				return nil
			}
		}

		return fmt.Errorf("device '%s' not found in available devices", deviceName)
	}

	// For other platforms, just log and accept
	log.Printf("Audio device validation: Platform-specific validation not implemented for %s", runtime.GOOS)
	return nil
}

// handleGetSerialDevices returns available serial/USB devices
func (d *JS8Daemon) handleGetSerialDevices(c *gin.Context) {
	devices := []string{}

	// Detect serial devices based on operating system
	switch runtime.GOOS {
	case "linux":
		// Check common Linux USB serial device patterns
		patterns := []string{
			"/dev/ttyUSB*",
			"/dev/ttyACM*",
			"/dev/ttyAMA*",
			"/dev/serial/by-id/*",
		}

		for _, pattern := range patterns {
			matches, err := filepath.Glob(pattern)
			if err == nil {
				devices = append(devices, matches...)
			}
		}

	case "darwin":
		// Check common macOS USB serial device patterns
		patterns := []string{
			"/dev/tty.usb*",
			"/dev/tty.SLAB_*", // Silicon Labs CP210x
			"/dev/tty.wchusbserial*", // WCH CH340
			"/dev/tty.usbmodem*", // USB CDC ACM devices
		}

		for _, pattern := range patterns {
			matches, err := filepath.Glob(pattern)
			if err == nil {
				devices = append(devices, matches...)
			}
		}

	case "windows":
		// On Windows, we would typically enumerate COM ports
		// For now, provide common examples
		for i := 1; i <= 20; i++ {
			comPort := fmt.Sprintf("COM%d", i)
			devices = append(devices, comPort)
		}
	}

	// Add some common defaults if no devices found
	if len(devices) == 0 {
		devices = []string{
			"/dev/ttyUSBmodem14201", // Common IC-7300 device
			"/dev/ttyUSB0",
			"/dev/ttyUSB1",
			"/dev/ttyACM0",
			"/dev/ttyACM1",
		}
	}

	// Filter out non-existent devices on Unix systems
	if runtime.GOOS != "windows" {
		filteredDevices := []string{}
		for _, device := range devices {
			if _, err := os.Stat(device); err == nil {
				filteredDevices = append(filteredDevices, device)
			}
		}

		// If filtering removed everything, keep the defaults
		if len(filteredDevices) > 0 {
			devices = filteredDevices
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"serial_devices": devices,
	})
}

// handleTestCAT tests the CAT (Computer Aided Transceiver) connection
func (d *JS8Daemon) handleTestCAT(c *gin.Context) {
	var req struct {
		Device   string `json:"device"`
		Model    string `json:"model"`
		BaudRate int    `json:"baud_rate"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Send CAT test command to daemon via socket
	cmd := fmt.Sprintf("TEST_CAT %s %s %d", req.Device, req.Model, req.BaudRate)
	resp, err := d.socketClient.SendCommand(cmd)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("failed to test CAT: %v", err),
		})
		return
	}

	if !resp.Success {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": resp.Error,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "CAT connection test successful",
		"details": resp.Data,
	})
}

// handleTestPTT tests the PTT (Push To Talk) functionality
func (d *JS8Daemon) handleTestPTT(c *gin.Context) {
	var req struct {
		Method  string  `json:"method"`
		Port    string  `json:"port"`
		TxDelay float64 `json:"tx_delay"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Send PTT test command to daemon via socket
	cmd := fmt.Sprintf("TEST_PTT %s %s %.1f", req.Method, req.Port, req.TxDelay)
	resp, err := d.socketClient.SendCommand(cmd)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("failed to test PTT: %v", err),
		})
		return
	}

	if !resp.Success {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": resp.Error,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "PTT test successful",
		"details": resp.Data,
	})
}

// handleTestPTTOff turns off PTT for toggle mode
func (d *JS8Daemon) handleTestPTTOff(c *gin.Context) {
	// Send PTT off command to daemon via socket
	resp, err := d.socketClient.SendCommand("TEST_PTT_OFF")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("failed to turn off PTT: %v", err),
		})
		return
	}

	if !resp.Success {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": resp.Error,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "PTT turned off successfully",
		"details": resp.Data,
	})
}

// handleGetMessageHistory returns stored message history
func (d *JS8Daemon) handleGetMessageHistory(c *gin.Context) {
	// Parse query parameters
	limitStr := c.DefaultQuery("limit", "50")
	offsetStr := c.DefaultQuery("offset", "0")
	callsign := c.Query("callsign")
	direction := c.Query("direction") // RX, TX, or empty for both
	messageType := c.Query("type")
	unreadOnly := c.Query("unread") == "true"

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 50
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		offset = 0
	}

	// Send message history request to core engine
	cmd := fmt.Sprintf("GET_MESSAGE_HISTORY %d %d %s %s %s %t",
		limit, offset, callsign, direction, messageType, unreadOnly)

	resp, err := d.socketClient.SendCommand(cmd)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("failed to get message history: %v", err),
		})
		return
	}

	if !resp.Success {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": resp.Error,
		})
		return
	}

	c.JSON(http.StatusOK, resp.Data)
}

// handleGetConversations returns conversation summaries
func (d *JS8Daemon) handleGetConversations(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "20")
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 20
	}

	// Send conversations request to core engine
	cmd := fmt.Sprintf("GET_CONVERSATIONS %d", limit)
	resp, err := d.socketClient.SendCommand(cmd)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("failed to get conversations: %v", err),
		})
		return
	}

	if !resp.Success {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": resp.Error,
		})
		return
	}

	c.JSON(http.StatusOK, resp.Data)
}

// handleMarkMessagesRead marks messages as read for a callsign
func (d *JS8Daemon) handleMarkMessagesRead(c *gin.Context) {
	var req struct {
		Callsign string `json:"callsign" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Send mark read command to core engine
	cmd := fmt.Sprintf("MARK_MESSAGES_READ %s", req.Callsign)
	resp, err := d.socketClient.SendCommand(cmd)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("failed to mark messages as read: %v", err),
		})
		return
	}

	if !resp.Success {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": resp.Error,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"message": fmt.Sprintf("Messages marked as read for %s", req.Callsign),
	})
}

// handleSearchMessages searches through message history
func (d *JS8Daemon) handleSearchMessages(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "search query required"})
		return
	}

	limitStr := c.DefaultQuery("limit", "50")
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 50
	}

	// Send search command to core engine
	cmd := fmt.Sprintf("SEARCH_MESSAGES %s %d", query, limit)
	resp, err := d.socketClient.SendCommand(cmd)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("failed to search messages: %v", err),
		})
		return
	}

	if !resp.Success {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": resp.Error,
		})
		return
	}

	c.JSON(http.StatusOK, resp.Data)
}

// handleGetMessageStats returns database statistics
func (d *JS8Daemon) handleGetMessageStats(c *gin.Context) {
	// Send stats request to core engine
	resp, err := d.socketClient.SendCommand("GET_MESSAGE_STATS")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("failed to get message stats: %v", err),
		})
		return
	}

	if !resp.Success {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": resp.Error,
		})
		return
	}

	c.JSON(http.StatusOK, resp.Data)
}


// handleCleanupMessages triggers manual cleanup of old messages
func (d *JS8Daemon) handleCleanupMessages(c *gin.Context) {
	// Send cleanup command to core engine
	resp, err := d.socketClient.SendCommand("CLEANUP_MESSAGES")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("failed to cleanup messages: %v", err),
		})
		return
	}

	if !resp.Success {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": resp.Error,
		})
		return
	}

	c.JSON(http.StatusOK, resp.Data)
}

// WebSocket upgrader
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for development
	},
}

// handleAudioWebSocket handles WebSocket connections for real-time audio data
func (d *JS8Daemon) handleAudioWebSocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	log.Printf("Audio WebSocket client connected")

	// Get audio monitor from core engine
	audioMonitor := d.coreEngine.GetAudioMonitor()
	if audioMonitor == nil {
		log.Printf("Audio monitor not available")
		conn.WriteJSON(map[string]string{
			"error": "audio monitor not available",
		})
		return
	}

	// Send data at 10Hz (100ms intervals) to reduce CPU usage
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	// Handle client messages (for configuration)
	go func() {
		for {
			var msg map[string]interface{}
			if err := conn.ReadJSON(&msg); err != nil {
				log.Printf("WebSocket read error: %v", err)
				return
			}
			// Handle client configuration messages if needed
			log.Printf("WebSocket message received: %v", msg)
		}
	}()

	// Send audio visualization data
	for {
		select {
		case <-ticker.C:
			vizData := audioMonitor.GetVisualizationData()

			// Convert to format expected by JavaScript client
			data := map[string]interface{}{
				"type": "audio_data",
				"timestamp": vizData.SpectrumData.Timestamp,
				"sample_rate": vizData.SpectrumData.SampleRate,
				// VU meter data
				"rms": vizData.AudioLevelData.RMSLevel,
				"peak": vizData.AudioLevelData.PeakLevel,
				"clipping": vizData.AudioLevelData.Clipping,
				// Spectrum data
				"spectrum": map[string]interface{}{
					"bins": vizData.SpectrumData.Spectrum,
					"freq_step": vizData.SpectrumData.FreqStep,
				},
			}

			if err := conn.WriteJSON(data); err != nil {
				log.Printf("WebSocket write error: %v", err)
				return
			}

		case <-d.ctx.Done():
			log.Printf("Audio WebSocket client disconnected (context cancelled)")
			return
		}
	}
}

// handleTestAudioData returns raw audio data for debugging
func (d *JS8Daemon) handleTestAudioData(c *gin.Context) {
	audioMonitor := d.coreEngine.GetAudioMonitor()
	if audioMonitor == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "audio monitor not available",
		})
		return
	}

	// Get current data
	vizData := audioMonitor.GetVisualizationData()

	// Calculate some basic stats
	spectrumSum := float32(0)
	spectrumMax := float32(-100) // Start with very low dB
	spectrumActive := 0

	for _, val := range vizData.SpectrumData.Spectrum {
		spectrumSum += val
		if val > spectrumMax {
			spectrumMax = val
		}
		if val > -80 { // Count bins above -80dB as "active"
			spectrumActive++
		}
	}

	avgSpectrum := float32(0)
	if len(vizData.SpectrumData.Spectrum) > 0 {
		avgSpectrum = spectrumSum / float32(len(vizData.SpectrumData.Spectrum))
	}

	c.JSON(http.StatusOK, gin.H{
		"timestamp": vizData.SpectrumData.Timestamp,
		"sample_rate": vizData.SpectrumData.SampleRate,
		"spectrum_bins": len(vizData.SpectrumData.Spectrum),
		"spectrum_avg_db": avgSpectrum,
		"spectrum_max_db": spectrumMax,
		"spectrum_active_bins": spectrumActive,
		"rms_level": vizData.AudioLevelData.RMSLevel,
		"peak_level": vizData.AudioLevelData.PeakLevel,
		"clipping": vizData.AudioLevelData.Clipping,
		"raw_spectrum_sample": func() []float32 {
			if len(vizData.SpectrumData.Spectrum) > 10 {
				return vizData.SpectrumData.Spectrum[:10]
			}
			return vizData.SpectrumData.Spectrum
		}(), // First 10 bins
	})
}

// handleGetAudioStats returns audio monitoring statistics
func (d *JS8Daemon) handleGetAudioStats(c *gin.Context) {
	// Get audio monitor from core engine
	audioMonitor := d.coreEngine.GetAudioMonitor()
	if audioMonitor == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "audio monitor not available",
		})
		return
	}

	stats := audioMonitor.GetStatistics()
	levels := audioMonitor.GetCurrentLevels()

	response := gin.H{
		"status":     "ok",
		"statistics": stats,
		"current_levels": levels,
		"monitoring": audioMonitor.IsRunning(),
	}

	c.JSON(http.StatusOK, response)
}
