# js8d Development TODO List

*Comprehensive roadmap from current state to working MVP*

## 🎯 **Current Status: Foundation Complete**

✅ **Completed (Phase 1 Foundation)**
- [x] Project structure and build system
- [x] Go daemon with Unix domain socket architecture
- [x] Web interface with REST API polling
- [x] Command-line control tool (`js8ctl`)
- [x] Configuration management
- [x] Mock OLED driver framework
- [x] Cross-compilation support

---

## 📋 **Phase 1A: DSP Library Extraction (4-6 weeks)**

### **High Priority - Core DSP Functions**

- [ ] **Extract JS8 Normal Mode Decoder**
  - [ ] Copy `../js8call/JS8.cpp` to `libjs8dsp/js8_decoder.cpp`
  - [ ] Remove GUI dependencies (Qt includes)
  - [ ] Strip out non-Normal submodes (Fast, Turbo, Slow, Ultra)
  - [ ] Keep core BP decoder and sync detection
  - [ ] Preserve FFTW3 and Eigen math operations
  - [ ] Create minimal test harness

- [ ] **Extract Varicode Encoder/Decoder**
  - [ ] Copy `../js8call/varicode.cpp` to `libjs8dsp/varicode.cpp`
  - [ ] Remove Qt container dependencies (QVector → std::vector)
  - [ ] Keep message frame building logic
  - [ ] Preserve JS8 protocol constants
  - [ ] Test message encoding/decoding

- [ ] **Extract Audio DSP Chain**
  - [ ] Copy `../js8call/Detector.cpp` audio processing parts
  - [ ] Copy `../js8call/Modulator.cpp` audio generation parts
  - [ ] Keep 48kHz→12kHz downsampling filter
  - [ ] Preserve Eigen-based signal processing
  - [ ] Remove Qt audio device dependencies

- [ ] **Create C API Wrapper**
  - [ ] Design clean C interface for Go CGO integration
  - [ ] Implement `js8dsp_init()`, `js8dsp_cleanup()`
  - [ ] Implement `js8dsp_decode_buffer()` for audio input
  - [ ] Implement `js8dsp_encode_message()` for audio output
  - [ ] Add error handling and memory management
  - [ ] Create header file for Go import

### **Medium Priority - Build System**

- [ ] **CMake Build Configuration**
  - [ ] Create `libjs8dsp/CMakeLists.txt`
  - [ ] Configure FFTW3 dependency detection
  - [ ] Configure Eigen dependency detection
  - [ ] Configure Boost dependency detection
  - [ ] Set up cross-compilation for ARM targets
  - [ ] Generate static library for Go linking

- [ ] **Dependency Management**
  - [ ] Install FFTW3 development packages
  - [ ] Install Eigen3 development packages
  - [ ] Install Boost development packages
  - [ ] Test compilation on macOS (development)
  - [ ] Test cross-compilation for Pi Zero (ARM6)
  - [ ] Document dependency installation

### **Testing & Validation**

- [ ] **DSP Library Testing**
  - [ ] Create test audio files with known JS8 signals
  - [ ] Verify decode results match original JS8Call
  - [ ] Test encode→decode round-trip accuracy
  - [ ] Benchmark performance vs original
  - [ ] Test memory usage and leak detection

- [ ] **Integration Testing**
  - [ ] Test CGO interface from Go
  - [ ] Verify no memory leaks across Go/C boundary
  - [ ] Test concurrent decode operations
  - [ ] Test error handling edge cases

---

## 📋 **Phase 1B: Go Daemon Integration (2-3 weeks)**

### **DSP Integration**

- [ ] **CGO Integration**
  - [ ] Import C headers in Go (`import "C"`)
  - [ ] Configure CGO build flags for libjs8dsp
  - [ ] Implement Go wrapper functions
  - [ ] Add proper error handling
  - [ ] Test memory management

- [ ] **Audio System**
  - [ ] Implement ALSA audio input interface
  - [ ] Implement ALSA audio output interface
  - [ ] Create audio buffer management
  - [ ] Add configurable sample rates
  - [ ] Test audio device enumeration

- [ ] **Real Message Processing**
  - [ ] Replace mock decoder with real DSP calls
  - [ ] Replace mock encoder with real DSP calls
  - [ ] Implement audio→DSP→messages pipeline
  - [ ] Implement messages→DSP→audio pipeline
  - [ ] Add SNR calculation and frequency offset

### **Protocol Implementation**

- [ ] **JS8 Protocol Handling**
  - [ ] Implement heartbeat transmission scheduling
  - [ ] Add callsign and grid extraction
  - [ ] Implement directed message parsing
  - [ ] Add message acknowledgment logic
  - [ ] Test protocol compliance with JS8Call networks

---

## 📋 **Phase 1C: Radio Integration (2-3 weeks)**

### **Hamlib Integration**

- [ ] **Radio Control**
  - [ ] Implement Hamlib wrapper in Go
  - [ ] Add frequency get/set operations
  - [ ] Add PTT control
  - [ ] Add mode detection/setting
  - [ ] Test with common radio models

- [ ] **CAT Interface**
  - [ ] Implement serial port communication
  - [ ] Add radio auto-detection
  - [ ] Implement polling for frequency changes
  - [ ] Add error handling and reconnection
  - [ ] Test USB serial adapters

### **Timing & Coordination**

- [ ] **TX/RX Timing**
  - [ ] Implement proper PTT timing
  - [ ] Add TX audio timing coordination
  - [ ] Implement collision detection
  - [ ] Add frequency coordination
  - [ ] Test with real radios

---

## 📋 **Phase 1D: Storage & Persistence (1-2 weeks)**

### **Message Database**

- [ ] **SQLite Integration**
  - [ ] Implement message storage schema
  - [ ] Add message history queries
  - [ ] Implement conversation threading
  - [ ] Add unread message tracking
  - [ ] Test database performance

- [ ] **Configuration Persistence**
  - [ ] Save runtime frequency changes
  - [ ] Persist window/UI state
  - [ ] Save message history settings
  - [ ] Add configuration backup/restore

---

## 📋 **Phase 1E: Hardware Integration (1-3 weeks)**

### **GPIO Control (Optional for MVP)**

- [ ] **PTT Control**
  - [ ] Implement GPIO PTT output
  - [ ] Add PTT timing configuration
  - [ ] Test on Raspberry Pi hardware
  - [ ] Add safety timeouts

- [ ] **OLED Display**
  - [ ] Implement SSD1306 I2C driver
  - [ ] Create display layout manager
  - [ ] Add real-time message display
  - [ ] Test on Pi Zero hardware

### **Hardware Buttons (Optional)**

- [ ] **Control Buttons**
  - [ ] Implement GPIO button input
  - [ ] Add button debouncing
  - [ ] Map to common functions (CQ, heartbeat)
  - [ ] Test responsiveness

---

## 📋 **Phase 1F: Web Interface Enhancements (1-2 weeks)**

### **UI Polish**

- [ ] **Real-time Features**
  - [ ] Add spectrum display (audio levels)
  - [ ] Implement transmission progress bar
  - [ ] Add live frequency display
  - [ ] Show actual SNR values

- [ ] **Configuration Interface**
  - [ ] Add web-based settings page
  - [ ] Implement radio configuration UI
  - [ ] Add audio device selection
  - [ ] Test mobile responsiveness

---

## 📋 **Phase 1G: Testing & Deployment (1-2 weeks)**

### **System Testing**

- [ ] **End-to-End Testing**
  - [ ] Test complete RX→decode→display pipeline
  - [ ] Test complete compose→encode→TX pipeline
  - [ ] Verify JS8Call network compatibility
  - [ ] Test extended operation (24+ hours)

- [ ] **Performance Testing**
  - [ ] Measure CPU usage on Pi Zero
  - [ ] Test memory usage over time
  - [ ] Benchmark decode sensitivity vs JS8Call
  - [ ] Test multiple simultaneous connections

### **Documentation**

- [ ] **User Documentation**
  - [ ] Write installation guide
  - [ ] Create configuration reference
  - [ ] Document API endpoints
  - [ ] Add troubleshooting guide

- [ ] **Developer Documentation**
  - [ ] Document DSP library API
  - [ ] Create build instructions
  - [ ] Add architecture diagrams
  - [ ] Document socket protocol

### **Packaging**

- [ ] **Distribution**
  - [ ] Create installation scripts
  - [ ] Build ARM binaries (Pi Zero, Pi 4)
  - [ ] Test systemd service integration
  - [ ] Create GitHub releases

---

## 🎯 **MVP Success Criteria**

### **Must Have for Release**

- [ ] **Core Functionality**
  - [ ] Decode JS8 Normal mode signals from radio
  - [ ] Encode and transmit JS8 Normal mode signals
  - [ ] Web interface shows live decoded messages
  - [ ] Can send messages via web interface
  - [ ] Radio frequency control working
  - [ ] Stable operation for 24+ hours on Pi Zero

- [ ] **Network Compatibility**
  - [ ] Decode messages from other JS8Call stations
  - [ ] Other JS8Call stations can decode our transmissions
  - [ ] Proper callsign and grid handling
  - [ ] Message acknowledgments working

- [ ] **Deployment**
  - [ ] Single binary installation
  - [ ] Cross-compilation for Pi Zero/Pi 4
  - [ ] Basic configuration examples
  - [ ] Service integration (systemd)

### **Nice to Have (Post-MVP)**

- [ ] All JS8 submodes (Fast, Turbo, Slow, Ultra)
- [ ] Store-and-forward messaging
- [ ] Auto-reply functionality
- [ ] OLED display support
- [ ] Hardware button control
- [ ] Advanced scheduling features
- [ ] Multi-band operation
- [ ] Comprehensive logging

---

## 📊 **Estimated Timeline**

### **Aggressive Schedule (12-16 weeks total)**
- **Phase 1A (DSP):** 4 weeks
- **Phase 1B (Integration):** 2 weeks
- **Phase 1C (Radio):** 2 weeks
- **Phase 1D (Storage):** 1 week
- **Phase 1E (Hardware):** 2 weeks
- **Phase 1F (Web UI):** 1 week
- **Phase 1G (Testing):** 2 weeks

### **Conservative Schedule (16-20 weeks total)**
- **Phase 1A (DSP):** 6 weeks (complexity buffer)
- **Phase 1B (Integration):** 3 weeks
- **Phase 1C (Radio):** 3 weeks
- **Phase 1D (Storage):** 2 weeks
- **Phase 1E (Hardware):** 3 weeks
- **Phase 1F (Web UI):** 2 weeks
- **Phase 1G (Testing):** 3 weeks

---

## ⚠️ **Risk Assessment**

### **High Risk Items**
1. **DSP Extraction Complexity** - Original code is tightly coupled
2. **CGO Memory Management** - Potential for leaks and crashes
3. **Radio Timing** - PTT coordination with audio can be tricky
4. **JS8 Protocol Compliance** - Must be wire-compatible

### **Mitigation Strategies**
- Start with DSP extraction immediately (highest risk)
- Test extensively against original JS8Call
- Use proven Hamlib for radio control
- Implement comprehensive error handling

---

## 🚀 **Next Immediate Actions**

### **Week 1: Start DSP Extraction**
1. Set up FFTW3, Eigen, Boost development environment
2. Copy `JS8.cpp` and begin stripping Qt dependencies
3. Create basic CMake build system
4. Start with simple decode test

### **Week 2: Continue DSP Work**
1. Extract varicode functionality
2. Create C API wrapper skeleton
3. Test basic encode/decode operations
4. Begin CGO integration

**The foundation is solid - time to tackle the DSP challenge!** 🎯