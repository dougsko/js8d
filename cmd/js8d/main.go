package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/js8call/js8d/pkg/config"
)

var (
	configPath = flag.String("config", "config.yaml", "Configuration file path")
	version    = flag.Bool("version", false, "Show version information")
)

const (
	Version = "0.1.0-dev"
	Build   = "development"
)

func main() {
	flag.Parse()

	if *version {
		fmt.Printf("js8d version %s (%s)\n", Version, Build)
		os.Exit(0)
	}

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	log.Printf("js8d version %s starting...", Version)
	log.Printf("Station: %s (%s)", cfg.Station.Callsign, cfg.Station.Grid)
	log.Printf("Radio: %s on %s", cfg.Radio.Model, cfg.Radio.Device)
	log.Printf("Web interface: http://%s:%d", cfg.Web.BindAddress, cfg.Web.Port)

	// Create the daemon
	daemon, err := NewJS8Daemon(cfg)
	if err != nil {
		log.Fatalf("Failed to create daemon: %v", err)
	}

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start the daemon
	if err := daemon.Start(); err != nil {
		log.Fatalf("Failed to start daemon: %v", err)
	}

	log.Printf("js8d started successfully")

	// Wait for shutdown signal
	<-sigChan
	log.Printf("Shutting down...")

	// Graceful shutdown
	if err := daemon.Stop(); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}

	log.Printf("js8d stopped")
}
