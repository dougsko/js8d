# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

js8d is a headless JS8Call daemon designed for Single Board Computers (SBC). It implements JS8 digital mode signal processing with a web interface and REST API, optimized for resource-constrained environments like Raspberry Pi.

## Build Commands

### Go Application
```bash
# Standard build
make build

# Development build and run
make dev

# Cross-compile for different ARM platforms
make build-linux-arm64    # Pi 4
make build-linux-arm      # Pi 3
make build-linux-arm6     # Pi Zero

# Build all platforms
make build-all
```

### C++ DSP Library
```bash
# Build the DSP library (required for some components)
cd libjs8dsp && mkdir -p build && cd build
cmake .. && make

# Run DSP library tests
./js8dsp_test
```

### Testing and Quality
```bash
# Run all Go tests
make test

# Run tests with coverage
make test-coverage

# Code formatting and linting
make fmt
make vet
make lint
```

## Architecture Overview

### Hybrid Go/C++ Design
The project uses a hybrid architecture combining Go for the daemon/web services with C++ for high-performance DSP operations:

- **Go Daemon** (`cmd/js8d/`): Main service with HTTP/WebSocket servers, configuration, and hardware control
- **C++ DSP Library** (`libjs8dsp/`): High-performance JS8 signal processing with Eigen linear algebra
- **Pure Go DSP** (`pkg/dsp/`): Alternative DSP implementation using `mjibson/go-dsp` for FFT operations

### Key Components

#### DSP Processing
- **C++ Implementation**: Uses authentic algorithms extracted from JS8Call
  - `baseline_computation.cpp`: Advanced noise floor estimation with Eigen polynomial fitting
  - `bp_decoder.cpp`: LDPC belief propagation forward error correction
  - `varicode.cpp`: Huffman varicode encoding/decoding with frequency optimization
  - `js8_decoder.cpp`: Complete JS8 decode pipeline with Costas synchronization
- **Go Implementation**: Pure Go alternative using `github.com/mjibson/go-dsp/fft`

#### Hardware Abstraction
- **Cross-platform audio**: ALSA (Linux), Core Audio (macOS), fallback implementations
- **Radio control**: Hamlib integration for CAT control
- **GPIO support**: PTT control and status LEDs on Raspberry Pi

#### Web Interface
- **RESTful API**: Complete API for external integration
- **WebSocket**: Real-time message updates
- **Mobile-responsive UI**: Optimized for small screens

### Package Structure
- `pkg/config/`: YAML configuration management
- `pkg/hardware/`: Cross-platform hardware abstraction (audio, radio, GPIO)
- `pkg/dsp/`: Pure Go DSP implementation
- `pkg/protocol/`: JS8 protocol implementation
- `pkg/storage/`: SQLite message storage
- `cmd/js8d/`: Main daemon entry point
- `cmd/js8ctl/`: Command-line client tool
- `cmd/js8encode/`: Standalone encoding utility

## Configuration

The project uses YAML configuration with platform-specific defaults:

```bash
# Copy and customize configuration
cp configs/config.example.yaml config.yaml
```

Key configuration sections:
- `station`: Callsign and grid square
- `radio`: Serial device, model, baud rate, PTT method
- `audio`: ALSA/Core Audio device configuration
- `web`: HTTP server binding and ports
- `hardware`: GPIO pin assignments for Pi deployment

## Development Notes

### Dependencies
- **Go 1.21+** required
- **Eigen3** (optional): For advanced baseline computation in C++ DSP library
- **Boost** (optional): Used by C++ components
- **CMake 3.14+**: For C++ library build
- **ALSA development libraries** (Linux): For audio support

### DSP Library Integration
The C++ DSP library provides authentic JS8Call algorithms:
- Requires Eigen3 for polynomial fitting (falls back to simpler implementation without)
- Built as static library linked via CGO
- Includes comprehensive test suite

### Cross-Platform Considerations
- Audio system detection at runtime (ALSA vs Core Audio vs fallback)
- GPIO features require root privileges on Raspberry Pi
- ARM cross-compilation supported for Pi deployment

### Testing
- Go tests cover protocol, configuration, and hardware abstraction
- C++ tests validate DSP algorithms and round-trip encoding
- Integration tests require actual audio hardware

### Hardware-Specific Features
- **Raspberry Pi**: GPIO PTT control, status LEDs, optimized for Pi Zero
- **General Linux**: ALSA audio, Hamlib radio control
- **macOS**: Core Audio support for development

The codebase is designed for headless operation on SBCs while maintaining development-friendly features on desktop platforms.