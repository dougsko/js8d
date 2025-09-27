package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "js8d-config-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Run("Valid Config", func(t *testing.T) {
		configContent := `
station:
  callsign: "K3DEP"
  grid: "FN20"

radio:
  use_hamlib: true
  model: "2028"
  device: "/dev/ttyUSB0"
  baud_rate: 38400

audio:
  input_device: "hw:1,0"
  output_device: "hw:1,0"
  sample_rate: 48000
  buffer_size: 1024

web:
  port: 8080
  bind_address: "0.0.0.0"

storage:
  database_path: "/tmp/js8d.db"
  max_messages: 5000

logging:
  level: "debug"
  file: "/var/log/js8d.log"
  console: true
`
		configPath := filepath.Join(tempDir, "valid.yaml")
		if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
			t.Fatalf("Failed to write config file: %v", err)
		}

		config, err := LoadConfig(configPath)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		// Test parsed values
		if config.Station.Callsign != "K3DEP" {
			t.Errorf("Expected callsign K3DEP, got %s", config.Station.Callsign)
		}
		if config.Station.Grid != "FN20" {
			t.Errorf("Expected grid FN20, got %s", config.Station.Grid)
		}
		if config.Radio.Model != "2028" {
			t.Errorf("Expected radio model 2028, got %s", config.Radio.Model)
		}
		if config.Radio.BaudRate != 38400 {
			t.Errorf("Expected baud rate 38400, got %d", config.Radio.BaudRate)
		}
		if config.Audio.SampleRate != 48000 {
			t.Errorf("Expected sample rate 48000, got %d", config.Audio.SampleRate)
		}
		if config.Web.Port != 8080 {
			t.Errorf("Expected web port 8080, got %d", config.Web.Port)
		}
		if config.Storage.MaxMessages != 5000 {
			t.Errorf("Expected max messages 5000, got %d", config.Storage.MaxMessages)
		}
		if config.Logging.Level != "debug" {
			t.Errorf("Expected log level debug, got %s", config.Logging.Level)
		}
	})

	t.Run("Config With Defaults", func(t *testing.T) {
		// Minimal config that should get defaults applied
		configContent := `
station:
  callsign: "N0ABC"
  grid: "EM12"
`
		configPath := filepath.Join(tempDir, "minimal.yaml")
		if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
			t.Fatalf("Failed to write config file: %v", err)
		}

		config, err := LoadConfig(configPath)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		// Test default values
		if config.Audio.SampleRate != 48000 {
			t.Errorf("Expected default sample rate 48000, got %d", config.Audio.SampleRate)
		}
		if config.Audio.BufferSize != 1024 {
			t.Errorf("Expected default buffer size 1024, got %d", config.Audio.BufferSize)
		}
		if config.Audio.InputChannels != "mono" {
			t.Errorf("Expected default input channels mono, got %s", config.Audio.InputChannels)
		}
		if config.Radio.Model != "10001" {
			t.Errorf("Expected default radio model 10001, got %s", config.Radio.Model)
		}
		if config.Radio.BaudRate != 115200 {
			t.Errorf("Expected default baud rate 115200, got %d", config.Radio.BaudRate)
		}
		if config.Radio.PTTMethod != "cat" {
			t.Errorf("Expected default PTT method cat, got %s", config.Radio.PTTMethod)
		}
		if config.Web.Port != 8080 {
			t.Errorf("Expected default web port 8080, got %d", config.Web.Port)
		}
		if config.Web.BindAddress != "0.0.0.0" {
			t.Errorf("Expected default bind address 0.0.0.0, got %s", config.Web.BindAddress)
		}
		if config.Storage.MaxMessages != 10000 {
			t.Errorf("Expected default max messages 10000, got %d", config.Storage.MaxMessages)
		}
		if config.Logging.Level != "info" {
			t.Errorf("Expected default log level info, got %s", config.Logging.Level)
		}
		if config.Logging.MaxSize != 100 {
			t.Errorf("Expected default log max size 100, got %d", config.Logging.MaxSize)
		}
		if config.Logging.MaxBackups != 5 {
			t.Errorf("Expected default log max backups 5, got %d", config.Logging.MaxBackups)
		}
		if config.Logging.MaxAge != 30 {
			t.Errorf("Expected default log max age 30, got %d", config.Logging.MaxAge)
		}
	})

	t.Run("File Not Found", func(t *testing.T) {
		_, err := LoadConfig("/nonexistent/path/config.yaml")
		if err == nil {
			t.Error("Expected error for nonexistent file, got nil")
		}
		if !strings.Contains(err.Error(), "failed to read config file") {
			t.Errorf("Expected 'failed to read config file' error, got: %v", err)
		}
	})

	t.Run("Invalid YAML", func(t *testing.T) {
		configContent := `
station:
  callsign: "K3DEP"
  grid: [invalid yaml structure
`
		configPath := filepath.Join(tempDir, "invalid.yaml")
		if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
			t.Fatalf("Failed to write config file: %v", err)
		}

		_, err := LoadConfig(configPath)
		if err == nil {
			t.Error("Expected error for invalid YAML, got nil")
		}
		if !strings.Contains(err.Error(), "failed to parse config file") {
			t.Errorf("Expected 'failed to parse config file' error, got: %v", err)
		}
	})

	t.Run("Empty File", func(t *testing.T) {
		configPath := filepath.Join(tempDir, "empty.yaml")
		if err := os.WriteFile(configPath, []byte(""), 0644); err != nil {
			t.Fatalf("Failed to write empty config file: %v", err)
		}

		config, err := LoadConfig(configPath)
		if err != nil {
			t.Fatalf("Expected no error for empty file, got: %v", err)
		}

		// Should have all defaults applied
		if config.Audio.SampleRate != 48000 {
			t.Errorf("Expected default sample rate for empty file, got %d", config.Audio.SampleRate)
		}
	})
}

func TestValidate(t *testing.T) {
	t.Run("Valid Config", func(t *testing.T) {
		config := &Config{
			Station: struct {
				Callsign string `yaml:"callsign"`
				Grid     string `yaml:"grid"`
			}{
				Callsign: "K3DEP",
				Grid:     "FN20",
			},
			Radio: struct {
				UseHamlib       bool    `yaml:"use_hamlib"`
				Model           string  `yaml:"model"`
				PollInterval    int     `yaml:"poll_interval"`
				Device          string  `yaml:"device"`
				BaudRate        int     `yaml:"baud_rate"`
				DataBits        string  `yaml:"data_bits"`
				StopBits        string  `yaml:"stop_bits"`
				Handshake       string  `yaml:"handshake"`
				DTR             string  `yaml:"dtr"`
				RTS             string  `yaml:"rts"`
				CIVAddress      string  `yaml:"civ_address"`
				CIVTransceive   bool    `yaml:"civ_transceive"`
				PTTMethod       string  `yaml:"ptt_method"`
				PTTPort         string  `yaml:"ptt_port"`
				Mode            string  `yaml:"mode"`
				TxAudioSource   string  `yaml:"tx_audio_source"`
				SplitOperation  string  `yaml:"split_operation"`
				PTTCommand      string  `yaml:"ptt_command"`
				TxDelay         float64 `yaml:"tx_delay"`
			}{
				UseHamlib: true,
				Model:     "2028",
				Device:    "/dev/ttyUSB0",
			},
		}

		err := config.Validate()
		if err != nil {
			t.Errorf("Expected no error for valid config, got: %v", err)
		}
	})

	t.Run("Missing Callsign", func(t *testing.T) {
		config := &Config{
			Station: struct {
				Callsign string `yaml:"callsign"`
				Grid     string `yaml:"grid"`
			}{
				Grid: "FN20",
			},
		}

		err := config.Validate()
		if err == nil {
			t.Error("Expected error for missing callsign, got nil")
		}
		if !strings.Contains(err.Error(), "station callsign is required") {
			t.Errorf("Expected callsign error, got: %v", err)
		}
	})

	t.Run("Missing Grid", func(t *testing.T) {
		config := &Config{
			Station: struct {
				Callsign string `yaml:"callsign"`
				Grid     string `yaml:"grid"`
			}{
				Callsign: "K3DEP",
			},
		}

		err := config.Validate()
		if err == nil {
			t.Error("Expected error for missing grid, got nil")
		}
		if !strings.Contains(err.Error(), "station grid is required") {
			t.Errorf("Expected grid error, got: %v", err)
		}
	})

	t.Run("Hamlib Without Device", func(t *testing.T) {
		config := &Config{
			Station: struct {
				Callsign string `yaml:"callsign"`
				Grid     string `yaml:"grid"`
			}{
				Callsign: "K3DEP",
				Grid:     "FN20",
			},
			Radio: struct {
				UseHamlib       bool    `yaml:"use_hamlib"`
				Model           string  `yaml:"model"`
				PollInterval    int     `yaml:"poll_interval"`
				Device          string  `yaml:"device"`
				BaudRate        int     `yaml:"baud_rate"`
				DataBits        string  `yaml:"data_bits"`
				StopBits        string  `yaml:"stop_bits"`
				Handshake       string  `yaml:"handshake"`
				DTR             string  `yaml:"dtr"`
				RTS             string  `yaml:"rts"`
				CIVAddress      string  `yaml:"civ_address"`
				CIVTransceive   bool    `yaml:"civ_transceive"`
				PTTMethod       string  `yaml:"ptt_method"`
				PTTPort         string  `yaml:"ptt_port"`
				Mode            string  `yaml:"mode"`
				TxAudioSource   string  `yaml:"tx_audio_source"`
				SplitOperation  string  `yaml:"split_operation"`
				PTTCommand      string  `yaml:"ptt_command"`
				TxDelay         float64 `yaml:"tx_delay"`
			}{
				UseHamlib: true,
				Model:     "2028", // Not dummy rig
				Device:    "",     // Missing device
			},
		}

		err := config.Validate()
		if err == nil {
			t.Error("Expected error for Hamlib without device, got nil")
		}
		if !strings.Contains(err.Error(), "radio device is required") {
			t.Errorf("Expected device error, got: %v", err)
		}
	})

	t.Run("Dummy Rig Without Device", func(t *testing.T) {
		config := &Config{
			Station: struct {
				Callsign string `yaml:"callsign"`
				Grid     string `yaml:"grid"`
			}{
				Callsign: "K3DEP",
				Grid:     "FN20",
			},
			Radio: struct {
				UseHamlib       bool    `yaml:"use_hamlib"`
				Model           string  `yaml:"model"`
				PollInterval    int     `yaml:"poll_interval"`
				Device          string  `yaml:"device"`
				BaudRate        int     `yaml:"baud_rate"`
				DataBits        string  `yaml:"data_bits"`
				StopBits        string  `yaml:"stop_bits"`
				Handshake       string  `yaml:"handshake"`
				DTR             string  `yaml:"dtr"`
				RTS             string  `yaml:"rts"`
				CIVAddress      string  `yaml:"civ_address"`
				CIVTransceive   bool    `yaml:"civ_transceive"`
				PTTMethod       string  `yaml:"ptt_method"`
				PTTPort         string  `yaml:"ptt_port"`
				Mode            string  `yaml:"mode"`
				TxAudioSource   string  `yaml:"tx_audio_source"`
				SplitOperation  string  `yaml:"split_operation"`
				PTTCommand      string  `yaml:"ptt_command"`
				TxDelay         float64 `yaml:"tx_delay"`
			}{
				UseHamlib: true,
				Model:     "1", // Dummy rig
				Device:    "",  // No device needed for dummy
			},
		}

		err := config.Validate()
		if err != nil {
			t.Errorf("Expected no error for dummy rig without device, got: %v", err)
		}
	})

	t.Run("Default Audio Devices", func(t *testing.T) {
		config := &Config{
			Station: struct {
				Callsign string `yaml:"callsign"`
				Grid     string `yaml:"grid"`
			}{
				Callsign: "K3DEP",
				Grid:     "FN20",
			},
			Audio: struct {
				InputDevice        string `yaml:"input_device"`
				InputChannels      string `yaml:"input_channels"`
				OutputDevice       string `yaml:"output_device"`
				OutputChannels     string `yaml:"output_channels"`
				NotificationDevice string `yaml:"notification_device"`
				SampleRate         int    `yaml:"sample_rate"`
				BufferSize         int    `yaml:"buffer_size"`
				SaveDirectory      string `yaml:"save_directory"`
				RememberPowerTx    bool   `yaml:"remember_power_tx"`
				RememberPowerTune  bool   `yaml:"remember_power_tune"`
			}{
				InputDevice:  "",
				OutputDevice: "",
			},
		}

		err := config.Validate()
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		// Should set defaults
		if config.Audio.InputDevice != "default" {
			t.Errorf("Expected default input device, got %s", config.Audio.InputDevice)
		}
		if config.Audio.OutputDevice != "default" {
			t.Errorf("Expected default output device, got %s", config.Audio.OutputDevice)
		}
	})
}

func TestGetRadioName(t *testing.T) {
	testCases := []struct {
		model    string
		expected string
	}{
		{"1", "Hamlib Dummy"},
		{"2028", "Elecraft K3/KX3"},
		{"10001", "QRP Labs QDX"},
		{"10002", "QRP Labs QMX"},
		{"2029", "Elecraft KX2"},
		{"2030", "Elecraft K4"},
		{"335", "Icom IC-7300"},
		{"229", "Kenwood TS-480"},
		{"120", "Yaesu FT-817"},
		{"135", "Yaesu FT-891"},
		{"1035", "Yaesu FT-991"},
		{"", "No Radio"},
		{"99999", "Radio Model 99999"},
		{"unknown", "Radio Model unknown"},
	}

	for _, tc := range testCases {
		t.Run(tc.model, func(t *testing.T) {
			config := &Config{
				Radio: struct {
					UseHamlib       bool    `yaml:"use_hamlib"`
					Model           string  `yaml:"model"`
					PollInterval    int     `yaml:"poll_interval"`
					Device          string  `yaml:"device"`
					BaudRate        int     `yaml:"baud_rate"`
					DataBits        string  `yaml:"data_bits"`
					StopBits        string  `yaml:"stop_bits"`
					Handshake       string  `yaml:"handshake"`
					DTR             string  `yaml:"dtr"`
					RTS             string  `yaml:"rts"`
					CIVAddress      string  `yaml:"civ_address"`
					CIVTransceive   bool    `yaml:"civ_transceive"`
					PTTMethod       string  `yaml:"ptt_method"`
					PTTPort         string  `yaml:"ptt_port"`
					Mode            string  `yaml:"mode"`
					TxAudioSource   string  `yaml:"tx_audio_source"`
					SplitOperation  string  `yaml:"split_operation"`
					PTTCommand      string  `yaml:"ptt_command"`
					TxDelay         float64 `yaml:"tx_delay"`
				}{
					Model: tc.model,
				},
			}

			result := config.GetRadioName()
			if result != tc.expected {
				t.Errorf("Expected %s, got %s", tc.expected, result)
			}
		})
	}
}

func TestConfigIntegration(t *testing.T) {
	// Test the full flow: load -> validate
	tempDir, err := os.MkdirTemp("", "js8d-config-integration")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configContent := `
station:
  callsign: "K3DEP"
  grid: "FN20"

radio:
  use_hamlib: true
  model: "10002"
  device: "/dev/serial/by-id/usb-QRP_Labs_QMX_Transceiver-if00"
  baud_rate: 115200

audio:
  input_device: "plughw:3,0"
  output_device: "plughw:3,0"

web:
  port: 8080

logging:
  level: "info"
  console: true
`

	configPath := filepath.Join(tempDir, "integration.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Load config
	config, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Validate config
	if err := config.Validate(); err != nil {
		t.Fatalf("Failed to validate config: %v", err)
	}

	// Test specific values
	if config.Station.Callsign != "K3DEP" {
		t.Errorf("Expected callsign K3DEP, got %s", config.Station.Callsign)
	}

	radioName := config.GetRadioName()
	if radioName != "QRP Labs QMX" {
		t.Errorf("Expected QRP Labs QMX, got %s", radioName)
	}

	// Verify defaults were applied
	if config.Storage.MaxMessages != 10000 {
		t.Errorf("Expected default max messages, got %d", config.Storage.MaxMessages)
	}
}