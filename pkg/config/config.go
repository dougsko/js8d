package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

// Config represents the js8d configuration
type Config struct {
	Station struct {
		Callsign string `yaml:"callsign"`
		Grid     string `yaml:"grid"`
	} `yaml:"station"`

	Radio struct {
		// Basic Configuration
		UseHamlib    bool   `yaml:"use_hamlib"`
		Model        string `yaml:"model"`
		PollInterval int    `yaml:"poll_interval"`

		// CAT Control Parameters
		Device    string `yaml:"device"`
		BaudRate  int    `yaml:"baud_rate"`
		DataBits  string `yaml:"data_bits"`
		StopBits  string `yaml:"stop_bits"`
		Handshake string `yaml:"handshake"`
		DTR       string `yaml:"dtr"`
		RTS       string `yaml:"rts"`

		// PTT Configuration
		PTTMethod      string  `yaml:"ptt_method"`
		PTTPort        string  `yaml:"ptt_port"`
		Mode           string  `yaml:"mode"`
		TxAudioSource  string  `yaml:"tx_audio_source"`
		SplitOperation string  `yaml:"split_operation"`
		PTTCommand     string  `yaml:"ptt_command"`
		TxDelay        float64 `yaml:"tx_delay"`
	} `yaml:"radio"`

	Audio struct {
		// Device Configuration
		InputDevice        string `yaml:"input_device"`
		InputChannels      string `yaml:"input_channels"`
		OutputDevice       string `yaml:"output_device"`
		OutputChannels     string `yaml:"output_channels"`
		NotificationDevice string `yaml:"notification_device"`

		// Audio Parameters
		SampleRate   int `yaml:"sample_rate"`
		BufferSize   int `yaml:"buffer_size"`

		// Advanced Options
		SaveDirectory     string `yaml:"save_directory"`
		RememberPowerTx   bool   `yaml:"remember_power_tx"`
		RememberPowerTune bool   `yaml:"remember_power_tune"`
	} `yaml:"audio"`

	Web struct {
		Port        int    `yaml:"port"`
		BindAddress string `yaml:"bind_address"`
	} `yaml:"web"`

	API struct {
		WebSocketPort int    `yaml:"websocket_port"`
		UnixSocket    string `yaml:"unix_socket"`
	} `yaml:"api"`

	Storage struct {
		DatabasePath string `yaml:"database_path"`
		MaxMessages  int    `yaml:"max_messages"`
	} `yaml:"storage"`

	Logging struct {
		Level string `yaml:"level"`
		File  string `yaml:"file"`
	} `yaml:"logging"`

	Hardware struct {
		PTTGPIOPin     int  `yaml:"ptt_gpio_pin"`
		StatusLEDPin   int  `yaml:"status_led_pin"`
		EnableGPIO     bool `yaml:"enable_gpio"`
		EnableOLED     bool `yaml:"enable_oled"`
		OLEDI2CAddress int  `yaml:"oled_i2c_address"`
		OLEDWidth      int  `yaml:"oled_width"`
		OLEDHeight     int  `yaml:"oled_height"`
	} `yaml:"hardware"`
}

// LoadConfig loads configuration from a YAML file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Set defaults
	if config.Audio.SampleRate == 0 {
		config.Audio.SampleRate = 48000
	}
	if config.Audio.BufferSize == 0 {
		config.Audio.BufferSize = 1024
	}
	if config.Audio.InputChannels == "" {
		config.Audio.InputChannels = "mono"
	}
	if config.Audio.OutputChannels == "" {
		config.Audio.OutputChannels = "mono"
	}
	if config.Audio.NotificationDevice == "" {
		config.Audio.NotificationDevice = "Built-in Output"
	}
	if config.Radio.Model == "" {
		config.Radio.Model = "10001" // QRP Labs QDX
	}
	if config.Radio.PollInterval == 0 {
		config.Radio.PollInterval = 1000
	}
	if config.Radio.BaudRate == 0 {
		config.Radio.BaudRate = 115200
	}
	if config.Radio.DataBits == "" {
		config.Radio.DataBits = "default"
	}
	if config.Radio.StopBits == "" {
		config.Radio.StopBits = "default"
	}
	if config.Radio.Handshake == "" {
		config.Radio.Handshake = "default"
	}
	if config.Radio.DTR == "" {
		config.Radio.DTR = "default"
	}
	if config.Radio.RTS == "" {
		config.Radio.RTS = "default"
	}
	if config.Radio.PTTMethod == "" {
		config.Radio.PTTMethod = "cat"
	}
	if config.Radio.Mode == "" {
		config.Radio.Mode = "data"
	}
	if config.Radio.TxAudioSource == "" {
		config.Radio.TxAudioSource = "front"
	}
	if config.Radio.SplitOperation == "" {
		config.Radio.SplitOperation = "rig"
	}
	if config.Radio.TxDelay == 0 {
		config.Radio.TxDelay = 0.2
	}
	if config.Web.Port == 0 {
		config.Web.Port = 8080
	}
	if config.Web.BindAddress == "" {
		config.Web.BindAddress = "0.0.0.0"
	}
	if config.Storage.MaxMessages == 0 {
		config.Storage.MaxMessages = 10000
	}

	return &config, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.Station.Callsign == "" {
		return fmt.Errorf("station callsign is required")
	}
	if c.Station.Grid == "" {
		return fmt.Errorf("station grid is required")
	}
	if c.Radio.UseHamlib && c.Radio.Device == "" {
		return fmt.Errorf("radio device is required when using Hamlib")
	}
	if c.Audio.InputDevice == "" {
		c.Audio.InputDevice = "default"
	}
	if c.Audio.OutputDevice == "" {
		c.Audio.OutputDevice = "default"
	}
	return nil
}
