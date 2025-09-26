# js8d - Headless JS8Call Daemon

A lightweight, headless implementation of JS8Call designed for Single Board Computers (SBC).

## Features

- **Headless Operation**: No GUI dependencies, perfect for SBC deployment
- **Web Interface**: Mobile-responsive web UI accessible from any device
- **REST API**: Complete API for external integration
- **Real-time Updates**: WebSocket interface for live message feeds
- **Single Binary**: Easy deployment with Go cross-compilation
- **Hardware Integration**: GPIO PTT control, OLED displays (planned)
- **Low Resource Usage**: Optimized for Pi Zero and similar hardware

## Quick Start

```bash
# Configure your station
cp config.example.yaml config.yaml
# Edit config.yaml with your callsign, grid, radio settings

# Run the daemon
./js8d -config config.yaml

# Access web interface
open http://localhost:8080
```

## Architecture

- **Go Daemon**: Main service with HTTP/WebSocket servers
- **C++ DSP Library**: High-performance JS8 signal processing
- **Web UI**: Mobile-first responsive interface
- **SQLite Storage**: Message history and configuration

## Development Status

**MVP Phase (Target: 6-8 weeks)**
- [ ] Basic JS8 Normal mode decode/encode
- [ ] Web interface with essential functions
- [ ] Hamlib radio control
- [ ] Message storage
- [ ] REST API endpoints
- [ ] WebSocket real-time updates

## Building

```bash
# Install dependencies
go mod download

# Build DSP library
cd libjs8dsp && mkdir build && cd build
cmake .. && make

# Build daemon
go build -o js8d cmd/js8d/main.go

# Cross-compile for Pi Zero
GOOS=linux GOARCH=arm GOARM=6 go build -o js8d-pizero cmd/js8d/main.go
```

## License

GPLv3 - Same as original JS8Call