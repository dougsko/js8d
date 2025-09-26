package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v2"
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
	c.JSON(http.StatusOK, d.config)
}

// handleSaveConfig saves the configuration to file
func (d *JS8Daemon) handleSaveConfig(c *gin.Context) {
	var newConfig map[string]interface{}
	if err := c.ShouldBindJSON(&newConfig); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Convert to YAML and save to file
	yamlData, err := yaml.Marshal(newConfig)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("failed to marshal config: %v", err),
		})
		return
	}

	// Determine config file path (use the one passed to daemon or default)
	configPath := "/tmp/claude/test_config.yaml" // Default for now
	if len(os.Args) > 2 && os.Args[1] == "-config" {
		configPath = os.Args[2]
	}

	// Write to file
	if err := ioutil.WriteFile(configPath, yamlData, 0644); err != nil {
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
