package hardware

// RadioConfig represents radio configuration
type RadioConfig struct {
	Model    string // Hamlib model name or number
	Device   string // Serial device path (e.g., /dev/ttyUSB0)
	BaudRate int    // Serial baud rate
	Enabled  bool   // Whether radio control is enabled
}

// RadioInterface defines radio control operations
type RadioInterface interface {
	Initialize() error
	Close() error

	// Frequency control
	SetFrequency(freq int64) error
	GetFrequency() (int64, error)

	// Mode control
	SetMode(mode string, bandwidth int) error
	GetMode() (string, int, error)

	// PTT control
	SetPTT(state bool) error
	GetPTT() (bool, error)

	// Radio information
	GetRadioInfo() (RadioInfo, error)
	IsConnected() bool

	// Power and status
	GetPowerLevel() (float32, error)
	GetSWRLevel() (float32, error)
	GetSignalLevel() (int, error)
}

// RadioInfo represents radio information
type RadioInfo struct {
	Model       string
	Manufacturer string
	Version     string
	Capabilities []string
}

// RadioMode constants for common amateur radio modes
const (
	ModeUSB  = "USB"
	ModeLSB  = "LSB"
	ModeCW   = "CW"
	ModeRTTY = "RTTY"
	ModePSK  = "PSK"
	ModeJT8  = "JT8" // For JS8 and similar digital modes
	ModeFM   = "FM"
	ModeAM   = "AM"
)

// Common amateur radio frequencies (Hz)
const (
	Band80m_JS8  = 3578000   // 80m JS8 frequency
	Band40m_JS8  = 7078000   // 40m JS8 frequency
	Band20m_JS8  = 14078000  // 20m JS8 frequency
	Band17m_JS8  = 18104000  // 17m JS8 frequency
	Band15m_JS8  = 21078000  // 15m JS8 frequency
	Band12m_JS8  = 24922000  // 12m JS8 frequency
	Band10m_JS8  = 28078000  // 10m JS8 frequency
	Band6m_JS8   = 50318000  // 6m JS8 frequency
	Band2m_JS8   = 144178000 // 2m JS8 frequency
)

// JS8 bandwidth in Hz
const JS8Bandwidth = 50