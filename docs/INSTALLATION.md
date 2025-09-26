# js8d Installation Guide

js8d is a headless JS8Call daemon designed for Single Board Computers (SBC) with a web interface and REST API. This guide covers installation on various platforms.

## Table of Contents

- [System Requirements](#system-requirements)
- [Quick Start](#quick-start)
- [Platform-Specific Installation](#platform-specific-installation)
- [Configuration](#configuration)
- [Running js8d](#running-js8d)
- [Troubleshooting](#troubleshooting)

## System Requirements

### Minimum Requirements
- **CPU**: ARM Cortex-A7 (Pi Zero) or better, x86_64
- **RAM**: 256MB available memory
- **Storage**: 100MB free space
- **OS**: Linux (Raspberry Pi OS, Ubuntu, Debian), macOS (development)

### Recommended Requirements
- **CPU**: ARM Cortex-A72 (Pi 4) or x86_64
- **RAM**: 512MB available memory
- **Storage**: 500MB free space (for message storage)
- **Network**: Ethernet or WiFi connection for web interface

### Dependencies
- **Audio**: ALSA (Linux) or Core Audio (macOS)
- **Radio Control**: Hamlib (optional but recommended)
- **Build Tools**: Go 1.21+, GCC, CMake (for building from source)

## Quick Start

### Pre-built Binaries (Recommended)

1. **Download the latest release** for your platform:
   ```bash
   # Raspberry Pi (ARM64)
   wget https://github.com/dougsko/js8d/releases/latest/download/js8d-linux-arm64.tar.gz

   # Raspberry Pi Zero/3 (ARM)
   wget https://github.com/dougsko/js8d/releases/latest/download/js8d-linux-arm.tar.gz

   # Linux x86_64
   wget https://github.com/dougsko/js8d/releases/latest/download/js8d-linux-amd64.tar.gz

   # macOS (development/testing)
   wget https://github.com/dougsko/js8d/releases/latest/download/js8d-darwin-amd64.tar.gz
   ```

2. **Extract and install**:
   ```bash
   tar -xzf js8d-*.tar.gz
   cd js8d-*
   sudo cp js8d /usr/local/bin/
   sudo cp js8ctl /usr/local/bin/
   sudo chmod +x /usr/local/bin/js8d /usr/local/bin/js8ctl
   ```

3. **Create configuration**:
   ```bash
   mkdir -p ~/.config/js8d
   cp configs/config.example.yaml ~/.config/js8d/config.yaml
   ```

4. **Edit configuration** (see [Configuration](#configuration) section)

5. **Run js8d**:
   ```bash
   js8d -config ~/.config/js8d/config.yaml
   ```

6. **Open web interface**: http://localhost:8080

### Building from Source

1. **Install dependencies**:
   ```bash
   # Ubuntu/Debian
   sudo apt update
   sudo apt install -y golang-go gcc cmake libasound2-dev libhamlib-dev

   # Fedora/RHEL
   sudo dnf install -y golang gcc cmake alsa-lib-devel hamlib-devel

   # Arch Linux
   sudo pacman -S go gcc cmake alsa-lib hamlib

   # macOS (Homebrew)
   brew install go cmake hamlib pkg-config
   ```

2. **Clone and build**:
   ```bash
   git clone https://github.com/dougsko/js8d.git
   cd js8d
   make build
   ```

3. **Install binaries**:
   ```bash
   sudo make install
   ```

## Platform-Specific Installation

### Raspberry Pi OS

1. **Update system**:
   ```bash
   sudo apt update && sudo apt upgrade -y
   ```

2. **Install dependencies**:
   ```bash
   sudo apt install -y libasound2-dev libhamlib-dev
   ```

3. **Download and install js8d**:
   ```bash
   # For Pi 4/5 (64-bit)
   wget https://github.com/dougsko/js8d/releases/latest/download/js8d-linux-arm64.tar.gz

   # For Pi Zero/3 (32-bit)
   wget https://github.com/dougsko/js8d/releases/latest/download/js8d-linux-arm.tar.gz

   tar -xzf js8d-*.tar.gz
   cd js8d-*
   sudo cp js8d /usr/local/bin/
   sudo cp js8ctl /usr/local/bin/
   sudo chmod +x /usr/local/bin/js8d /usr/local/bin/js8ctl
   ```

4. **Create systemd service** (optional):
   ```bash
   sudo tee /etc/systemd/system/js8d.service << EOF
   [Unit]
   Description=js8d JS8Call Daemon
   After=network.target sound.target
   Wants=network.target

   [Service]
   Type=simple
   User=pi
   Group=audio
   WorkingDirectory=/home/pi
   ExecStart=/usr/local/bin/js8d -config /home/pi/.config/js8d/config.yaml
   Restart=always
   RestartSec=5

   [Install]
   WantedBy=multi-user.target
   EOF

   sudo systemctl daemon-reload
   sudo systemctl enable js8d
   ```

### Ubuntu/Debian

1. **Install dependencies**:
   ```bash
   sudo apt update
   sudo apt install -y libasound2-dev libhamlib-dev
   ```

2. **Create js8d user** (recommended):
   ```bash
   sudo useradd -r -s /bin/false -d /var/lib/js8d -c "js8d daemon" js8d
   sudo mkdir -p /var/lib/js8d/.config/js8d
   sudo mkdir -p /var/log/js8d
   sudo chown -R js8d:js8d /var/lib/js8d /var/log/js8d
   ```

3. **Install js8d**:
   ```bash
   wget https://github.com/dougsko/js8d/releases/latest/download/js8d-linux-amd64.tar.gz
   tar -xzf js8d-*.tar.gz
   cd js8d-*
   sudo cp js8d /usr/local/bin/
   sudo cp js8ctl /usr/local/bin/
   sudo chmod +x /usr/local/bin/js8d /usr/local/bin/js8ctl
   ```

4. **Configure systemd service**:
   ```bash
   sudo tee /etc/systemd/system/js8d.service << EOF
   [Unit]
   Description=js8d JS8Call Daemon
   After=network.target sound.target
   Wants=network.target

   [Service]
   Type=simple
   User=js8d
   Group=audio
   WorkingDirectory=/var/lib/js8d
   ExecStart=/usr/local/bin/js8d -config /var/lib/js8d/.config/js8d/config.yaml
   StandardOutput=journal
   StandardError=journal
   Restart=always
   RestartSec=5

   [Install]
   WantedBy=multi-user.target
   EOF

   sudo systemctl daemon-reload
   sudo systemctl enable js8d
   ```

### macOS (Development)

1. **Install Homebrew** (if not already installed):
   ```bash
   /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
   ```

2. **Install dependencies**:
   ```bash
   brew install hamlib pkg-config
   ```

3. **Install js8d**:
   ```bash
   wget https://github.com/dougsko/js8d/releases/latest/download/js8d-darwin-amd64.tar.gz
   tar -xzf js8d-*.tar.gz
   cd js8d-*
   sudo cp js8d /usr/local/bin/
   sudo cp js8ctl /usr/local/bin/
   ```

## Configuration

### Basic Configuration

1. **Copy example configuration**:
   ```bash
   mkdir -p ~/.config/js8d
   cp configs/config.example.yaml ~/.config/js8d/config.yaml
   ```

2. **Edit configuration**:
   ```bash
   nano ~/.config/js8d/config.yaml
   ```

3. **Minimum required settings**:
   ```yaml
   station:
     callsign: "YOUR_CALL"  # Replace with your amateur radio callsign
     grid: "FN31pr"         # Replace with your 6-character grid square

   audio:
     input_device: "hw:0,0"   # Your audio input device
     output_device: "hw:0,0"  # Your audio output device
     sample_rate: 48000
     buffer_size: 1024

   web:
     bind_address: "0.0.0.0"  # Listen on all interfaces
     port: 8080               # Web interface port
   ```

### Advanced Configuration

See [CONFIGURATION.md](CONFIGURATION.md) for detailed configuration options including:
- Radio control (Hamlib) setup
- Audio device configuration
- GPIO settings for Raspberry Pi
- Database and storage options
- API and security settings

## Running js8d

### Command Line Options

```bash
js8d [options]
```

**Options:**
- `-config <file>`: Configuration file path (default: config.yaml)
- `-verbose`: Enable verbose logging (includes hamlib debug output)
- `-version`: Show version information

### Starting js8d

1. **Run in foreground** (for testing):
   ```bash
   js8d -config ~/.config/js8d/config.yaml
   ```

2. **Run with verbose logging**:
   ```bash
   js8d -config ~/.config/js8d/config.yaml -verbose
   ```

3. **Run as systemd service**:
   ```bash
   sudo systemctl start js8d
   sudo systemctl status js8d
   ```

4. **View logs**:
   ```bash
   # Systemd service logs
   sudo journalctl -u js8d -f

   # Or run in foreground to see output directly
   js8d -config ~/.config/js8d/config.yaml
   ```

### Accessing the Web Interface

1. **Open your web browser** to: http://localhost:8080
2. **For remote access**: http://YOUR_PI_IP:8080
3. **Mobile-friendly**: The interface works well on tablets and phones

### Using the Command Line Client

```bash
# Check daemon status
js8ctl status

# Send a message
js8ctl send "Hello from js8d!"

# Send a message to specific station
js8ctl send -to N0CALL "Hello there!"

# Get recent messages
js8ctl messages

# Set frequency
js8ctl freq 14078000
```

## Troubleshooting

### Common Issues

1. **Audio device not found**:
   ```bash
   # List ALSA devices
   arecord -l
   aplay -l

   # Test audio
   arecord -D hw:0,0 -d 5 test.wav
   aplay -D hw:0,0 test.wav
   ```

2. **Radio connection failed**:
   ```bash
   # Test hamlib connection
   rigctl -m 1 -r /dev/ttyUSB0 f

   # List available rig models
   rigctl -l
   ```

3. **Permission errors**:
   ```bash
   # Add user to audio group
   sudo usermod -a -G audio $USER

   # Add user to dialout group (for serial devices)
   sudo usermod -a -G dialout $USER

   # Logout and login again for groups to take effect
   ```

4. **Web interface not accessible**:
   - Check if js8d is running: `ps aux | grep js8d`
   - Check port binding: `netstat -ln | grep 8080`
   - Check firewall settings
   - Try accessing via localhost first: http://localhost:8080

5. **Memory issues on Pi Zero**:
   ```yaml
   # Reduce buffer sizes in config.yaml
   audio:
     buffer_size: 512  # Reduce from 1024

   # Reduce database cache
   database:
     max_connections: 2
     max_idle_connections: 1
   ```

### Log Analysis

**Common log messages:**
- `Hardware: Audio initialized` - Audio system working
- `Hardware: Radio initialized` - Radio control working
- `js8d started successfully` - Daemon started properly
- `Audio: WARNING - ALSA audio initialization failed` - Audio problem
- `Hardware: Warning - failed to initialize radio` - Radio connection issue

### Getting Help

1. **Check logs first**: Use `-verbose` flag for detailed output
2. **Verify configuration**: Compare with `config.example.yaml`
3. **Test components individually**: Use `arecord`, `aplay`, `rigctl`
4. **Check GitHub issues**: https://github.com/dougsko/js8d/issues
5. **Create new issue**: Include logs, configuration, and system info

### Performance Tuning

**For Raspberry Pi Zero:**
```yaml
audio:
  buffer_size: 512
  sample_rate: 24000  # Lower sample rate if needed

database:
  max_connections: 2
  max_idle_connections: 1

web:
  read_timeout: 30
  write_timeout: 30
```

**For Raspberry Pi 4:**
```yaml
audio:
  buffer_size: 1024
  sample_rate: 48000

database:
  max_connections: 10
  max_idle_connections: 5
```

## Next Steps

- Read [CONFIGURATION.md](CONFIGURATION.md) for detailed configuration options
- See [API.md](API.md) for REST API documentation
- Check [TROUBLESHOOTING.md](TROUBLESHOOTING.md) for common issues
- Review [CONTRIBUTING.md](CONTRIBUTING.md) to contribute to the project

## License

js8d is licensed under the MIT License. See [LICENSE](../LICENSE) for details.