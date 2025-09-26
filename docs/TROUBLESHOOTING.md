# js8d Troubleshooting Guide

This guide covers common issues and solutions when using js8d.

## Table of Contents

- [Quick Diagnostics](#quick-diagnostics)
- [Audio Issues](#audio-issues)
- [Radio Control Issues](#radio-control-issues)
- [Web Interface Issues](#web-interface-issues)
- [Performance Issues](#performance-issues)
- [Network and Connectivity](#network-and-connectivity)
- [Platform-Specific Issues](#platform-specific-issues)
- [Error Messages](#error-messages)
- [Debug and Logging](#debug-and-logging)
- [Getting Help](#getting-help)

## Quick Diagnostics

### Basic Health Check

1. **Check if js8d is running**:
   ```bash
   ps aux | grep js8d
   # Or check with systemd
   sudo systemctl status js8d
   ```

2. **Test web interface**:
   ```bash
   curl http://localhost:8080/api/v1/health
   ```

3. **Check logs**:
   ```bash
   # If running as systemd service
   sudo journalctl -u js8d -f

   # If running in foreground
   js8d -config config.yaml -verbose
   ```

4. **Verify configuration**:
   ```bash
   js8d -config config.yaml -version
   ```

### System Requirements Check

```bash
# Check Go version (if building from source)
go version  # Should be 1.21+

# Check available memory
free -m

# Check disk space
df -h

# Check audio system
aplay -l    # List playback devices
arecord -l  # List recording devices

# Check USB devices (for radio)
lsusb
```

## Audio Issues

### Audio Device Not Found

**Symptoms:**
- "Audio device not found" errors
- No audio input/output
- ALSA errors in logs

**Solutions:**

1. **List available devices**:
   ```bash
   # ALSA devices
   aplay -l    # Playback
   arecord -l  # Recording

   # Test devices
   arecord -D hw:0,0 -d 5 test.wav
   aplay -D hw:0,0 test.wav
   ```

2. **Check device permissions**:
   ```bash
   # Add user to audio group
   sudo usermod -a -G audio $USER

   # Check current groups
   groups

   # Logout and login for changes to take effect
   ```

3. **Fix common device names**:
   ```yaml
   # config.yaml - try different formats
   audio:
     input_device: "hw:0,0"      # Hardware device
     output_device: "plughw:0,0" # Hardware with format conversion
     # Or try:
     input_device: "default"     # System default
     output_device: "pulse"      # PulseAudio
   ```

4. **USB audio device issues**:
   ```bash
   # Check if USB device is detected
   lsusb | grep -i audio

   # Check kernel messages
   dmesg | grep -i audio

   # Restart ALSA
   sudo alsa force-reload
   ```

### Audio Quality Issues

**Symptoms:**
- Distorted audio
- Poor decoding performance
- High CPU usage

**Solutions:**

1. **Adjust buffer sizes**:
   ```yaml
   audio:
     buffer_size: 1024    # Try 512, 1024, 2048
     sample_rate: 48000   # Try 24000 for Pi Zero
   ```

2. **Check audio levels**:
   ```bash
   # Monitor audio levels
   alsamixer

   # Or use amixer
   amixer sget Master
   amixer sset Master 50%
   ```

3. **Reduce latency**:
   ```yaml
   audio:
     latency_ms: 50      # Lower for better performance
     use_float32: false  # Use 16-bit for lower CPU
   ```

### No Audio on Raspberry Pi

**Common Issues:**

1. **Wrong audio output**:
   ```bash
   # Force audio to 3.5mm jack
   sudo raspi-config
   # Advanced Options > Audio > Force 3.5mm

   # Or via command line
   amixer cset numid=3 1
   ```

2. **HDMI audio conflict**:
   ```bash
   # Disable HDMI audio in /boot/config.txt
   sudo nano /boot/config.txt
   # Add: hdmi_drive=2
   ```

3. **USB audio priority**:
   ```bash
   # Create /etc/modprobe.d/alsa-base.conf
   echo "options snd-usb-audio index=0" | sudo tee /etc/modprobe.d/alsa-base.conf
   ```

## Radio Control Issues

### Radio Not Connecting

**Symptoms:**
- "Radio connection failed" messages
- PTT not working
- Frequency changes ignored

**Solutions:**

1. **Check serial connection**:
   ```bash
   # List serial devices
   ls -la /dev/tty*

   # Check USB serial devices
   lsusb | grep -i serial

   # Test serial connection
   minicom -D /dev/ttyUSB0
   ```

2. **Verify Hamlib model**:
   ```bash
   # List all rig models
   rigctl -l | grep -i "your_radio"

   # Test specific model
   rigctl -m 311 -r /dev/ttyUSB0 f
   ```

3. **Check permissions**:
   ```bash
   # Add user to dialout group
   sudo usermod -a -G dialout $USER

   # Check device permissions
   ls -la /dev/ttyUSB0

   # Fix permissions if needed
   sudo chmod 666 /dev/ttyUSB0
   ```

4. **Configuration troubleshooting**:
   ```yaml
   radio:
     use_hamlib: true
     model: "311"              # IC-7300
     device: "/dev/ttyUSB0"    # Check actual device
     baud_rate: 9600           # Check radio manual
     timeout: 5000             # Increase if needed
     retry_count: 3
   ```

### PTT Issues

**Symptoms:**
- PTT not activating
- Radio not transmitting
- "PTT failed" errors

**Solutions:**

1. **Test PTT methods**:
   ```yaml
   radio:
     ptt_method: "cat"    # Try: cat, dtr, rts, vox
   ```

2. **GPIO PTT (Raspberry Pi)**:
   ```yaml
   hardware:
     enable_gpio: true
     ptt_gpio_pin: 18    # Physical pin 12
   ```

3. **Manual PTT testing**:
   ```bash
   # Test with rigctl
   rigctl -m 311 -r /dev/ttyUSB0 T 1  # PTT on
   rigctl -m 311 -r /dev/ttyUSB0 T 0  # PTT off
   ```

### Frequency Issues

**Symptoms:**
- Frequency not changing
- Wrong frequency displayed
- "Frequency out of range" errors

**Solutions:**

1. **Check frequency limits**:
   ```bash
   # Get rig capabilities
   rigctl -m 311 -r /dev/ttyUSB0 dump_caps
   ```

2. **Test frequency setting**:
   ```bash
   # Set frequency manually
   rigctl -m 311 -r /dev/ttyUSB0 F 14078000

   # Get current frequency
   rigctl -m 311 -r /dev/ttyUSB0 f
   ```

## Web Interface Issues

### Cannot Access Web Interface

**Symptoms:**
- Browser shows "connection refused"
- Timeout errors
- 404 errors

**Solutions:**

1. **Check service status**:
   ```bash
   # Check if js8d is running
   ps aux | grep js8d

   # Check port binding
   netstat -ln | grep 8080
   sudo ss -tlnp | grep 8080
   ```

2. **Test local access first**:
   ```bash
   curl http://localhost:8080/
   curl http://127.0.0.1:8080/api/v1/health
   ```

3. **Check firewall**:
   ```bash
   # Ubuntu/Debian
   sudo ufw status
   sudo ufw allow 8080

   # CentOS/RHEL
   sudo firewall-cmd --list-ports
   sudo firewall-cmd --add-port=8080/tcp --permanent
   sudo firewall-cmd --reload
   ```

4. **Configuration issues**:
   ```yaml
   web:
     bind_address: "0.0.0.0"  # Listen on all interfaces
     port: 8080               # Check port availability
   ```

### WebSocket Connection Issues

**Symptoms:**
- Spectrum display not working
- No real-time updates
- WebSocket connection errors

**Solutions:**

1. **Check WebSocket endpoint**:
   ```javascript
   // Test WebSocket connection
   const ws = new WebSocket('ws://localhost:8080/ws/audio');
   ws.onopen = () => console.log('Connected');
   ws.onerror = (error) => console.error('Error:', error);
   ```

2. **Browser developer tools**:
   - Open browser dev tools (F12)
   - Check Console for JavaScript errors
   - Check Network tab for WebSocket connections

3. **Proxy/reverse proxy issues**:
   ```nginx
   # Nginx configuration for WebSocket
   location /ws/ {
       proxy_pass http://localhost:8080;
       proxy_http_version 1.1;
       proxy_set_header Upgrade $http_upgrade;
       proxy_set_header Connection "upgrade";
   }
   ```

### API Authentication Issues

**Symptoms:**
- 401 Unauthorized errors
- API key rejected

**Solutions:**

1. **Check API key configuration**:
   ```yaml
   web:
     api_key: "your-secret-key"
   ```

2. **Test API key**:
   ```bash
   curl -H "Authorization: Bearer your-secret-key" \
        http://localhost:8080/api/v1/status
   ```

## Performance Issues

### High CPU Usage

**Symptoms:**
- System sluggish
- Audio dropouts
- High load average

**Solutions:**

1. **Check system resources**:
   ```bash
   top
   htop
   iostat 5
   ```

2. **Optimize configuration**:
   ```yaml
   audio:
     buffer_size: 512      # Smaller buffers
     sample_rate: 24000    # Lower sample rate
     use_float32: false    # 16-bit audio

   database:
     max_connections: 2    # Reduce for Pi
     max_idle_connections: 1
   ```

3. **System optimization**:
   ```bash
   # Set CPU governor (Pi)
   echo performance | sudo tee /sys/devices/system/cpu/cpu0/cpufreq/scaling_governor

   # Disable WiFi power save
   iwconfig wlan0 power off
   ```

### Memory Issues

**Symptoms:**
- Out of memory errors
- System freezing
- Swap usage high

**Solutions:**

1. **Check memory usage**:
   ```bash
   free -m
   cat /proc/meminfo
   ```

2. **Reduce memory usage**:
   ```yaml
   database:
     max_messages: 1000    # Reduce from 10000
     cache_size: 500       # Reduce cache

   web:
     max_request_size: 65536  # Reduce from 1MB
   ```

3. **Add swap (if needed)**:
   ```bash
   # Create swap file
   sudo fallocate -l 1G /swapfile
   sudo chmod 600 /swapfile
   sudo mkswap /swapfile
   sudo swapon /swapfile
   ```

### Poor Decoding Performance

**Symptoms:**
- Few messages decoded
- Low SNR readings
- Missed transmissions

**Solutions:**

1. **Check audio levels**:
   ```bash
   # Monitor input levels
   alsamixer
   ```

2. **Adjust DSP settings**:
   ```yaml
   audio:
     auto_gain: true       # Enable AGC
     input_gain: 0.8       # Adjust input gain
   ```

3. **Frequency accuracy**:
   - Ensure radio is on correct frequency (14.078 MHz for 20m)
   - Check frequency calibration
   - Verify antenna system

## Network and Connectivity

### Remote Access Issues

**Symptoms:**
- Cannot access from other devices
- Connection timeouts
- DNS issues

**Solutions:**

1. **Check network connectivity**:
   ```bash
   # Test from remote device
   ping YOUR_PI_IP
   telnet YOUR_PI_IP 8080
   ```

2. **Verify binding**:
   ```yaml
   web:
     bind_address: "0.0.0.0"  # Not 127.0.0.1
     port: 8080
   ```

3. **Check router/firewall**:
   - Port forwarding if needed
   - Router firewall settings
   - ISP blocking

### SSL/HTTPS Issues

**Symptoms:**
- Certificate errors
- Mixed content warnings
- Secure connection failed

**Solutions:**

1. **Use reverse proxy**:
   ```nginx
   # Nginx with Let's Encrypt
   server {
       listen 443 ssl;
       server_name your-domain.com;

       ssl_certificate /path/to/cert.pem;
       ssl_certificate_key /path/to/key.pem;

       location / {
           proxy_pass http://localhost:8080;
       }
   }
   ```

2. **Self-signed certificates**:
   ```bash
   # Create self-signed cert
   openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -days 365 -nodes
   ```

## Platform-Specific Issues

### Raspberry Pi Issues

1. **SD card corruption**:
   ```bash
   # Check filesystem
   sudo fsck /dev/mmcblk0p1

   # Check SD card health
   sudo badblocks -v /dev/mmcblk0
   ```

2. **Power supply issues**:
   - Check for lightning bolt icon (undervoltage)
   - Use quality power supply (5V 3A for Pi 4)
   - Check USB cable quality

3. **Temperature throttling**:
   ```bash
   # Check temperature
   vcgencmd measure_temp

   # Check throttling
   vcgencmd get_throttled
   ```

### macOS Issues

1. **Permission issues**:
   ```bash
   # Grant microphone access
   # System Preferences > Security & Privacy > Microphone
   ```

2. **Hamlib installation**:
   ```bash
   # Install via Homebrew
   brew install hamlib

   # Check installation
   brew list hamlib
   ```

### Linux Distribution Issues

1. **Package dependencies**:
   ```bash
   # Ubuntu/Debian
   sudo apt install libasound2-dev libhamlib-dev

   # CentOS/RHEL
   sudo dnf install alsa-lib-devel hamlib-devel

   # Arch
   sudo pacman -S alsa-lib hamlib
   ```

2. **SystemD service issues**:
   ```bash
   # Check service status
   sudo systemctl status js8d

   # Check service logs
   sudo journalctl -u js8d -f

   # Reload service
   sudo systemctl daemon-reload
   sudo systemctl restart js8d
   ```

## Error Messages

### Common Error Messages and Solutions

**"Audio device not found"**
- Check device name in configuration
- Verify device exists: `aplay -l`
- Check permissions: add user to audio group

**"Radio connection failed"**
- Verify serial device exists: `ls /dev/tty*`
- Check baud rate matches radio settings
- Add user to dialout group

**"bind: address already in use"**
- Another service using port 8080
- Find process: `sudo netstat -tlnp | grep 8080`
- Kill process or change port in config

**"permission denied"**
- User not in required groups (audio, dialout)
- Serial device permissions
- Run with sudo (not recommended) or fix permissions

**"frequency out of range"**
- Check radio specifications
- Verify rig model in configuration
- Use rigctl to test frequency range

**"message too long"**
- JS8 messages limited to 80 characters
- Check message preprocessing
- Split long messages

**"configuration validation failed"**
- Invalid YAML syntax
- Missing required fields (callsign, grid)
- Invalid values (frequency, device names)

### Log Analysis

**Enable verbose logging**:
```bash
js8d -config config.yaml -verbose
```

**Common log patterns**:
```
Hardware: Audio initialized     → Audio system working
Hardware: Radio initialized     → Radio control working
js8d started successfully       → Daemon started properly
WARNING - ALSA initialization   → Audio device problem
failed to initialize radio      → Radio connection issue
WebSocket connection closed     → Client disconnected
```

## Debug and Logging

### Enabling Debug Output

1. **Verbose mode**:
   ```bash
   js8d -config config.yaml -verbose
   ```

2. **Hamlib debugging**:
   ```bash
   export HAMLIB_DEBUG_LEVEL=1
   js8d -config config.yaml
   ```

3. **ALSA debugging**:
   ```bash
   export ALSA_DEBUG=1
   js8d -config config.yaml
   ```

### Log File Analysis

**SystemD logs**:
```bash
# Follow logs in real-time
sudo journalctl -u js8d -f

# Show recent logs
sudo journalctl -u js8d --since "1 hour ago"

# Export logs
sudo journalctl -u js8d > js8d.log
```

**Application logs**:
```bash
# Redirect to file
js8d -config config.yaml -verbose > js8d.log 2>&1

# Monitor log file
tail -f js8d.log
```

### Performance Monitoring

```bash
# Monitor system resources
htop

# Check I/O wait
iostat -x 1

# Network connections
ss -tuln

# Audio system status
cat /proc/asound/cards
cat /proc/asound/devices
```

## Getting Help

### Before Asking for Help

1. **Check this troubleshooting guide**
2. **Enable verbose logging**
3. **Collect system information**:
   ```bash
   # System info
   uname -a
   cat /etc/os-release

   # js8d version
   js8d -version

   # Hardware info
   lscpu
   free -m
   lsusb

   # Audio devices
   aplay -l > audio_devices.txt
   ```

4. **Test basic functionality**:
   ```bash
   # Test audio
   arecord -D hw:0,0 -d 5 test.wav
   aplay -D hw:0,0 test.wav

   # Test radio
   rigctl -m 311 -r /dev/ttyUSB0 f

   # Test web interface
   curl http://localhost:8080/api/v1/health
   ```

### Where to Get Help

1. **GitHub Issues**: https://github.com/dougsko/js8d/issues
   - Search existing issues first
   - Include system information
   - Attach relevant logs
   - Describe steps to reproduce

2. **Amateur Radio Forums**:
   - QRZ Forums
   - Reddit r/amateurradio
   - eHam.net

3. **Documentation**:
   - [INSTALLATION.md](INSTALLATION.md) - Setup instructions
   - [CONFIGURATION.md](CONFIGURATION.md) - Configuration options
   - [API.md](API.md) - API documentation

### Creating a Good Bug Report

**Include:**
- js8d version (`js8d -version`)
- Operating system and version
- Hardware details (Pi model, radio, audio device)
- Configuration file (remove sensitive info)
- Complete error messages
- Steps to reproduce
- What you expected vs. what happened

**Example:**
```
**js8d Version:** 1.0.0 (commit abc123)
**OS:** Raspberry Pi OS Lite (Debian 11)
**Hardware:** Pi 4B, IC-7300, USB sound card

**Issue:** Radio frequency not changing via web interface

**Configuration:**
```yaml
radio:
  use_hamlib: true
  model: "311"
  device: "/dev/ttyUSB0"
  baud_rate: 9600
```

**Steps to reproduce:**
1. Start js8d with above config
2. Open web interface
3. Change frequency from 14078000 to 14080000
4. Frequency remains unchanged

**Expected:** Radio frequency should change
**Actual:** No change, no error message

**Logs:**
```
[log output here]
```
```

This helps maintainers quickly understand and fix issues.