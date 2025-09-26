package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/dougsko/js8d/pkg/config"
	"github.com/dougsko/js8d/pkg/logging"
	"github.com/dougsko/js8d/pkg/verbose"
)

var (
	configPath = flag.String("config", "config.yaml", "Configuration file path")
	version    = flag.Bool("version", false, "Show version information")
	verboseFlag = flag.Bool("verbose", false, "Enable verbose logging")
)

const (
	Version = "0.1.0-dev"
	Build   = "development"
)

func main() {
	flag.Parse()

	// Set verbose logging flag
	verbose.SetEnabled(*verboseFlag)

	// Set hamlib debug level early based on verbose flag
	if *verboseFlag {
		os.Setenv("HAMLIB_DEBUG_LEVEL", "3") // Verbose hamlib debugging
	} else {
		os.Setenv("HAMLIB_DEBUG_LEVEL", "0") // Suppress hamlib debug output
	}

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

	// Initialize logging system
	if err := logging.InitGlobalLogger(cfg); err != nil {
		log.Fatalf("Failed to initialize logging: %v", err)
	}
	defer logging.CloseGlobalLogger()

	// Switch to using the new logger
	logging.Info("main", fmt.Sprintf("js8d version %s starting...", Version))
	logging.Info("main", fmt.Sprintf("Station: %s (%s)", cfg.Station.Callsign, cfg.Station.Grid))
	logging.Info("main", fmt.Sprintf("Radio: %s on %s", cfg.GetRadioName(), cfg.Radio.Device))
	logging.Info("main", fmt.Sprintf("Web interface: http://%s:%d", cfg.Web.BindAddress, cfg.Web.Port))

	// Create the daemon with config path for reloading
	daemon, err := NewJS8Daemon(cfg, *configPath, *verboseFlag)
	if err != nil {
		logging.Error("main", fmt.Sprintf("Failed to create daemon: %v", err))
		os.Exit(1)
	}

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start the daemon
	if err := daemon.Start(); err != nil {
		logging.Error("main", fmt.Sprintf("Failed to start daemon: %v", err))
		os.Exit(1)
	}

	logging.Info("main", "js8d started successfully")

	// Wait for shutdown signal
	<-sigChan
	logging.Info("main", "Shutting down...")

	// Graceful shutdown
	if err := daemon.Stop(); err != nil {
		logging.Error("main", fmt.Sprintf("Error during shutdown: %v", err))
	}

	logging.Info("main", "js8d stopped")
}
