package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/js8call/js8d/pkg/client"
	"github.com/js8call/js8d/pkg/config"
	"github.com/js8call/js8d/pkg/engine"
	"github.com/js8call/js8d/pkg/protocol"
)

// JS8Daemon represents the main daemon with Unix socket architecture
type JS8Daemon struct {
	config *config.Config
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Core components
	coreEngine   *engine.CoreEngine
	socketClient *client.SocketClient
	webServer    *http.Server

	// Socket path
	socketPath string
}

// NewJS8Daemon creates a new daemon instance
func NewJS8Daemon(cfg *config.Config) (*JS8Daemon, error) {
	ctx, cancel := context.WithCancel(context.Background())

	socketPath := cfg.API.UnixSocket
	if socketPath == "" {
		socketPath = "/tmp/js8d.sock"
	}

	daemon := &JS8Daemon{
		config:       cfg,
		ctx:          ctx,
		cancel:       cancel,
		socketPath:   socketPath,
		socketClient: client.NewSocketClient(socketPath),
	}

	// Create core engine
	daemon.coreEngine = engine.NewCoreEngine(cfg, socketPath)

	// Initialize web server
	if err := daemon.setupWebServer(); err != nil {
		return nil, fmt.Errorf("failed to setup web server: %w", err)
	}

	return daemon, nil
}

// Start starts the daemon
func (d *JS8Daemon) Start() error {
	log.Printf("Starting js8d daemon...")

	// Start core engine first
	if err := d.coreEngine.Start(); err != nil {
		return fmt.Errorf("failed to start core engine: %w", err)
	}

	// Wait a moment for socket to be ready
	time.Sleep(100 * time.Millisecond)

	// Test socket connection
	if !d.socketClient.IsConnected() {
		return fmt.Errorf("failed to connect to core engine socket")
	}

	// Start web server
	d.wg.Add(1)
	go func() {
		defer d.wg.Done()
		addr := fmt.Sprintf("%s:%d", d.config.Web.BindAddress, d.config.Web.Port)
		log.Printf("Starting web server on %s", addr)
		if err := d.webServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Web server error: %v", err)
		}
	}()

	// Start OLED driver (if enabled)
	if d.config.Hardware.EnableGPIO {
		d.wg.Add(1)
		go d.oledDriver()
	}

	return nil
}

// Stop stops the daemon gracefully
func (d *JS8Daemon) Stop() error {
	log.Printf("Stopping daemon...")

	d.cancel()

	// Shutdown web server
	if d.webServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := d.webServer.Shutdown(ctx); err != nil {
			log.Printf("Web server shutdown error: %v", err)
		}
	}

	// Stop core engine
	if d.coreEngine != nil {
		if err := d.coreEngine.Stop(); err != nil {
			log.Printf("Core engine shutdown error: %v", err)
		}
	}

	// Wait for goroutines to finish
	d.wg.Wait()

	log.Printf("Daemon stopped")
	return nil
}

// setupWebServer initializes the web server and routes
func (d *JS8Daemon) setupWebServer() error {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	// Serve static files
	router.Static("/static", "./web/static")
	router.LoadHTMLGlob("web/templates/*")

	// Main web interface
	router.GET("/", d.handleHome)

	// API routes
	api := router.Group("/api/v1")
	{
		api.GET("/status", d.handleGetStatus)
		api.GET("/messages", d.handleGetMessages)
		api.POST("/messages", d.handleSendMessage)
		api.GET("/radio", d.handleGetRadio)
		api.PUT("/radio/frequency", d.handleSetFrequency)
	}

	addr := fmt.Sprintf("%s:%d", d.config.Web.BindAddress, d.config.Web.Port)
	d.webServer = &http.Server{
		Addr:    addr,
		Handler: router,
	}

	return nil
}

// oledDriver manages OLED display updates
func (d *JS8Daemon) oledDriver() {
	defer d.wg.Done()

	log.Printf("Starting OLED driver")

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-d.ctx.Done():
			return

		case <-ticker.C:
			// Get current status
			status, err := d.socketClient.GetStatus()
			if err != nil {
				log.Printf("OLED: failed to get status: %v", err)
				continue
			}

			// Get latest message
			messages, err := d.socketClient.GetMessages(1)
			if err != nil {
				log.Printf("OLED: failed to get messages: %v", err)
				continue
			}

			// Update OLED display
			d.updateOLED(status, messages)
		}
	}
}

// updateOLED updates the OLED display (mock implementation)
func (d *JS8Daemon) updateOLED(status *protocol.Status, messages []protocol.Message) {
	// TODO: Implement actual OLED driver
	// For now, just log what would be displayed
	log.Printf("OLED: %s %s | Freq: %.3f", status.Callsign, status.Grid, float64(status.Frequency)/1000000.0)

	if len(messages) > 0 {
		msg := messages[0]
		log.Printf("OLED: RX: %s: %s", msg.From, msg.Message)
	}
}
