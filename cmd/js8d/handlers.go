package main

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
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