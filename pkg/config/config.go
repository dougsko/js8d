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
		Level       string `yaml:"level"`        // debug, info, warn, error
		File        string `yaml:"file"`         // log file path
		MaxSize     int    `yaml:"max_size"`     // maximum size in MB
		MaxBackups  int    `yaml:"max_backups"`  // number of old log files to keep
		MaxAge      int    `yaml:"max_age"`      // maximum age in days
		Compress    bool   `yaml:"compress"`     // compress old log files
		Console     bool   `yaml:"console"`      // also log to console/stdout
		Structured  bool   `yaml:"structured"`   // use structured JSON logging
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

	// Set logging defaults
	if config.Logging.Level == "" {
		config.Logging.Level = "info"
	}
	// Only set default log file if not explicitly configured
	// Empty file path means console-only logging
	if config.Logging.MaxSize == 0 {
		config.Logging.MaxSize = 100 // 100MB
	}
	if config.Logging.MaxBackups == 0 {
		config.Logging.MaxBackups = 5
	}
	if config.Logging.MaxAge == 0 {
		config.Logging.MaxAge = 30 // 30 days
	}
	// Console and Compress default to false
	// Structured defaults to false

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

// GetRadioName returns a friendly name for the radio model
func (c *Config) GetRadioName() string {
	switch c.Radio.Model {
	case "1":
		return "Hamlib Dummy"
	case "2":
		return "Yaesu FT-847"
	case "120":
		return "Yaesu FT-817"
	case "122":
		return "Yaesu FT-857"
	case "135":
		return "Yaesu FT-891"
	case "1035":
		return "Yaesu FT-991"
	case "1001":
		return "Yaesu FT-1000"
	case "1007":
		return "Yaesu FT-1000MP"
	case "1014":
		return "Yaesu FT-2000"
	case "1020":
		return "Yaesu FT-450"
	case "1021":
		return "Yaesu FT-950"
	case "1024":
		return "Yaesu FT-5000"
	case "1027":
		return "Yaesu FT-9000"
	case "201":
		return "Kenwood TS-50"
	case "202":
		return "Kenwood TS-440"
	case "203":
		return "Kenwood TS-450"
	case "204":
		return "Kenwood TS-570"
	case "205":
		return "Kenwood TS-680"
	case "206":
		return "Kenwood TS-690"
	case "207":
		return "Kenwood TS-711"
	case "208":
		return "Kenwood TS-790"
	case "209":
		return "Kenwood TS-811"
	case "210":
		return "Kenwood TS-850"
	case "211":
		return "Kenwood TS-870"
	case "212":
		return "Kenwood TS-940"
	case "213":
		return "Kenwood TS-950"
	case "214":
		return "Kenwood TS-2000"
	case "229":
		return "Kenwood TS-480"
	case "230":
		return "Kenwood TS-590"
	case "231":
		return "Kenwood TS-590SG"
	case "232":
		return "Kenwood TS-990"
	case "235":
		return "Kenwood TS-890"
	case "301":
		return "Icom IC-706"
	case "306":
		return "Icom IC-7000"
	case "307":
		return "Icom IC-7100"
	case "311":
		return "Icom IC-746"
	case "315":
		return "Icom IC-756"
	case "318":
		return "Icom IC-7200"
	case "335":
		return "Icom IC-7300"
	case "360":
		return "Icom IC-9100"
	case "362":
		return "Icom IC-7410"
	case "363":
		return "Icom IC-7700"
	case "364":
		return "Icom IC-7800"
	case "365":
		return "Icom IC-7600"
	case "368":
		return "Icom IC-7851"
	case "2028":
		return "Elecraft K3/KX3"
	case "2029":
		return "Elecraft KX2"
	case "2030":
		return "Elecraft K4"
	case "10001":
		return "QRP Labs QDX"
	case "10002":
		return "QRP Labs QMX"
	case "":
		return "No Radio"
	default:
		return fmt.Sprintf("Radio Model %s", c.Radio.Model)
	}
}
