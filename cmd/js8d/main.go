package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/dougsko/js8d/pkg/config"
	"github.com/dougsko/js8d/pkg/logging"
	"github.com/dougsko/js8d/pkg/verbose"
)

var (
	configPath  = flag.String("config", "config.yaml", "Configuration file path")
	pidFilePath = flag.String("pidfile", "", "PID file path (default: /var/run/js8d.pid or ./js8d.pid)")
	version     = flag.Bool("version", false, "Show version information")
	verboseFlag = flag.Bool("verbose", false, "Enable verbose logging")
)

const (
	Version = "0.1.0-dev"
	Build   = "development"
)

// PID file management functions
func getDefaultPidFile() string {
	// Try /var/run/js8d.pid first (system daemon location)
	systemPidFile := "/var/run/js8d.pid"
	if dir := filepath.Dir(systemPidFile); isWritableDir(dir) {
		return systemPidFile
	}

	// Fall back to current directory
	return "./js8d.pid"
}

func isWritableDir(dir string) bool {
	// Check if directory exists and is writable
	if stat, err := os.Stat(dir); err == nil && stat.IsDir() {
		// Try to create a temporary file to test write access
		testFile := filepath.Join(dir, ".js8d_write_test")
		if f, err := os.Create(testFile); err == nil {
			f.Close()
			os.Remove(testFile)
			return true
		}
	}
	return false
}

func createPidFile(pidFile string) error {
	// Check if another instance is running
	if err := checkExistingPid(pidFile); err != nil {
		return err
	}

	// Create directory if it doesn't exist
	if dir := filepath.Dir(pidFile); dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create PID file directory: %v", err)
		}
	}

	// Write current PID to file
	pid := os.Getpid()
	content := fmt.Sprintf("%d\n", pid)

	if err := os.WriteFile(pidFile, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write PID file: %v", err)
	}

	return nil
}

func checkExistingPid(pidFile string) error {
	// Read existing PID file if it exists
	data, err := os.ReadFile(pidFile)
	if os.IsNotExist(err) {
		return nil // No existing PID file, OK to proceed
	}
	if err != nil {
		return fmt.Errorf("failed to read existing PID file: %v", err)
	}

	// Parse PID from file
	pidStr := strings.TrimSpace(string(data))
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		// Invalid PID file, remove it and continue
		os.Remove(pidFile)
		return nil
	}

	// Check if process is still running
	if isProcessRunning(pid) {
		return fmt.Errorf("js8d is already running with PID %d", pid)
	}

	// Stale PID file, remove it
	os.Remove(pidFile)
	return nil
}

func isProcessRunning(pid int) bool {
	// Try to send signal 0 to check if process exists
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// Signal 0 doesn't actually send a signal, just checks if process exists
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

func removePidFile(pidFile string) {
	if pidFile != "" {
		if err := os.Remove(pidFile); err != nil && !os.IsNotExist(err) {
			log.Printf("Warning: failed to remove PID file %s: %v", pidFile, err)
		}
	}
}

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

	// Determine PID file path
	var actualPidFile string
	if *pidFilePath != "" {
		actualPidFile = *pidFilePath
	} else {
		actualPidFile = getDefaultPidFile()
	}

	// Create PID file and check for existing instances
	if err := createPidFile(actualPidFile); err != nil {
		log.Fatalf("Failed to create PID file: %v", err)
	}

	// Ensure PID file is removed on exit
	defer removePidFile(actualPidFile)

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
	logging.Info("main", fmt.Sprintf("PID: %d, PID file: %s", os.Getpid(), actualPidFile))
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
