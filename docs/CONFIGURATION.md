# js8d Configuration Reference

This document provides a comprehensive reference for all js8d configuration options.

## Table of Contents

- [Configuration File Format](#configuration-file-format)
- [Station Configuration](#station-configuration)
- [Audio Configuration](#audio-configuration)
- [Radio Configuration](#radio-configuration)
- [Web Interface Configuration](#web-interface-configuration)
- [Hardware Configuration](#hardware-configuration)
- [Database Configuration](#database-configuration)
- [API Configuration](#api-configuration)
- [Environment Variables](#environment-variables)
- [Configuration Examples](#configuration-examples)

## Configuration File Format

js8d uses YAML format for configuration files. The default configuration file is `config.yaml` in the current directory, but you can specify a different file with the `-config` flag.

```yaml
# Example configuration structure
station:
  # Station identification
audio:
  # Audio system settings
radio:
  # Radio control settings
web:
  # Web interface settings
hardware:
  # Hardware-specific settings
database:
  # Database and storage settings
api:
  # API server settings
```

## Station Configuration

Configure your amateur radio station identification.

```yaml
station:
  callsign: "N0CALL"      # Your amateur radio callsign (required)
  grid: "FN31pr"          # Your 6-character grid square (required)
  name: "js8d Station"    # Optional station name
  qth: "City, State"      # Optional location description
```

**Parameters:**
- `callsign` (string, required): Your amateur radio callsign. Must be valid and properly licensed.
- `grid` (string, required): Your 6-character Maidenhead grid square locator.
- `name` (string, optional): Human-readable station name for display.
- `qth` (string, optional): Location description (city, state, country).

## Audio Configuration

Configure audio input/output devices and parameters.

```yaml
audio:
  # Device Selection
  input_device: "hw:0,0"           # ALSA device for audio input
  output_device: "hw:0,0"          # ALSA device for audio output
  notification_device: "default"   # Device for notification sounds

  # Audio Parameters
  sample_rate: 48000               # Audio sample rate (Hz)
  buffer_size: 1024                # Buffer size (samples)
  input_channels: 1                # Number of input channels
  output_channels: 1               # Number of output channels

  # Audio Processing
  auto_gain: false                 # Enable automatic gain control
  input_gain: 0.8                  # Input gain multiplier (0.0-2.0)
  output_gain: 0.8                 # Output gain multiplier (0.0-2.0)

  # File Storage
  save_directory: "/home/user/js8d/audio"  # Directory for audio recordings
  save_received: false             # Save received audio to files
  save_transmitted: false          # Save transmitted audio to files

  # Advanced Settings
  latency_ms: 50                   # Target audio latency (milliseconds)
  use_float32: false               # Use 32-bit float audio (vs 16-bit int)
```

### Audio Device Configuration

**Linux (ALSA):**
```bash
# List available devices
aplay -l    # Playback devices
arecord -l  # Recording devices

# Common device formats
hw:0,0      # Hardware device, card 0, device 0
plughw:0,0  # Hardware device with format conversion
default     # System default device
pulse       # PulseAudio (if available)
```

**macOS (Core Audio):**
```bash
# Devices are automatically enumerated
# Use the web interface to select from available devices
```

**Device Selection Examples:**
```yaml
# USB Sound Card
audio:
  input_device: "hw:1,0"   # USB device, card 1
  output_device: "hw:1,0"

# Built-in Audio + USB
audio:
  input_device: "hw:1,0"   # USB microphone
  output_device: "hw:0,0"  # Built-in speakers

# PulseAudio
audio:
  input_device: "pulse"
  output_device: "pulse"
```

## Radio Configuration

Configure radio control via Hamlib.

```yaml
radio:
  # Connection Settings
  use_hamlib: true                 # Enable Hamlib radio control
  model: "1"                       # Hamlib rig model number
  device: "/dev/ttyUSB0"          # Serial device path
  baud_rate: 9600                 # Serial baud rate

  # Radio Parameters
  frequency: 14078000             # Default frequency (Hz)
  mode: "USB"                     # Operating mode
  bandwidth: 3000                 # Filter bandwidth (Hz)

  # PTT Configuration
  ptt_method: "cat"               # PTT method: cat, dtr, rts, vox, gpio
  ptt_command: ""                 # Custom PTT command (if ptt_method is "cmd")
  tx_delay: 0.2                   # TX delay in seconds

  # Advanced Settings
  poll_interval: 1000             # Status polling interval (ms)
  timeout: 5000                   # Command timeout (ms)
  retry_count: 3                  # Number of retries for failed commands

  # Radio-Specific Settings
  tx_audio_source: "data"         # TX audio source: data, mic, front, rear
  split_operation: "rig"          # Split operation: rig, fake
  vfo: "A"                        # Default VFO: A, B, MAIN, SUB

  # Power Settings
  power_level: 0.5                # TX power level (0.0-1.0)
  tune_power: 0.1                 # Tune power level (0.0-1.0)
  remember_power_tx: true         # Remember TX power setting
  remember_power_tune: true       # Remember tune power setting
```

### Hamlib Model Numbers

**Popular Radio Models:**
```yaml
# Icom
model: "311"    # IC-7300
model: "3012"   # IC-7610
model: "3070"   # IC-9700

# Yaesu
model: "1035"   # FT-991A
model: "1045"   # FT-710
model: "1049"   # FT-818

# Kenwood
model: "2014"   # TS-590S
model: "2015"   # TS-590SG
model: "2029"   # TS-890S

# Other
model: "1"      # Dummy rig (for testing)
model: "2"      # NET rigctl (network)
```

Use `rigctl -l | grep "Your Radio"` to find your specific model number.

### PTT Methods

```yaml
# CAT Control (recommended)
ptt_method: "cat"

# Serial Port Control
ptt_method: "dtr"    # Use DTR line
ptt_method: "rts"    # Use RTS line

# VOX (Voice Operated eXchange)
ptt_method: "vox"

# GPIO (Raspberry Pi)
ptt_method: "gpio"

# Custom Command
ptt_method: "cmd"
ptt_command: "/usr/local/bin/ptt_on"
```

## Web Interface Configuration

Configure the built-in web server and interface.

```yaml
web:
  # Server Settings
  bind_address: "0.0.0.0"         # Bind address (0.0.0.0 = all interfaces)
  port: 8080                      # HTTP port

  # Security Settings
  enable_cors: true               # Enable CORS headers
  allowed_origins: ["*"]          # Allowed CORS origins
  api_key: ""                     # API key (empty = no authentication)

  # Performance Settings
  read_timeout: 30                # Read timeout (seconds)
  write_timeout: 30               # Write timeout (seconds)
  max_request_size: 1048576       # Max request size (bytes)

  # Static Files
  static_dir: "web/static"        # Static files directory
  template_dir: "web/templates"   # Template files directory

  # Features
  enable_audio_monitoring: true   # Enable audio spectrum display
  enable_message_history: true    # Enable message history interface
  auto_save_config: true          # Auto-save configuration changes
```

### Network Configuration

**Local Only (Secure):**
```yaml
web:
  bind_address: "127.0.0.1"  # Localhost only
  port: 8080
```

**Network Access (Less Secure):**
```yaml
web:
  bind_address: "0.0.0.0"    # All interfaces
  port: 8080
  api_key: "your-secret-key" # Add authentication
```

**Custom Port:**
```yaml
web:
  bind_address: "0.0.0.0"
  port: 8088                 # Use port 8088
```

## Hardware Configuration

Configure hardware-specific features (primarily for Raspberry Pi).

```yaml
hardware:
  # GPIO Settings (Raspberry Pi)
  enable_gpio: false              # Enable GPIO support
  ptt_gpio_pin: 18               # GPIO pin for PTT output
  status_led_pin: 16             # GPIO pin for status LED

  # OLED Display
  enable_oled: false             # Enable OLED display
  oled_i2c_address: 0x3C         # I2C address for OLED
  oled_width: 128                # OLED width in pixels
  oled_height: 64                # OLED height in pixels

  # Audio Hardware
  enable_audio: true             # Enable audio hardware
  audio_priority: "normal"       # Audio thread priority: normal, high, realtime

  # System Settings
  cpu_governor: "performance"    # CPU governor: powersave, ondemand, performance
  disable_wifi_power_save: true  # Disable WiFi power saving
```

### GPIO Pin Configuration

**Standard Pi GPIO Pins:**
```yaml
hardware:
  ptt_gpio_pin: 18      # Physical pin 12 (GPIO 18)
  status_led_pin: 16    # Physical pin 36 (GPIO 16)
  # Avoid pins: 2,3 (I2C), 14,15 (UART), 9,10,11 (SPI)
```

**Pin Mapping Reference:**
- GPIO 18 (Pin 12): PWM0, good for PTT
- GPIO 16 (Pin 36): Available, good for LED
- GPIO 20 (Pin 38): Available
- GPIO 21 (Pin 40): Available

## Database Configuration

Configure message storage and database settings.

```yaml
database:
  # Database Type
  type: "sqlite"                  # Database type: sqlite

  # SQLite Settings
  path: "data/messages.db"        # Database file path
  max_connections: 10             # Maximum connections
  max_idle_connections: 5         # Maximum idle connections
  connection_timeout: 30          # Connection timeout (seconds)

  # Message Storage
  max_messages: 10000             # Maximum stored messages
  cleanup_interval: "24h"         # Cleanup interval
  retain_days: 30                 # Days to retain messages

  # Performance Settings
  wal_mode: true                  # Enable WAL mode (better concurrency)
  synchronous: "NORMAL"           # Synchronous mode: OFF, NORMAL, FULL
  cache_size: 2000                # Cache size (pages)

  # Backup Settings
  auto_backup: true               # Enable automatic backups
  backup_interval: "24h"          # Backup interval
  backup_directory: "data/backups" # Backup directory
  max_backups: 7                  # Maximum backup files
```

## API Configuration

Configure the REST API server.

```yaml
api:
  # Server Settings
  unix_socket: "/tmp/js8d.sock"   # Unix socket path
  socket_permissions: 0660        # Socket file permissions

  # API Features
  enable_websockets: true         # Enable WebSocket support
  websocket_ping_interval: 30     # WebSocket ping interval (seconds)

  # Rate Limiting
  rate_limit_enabled: true        # Enable rate limiting
  rate_limit_requests: 100        # Requests per minute
  rate_limit_burst: 10            # Burst size

  # Logging
  log_requests: false             # Log all API requests
  log_responses: false            # Log all API responses
```

## Environment Variables

js8d supports several environment variables for configuration:

```bash
# Audio Settings
export JS8D_AUDIO_INPUT_DEVICE="hw:1,0"
export JS8D_AUDIO_OUTPUT_DEVICE="hw:1,0"
export JS8D_AUDIO_SAMPLE_RATE="48000"

# Radio Settings
export JS8D_RADIO_MODEL="311"
export JS8D_RADIO_DEVICE="/dev/ttyUSB0"
export JS8D_RADIO_BAUD_RATE="9600"

# Station Settings
export JS8D_STATION_CALLSIGN="N0CALL"
export JS8D_STATION_GRID="FN31pr"

# Web Settings
export JS8D_WEB_PORT="8080"
export JS8D_WEB_BIND_ADDRESS="0.0.0.0"

# Debug Settings
export JS8D_DEBUG="true"
export JS8D_VERBOSE="true"
export HAMLIB_DEBUG_LEVEL="0"     # Suppress hamlib debug output
```

Environment variables override configuration file settings.

## Configuration Examples

### Basic Station Setup

```yaml
# Minimal configuration for basic operation
station:
  callsign: "W1ABC"
  grid: "FN42aa"

audio:
  input_device: "hw:0,0"
  output_device: "hw:0,0"
  sample_rate: 48000

web:
  bind_address: "0.0.0.0"
  port: 8080

radio:
  use_hamlib: false  # Start without radio control
```

### Raspberry Pi with IC-7300

```yaml
station:
  callsign: "W1ABC"
  grid: "FN42aa"

audio:
  input_device: "hw:1,0"      # USB sound card
  output_device: "hw:1,0"
  sample_rate: 48000
  buffer_size: 512            # Smaller buffer for Pi

radio:
  use_hamlib: true
  model: "311"                # IC-7300
  device: "/dev/ttyUSB0"
  baud_rate: 9600
  frequency: 14078000
  ptt_method: "cat"

web:
  bind_address: "0.0.0.0"
  port: 8080

hardware:
  enable_gpio: true
  ptt_gpio_pin: 18
  status_led_pin: 16

database:
  max_connections: 5          # Reduced for Pi
  max_idle_connections: 2
```

### High-Performance Station

```yaml
station:
  callsign: "W1ABC"
  grid: "FN42aa"

audio:
  input_device: "hw:1,0"
  output_device: "hw:1,0"
  sample_rate: 48000
  buffer_size: 1024
  latency_ms: 20              # Low latency
  use_float32: true           # High precision

radio:
  use_hamlib: true
  model: "3012"               # IC-7610
  device: "/dev/ttyUSB0"
  baud_rate: 19200
  poll_interval: 500          # Fast polling

web:
  bind_address: "0.0.0.0"
  port: 8080
  enable_audio_monitoring: true

database:
  max_connections: 20
  max_idle_connections: 10
  wal_mode: true
  cache_size: 5000
```

### Secure Remote Access

```yaml
station:
  callsign: "W1ABC"
  grid: "FN42aa"

audio:
  input_device: "hw:0,0"
  output_device: "hw:0,0"

radio:
  use_hamlib: true
  model: "311"
  device: "/dev/ttyUSB0"

web:
  bind_address: "0.0.0.0"
  port: 8443                  # Non-standard port
  api_key: "your-secret-api-key"
  allowed_origins: ["https://your-domain.com"]
  enable_cors: true
```

### Development/Testing Setup

```yaml
station:
  callsign: "N0CALL"
  grid: "AA00aa"

audio:
  input_device: "default"
  output_device: "default"
  sample_rate: 24000          # Lower rate for testing

radio:
  use_hamlib: false           # No radio needed
  # Or use dummy rig
  use_hamlib: true
  model: "1"                  # Dummy rig
  device: "/dev/null"

web:
  bind_address: "127.0.0.1"   # Local only
  port: 8080

database:
  path: ":memory:"            # In-memory database for testing
```

## Configuration Validation

js8d validates configuration on startup and will report errors:

```bash
# Test configuration
js8d -config myconfig.yaml -version  # Quick validation

# Full validation with verbose output
js8d -config myconfig.yaml -verbose
```

**Common validation errors:**
- Invalid callsign format
- Invalid grid square format
- Audio device not found
- Radio model not supported
- Port already in use
- Invalid file paths

## Hot Reloading

Configuration changes can be reloaded without restarting:

```bash
# Via command line
js8ctl reload

# Via web interface
# Go to Settings page and click "Reload Configuration"

# Via API
curl -X POST http://localhost:8080/api/v1/config/reload
```

**Note:** Some settings (like bind address and port) require a full restart.

## Best Practices

1. **Start Simple**: Begin with minimal configuration and add features gradually
2. **Use Version Control**: Keep your configuration in git for easy rollback
3. **Test Changes**: Use `-verbose` flag when testing new configurations
4. **Backup Configs**: Keep backups of working configurations
5. **Document Changes**: Comment your configuration files
6. **Security**: Use API keys and appropriate bind addresses for network access
7. **Performance**: Tune buffer sizes and polling intervals for your hardware

## See Also

- [INSTALLATION.md](INSTALLATION.md) - Installation instructions
- [TROUBLESHOOTING.md](TROUBLESHOOTING.md) - Common issues and solutions
- [API.md](API.md) - REST API documentation