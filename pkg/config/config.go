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
		Device     string `yaml:"device"`
		Model      string `yaml:"model"`
		BaudRate   int    `yaml:"baud_rate"`
		PTTMethod  string `yaml:"ptt_method"`
		UseHamlib  bool   `yaml:"use_hamlib"`
	} `yaml:"radio"`

	Audio struct {
		InputDevice  string `yaml:"input_device"`
		OutputDevice string `yaml:"output_device"`
		SampleRate   int    `yaml:"sample_rate"`
		BufferSize   int    `yaml:"buffer_size"`
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
	if c.Radio.Device == "" {
		return fmt.Errorf("radio device is required")
	}
	if c.Audio.InputDevice == "" {
		return fmt.Errorf("audio input device is required")
	}
	return nil
}
