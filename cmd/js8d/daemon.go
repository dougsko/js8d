package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/dougsko/js8d/pkg/client"
	"github.com/dougsko/js8d/pkg/config"
	"github.com/dougsko/js8d/pkg/engine"
)

// JS8Daemon represents the main daemon with Unix socket architecture
type JS8Daemon struct {
	config     *config.Config
	configPath string
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	verbose    bool

	// Core components
	coreEngine   *engine.CoreEngine
	socketClient *client.SocketClient
	webServer    *http.Server

	// Socket path
	socketPath string
}

// NewJS8Daemon creates a new daemon instance with config path for reloading
func NewJS8Daemon(cfg *config.Config, configPath string, verbose bool) (*JS8Daemon, error) {
	ctx, cancel := context.WithCancel(context.Background())

	socketPath := cfg.API.UnixSocket
	if socketPath == "" {
		socketPath = "/tmp/js8d.sock"
	}

	daemon := &JS8Daemon{
		config:       cfg,
		configPath:   configPath,
		ctx:          ctx,
		cancel:       cancel,
		verbose:      verbose,
		socketPath:   socketPath,
		socketClient: client.NewSocketClient(socketPath),
	}

	// Create core engine with config path for reloading
	daemon.coreEngine = engine.NewCoreEngine(cfg, socketPath, configPath)

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

	// OLED is handled directly by the core engine hardware manager

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
	router.GET("/settings", d.handleSettings)

	// API routes
	api := router.Group("/api/v1")
	{
		api.GET("/status", d.handleGetStatus)
		api.GET("/messages", d.handleGetMessages)
		api.POST("/messages", d.handleSendMessage)
		api.GET("/messages/history", d.handleGetMessageHistory)
		api.GET("/messages/conversations", d.handleGetConversations)
		api.POST("/messages/mark-read", d.handleMarkMessagesRead)
		api.GET("/messages/search", d.handleSearchMessages)
		api.GET("/messages/stats", d.handleGetMessageStats)
		api.POST("/messages/cleanup", d.handleCleanupMessages)
		api.GET("/radio", d.handleGetRadio)
		api.PUT("/radio/frequency", d.handleSetFrequency)
		api.POST("/abort", d.handleAbortTransmission)
		api.GET("/config", d.handleGetConfig)
		api.POST("/config", d.handleSaveConfig)
		api.POST("/config/reload", d.handleReloadConfig)
		api.POST("/radio/retry-connection", d.handleRetryRadioConnection)
		api.POST("/radio/test-cat", d.handleTestCAT)
		api.POST("/radio/test-ptt", d.handleTestPTT)
		api.POST("/radio/test-ptt-off", d.handleTestPTTOff)
		api.GET("/audio/stats", d.handleGetAudioStats)
		api.GET("/audio/test", d.handleTestAudioData)
		api.GET("/audio/devices", d.handleGetAudioDevices)
		api.GET("/serial/devices", d.handleGetSerialDevices)
	}

	// WebSocket endpoints
	router.GET("/ws/audio", d.handleAudioWebSocket)

	addr := fmt.Sprintf("%s:%d", d.config.Web.BindAddress, d.config.Web.Port)
	d.webServer = &http.Server{
		Addr:    addr,
		Handler: router,
	}

	return nil
}

