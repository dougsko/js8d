// js8d Settings Page JavaScript

class SettingsManager {
    constructor() {
        this.config = {};
        this.autoSaveTimeout = null;
        this.autoSaveDelay = 1000; // 1 second delay after last change
        this.init();
    }

    init() {
        this.setupEventListeners();
        this.loadConfig();
    }

    setupEventListeners() {
        // Form submission
        document.getElementById('config-form').addEventListener('submit', (e) => {
            e.preventDefault();
            this.saveConfig();
        });

        // Reload daemon button
        document.getElementById('reload-config').addEventListener('click', () => {
            this.reloadDaemon();
        });

        // Test buttons
        document.getElementById('test-cat').addEventListener('click', () => {
            this.testCAT();
        });

        document.getElementById('test-ptt').addEventListener('click', () => {
            this.testPTT();
        });

        document.getElementById('retry-radio-connection').addEventListener('click', () => {
            this.retryRadioConnection();
        });

        // File select button
        const fileSelectButton = document.querySelector('.file-select-button');
        if (fileSelectButton) {
            fileSelectButton.addEventListener('click', () => {
                this.selectSaveDirectory();
            });
        }

        // Storage statistics buttons
        document.getElementById('refresh-stats').addEventListener('click', () => {
            this.loadStorageStats();
        });

        document.getElementById('cleanup-messages').addEventListener('click', () => {
            this.cleanupMessages();
        });

        // Hamlib dependency handling
        document.getElementById('radio-use-hamlib').addEventListener('change', (e) => {
            this.handleHamlibChange(e.target.checked);
        });

        // Serial device changes no longer need to sync PTT port since they're the same

        // Auto-save functionality - setup after DOM is loaded
        this.setupAutoSave();
    }

    setupAutoSave() {
        // Wait for form to be populated before setting up auto-save
        setTimeout(() => {
            const form = document.getElementById('config-form');
            if (form) {
                // Add change listeners to all form elements
                const inputs = form.querySelectorAll('input, select, textarea');
                inputs.forEach(input => {
                    // Skip test buttons and file select buttons
                    if (input.type === 'button' || input.classList.contains('test-button') ||
                        input.classList.contains('file-select-button')) {
                        return;
                    }

                    const events = ['change', 'input'];
                    events.forEach(eventType => {
                        input.addEventListener(eventType, () => {
                            this.scheduleAutoSave();
                        });
                    });
                });

                console.log('Auto-save enabled for', inputs.length, 'form elements');
            }
        }, 500);
    }

    scheduleAutoSave() {
        // Clear existing timeout
        if (this.autoSaveTimeout) {
            clearTimeout(this.autoSaveTimeout);
        }

        // Schedule new save
        this.autoSaveTimeout = setTimeout(() => {
            this.autoSaveAndReload();
        }, this.autoSaveDelay);
    }

    async autoSaveAndReload() {
        try {
            this.showStatus('Auto-saving configuration...', 'info');

            // Save configuration
            const configData = this.collectFormData();
            const saveResponse = await fetch('/api/v1/config', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(configData),
            });

            if (!saveResponse.ok) {
                const error = await saveResponse.json();
                this.showStatus(`Auto-save failed: ${error.error}`, 'error');
                return;
            }

            // Reload daemon configuration
            const reloadResponse = await fetch('/api/v1/config/reload', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
            });

            if (reloadResponse.ok) {
                const data = await reloadResponse.json();
                if (data.warning) {
                    this.showStatus(`Auto-saved with warning: ${data.warning}`, 'error');
                } else {
                    this.showStatus('Configuration auto-saved and reloaded', 'success');
                }

                // Update local config
                this.config = configData;
            } else {
                const error = await reloadResponse.json();
                this.showStatus(`Auto-save succeeded, reload failed: ${error.error}`, 'error');
            }

        } catch (error) {
            console.error('Auto-save failed:', error);
            this.showStatus('Auto-save failed: Network error', 'error');
        }
    }

    async loadConfig() {
        try {
            const response = await fetch('/api/v1/config');
            if (!response.ok) {
                throw new Error(`HTTP ${response.status}`);
            }

            this.config = await response.json();
            await this.loadSerialDevices();
            await this.loadAudioDevices();
            this.populateForm();
            this.loadStorageStats();
            this.showStatus('Configuration loaded successfully', 'success');

        } catch (error) {
            console.error('Failed to load config:', error);
            this.showStatus('Failed to load configuration', 'error');
        }
    }

    async loadSerialDevices() {
        try {
            console.log('Loading serial devices...');
            const response = await fetch('/api/v1/serial/devices');
            if (!response.ok) {
                throw new Error(`HTTP ${response.status}`);
            }

            const data = await response.json();
            console.log('Serial devices response:', data);
            const deviceSelect = document.getElementById('radio-device');

            // Clear existing options
            deviceSelect.innerHTML = '';

            // Add an empty option
            const emptyOption = document.createElement('option');
            emptyOption.value = '';
            emptyOption.textContent = 'Select a serial device...';
            deviceSelect.appendChild(emptyOption);

            // Add serial devices
            if (data.serial_devices && data.serial_devices.length > 0) {
                data.serial_devices.forEach(device => {
                    const option = document.createElement('option');
                    option.value = device;
                    option.textContent = device;
                    deviceSelect.appendChild(option);
                });
            } else {
                // Add default devices if none found
                const defaultDevices = [
                    '/dev/ttyUSBmodem14201',
                    '/dev/ttyUSB0',
                    '/dev/ttyUSB1',
                    '/dev/ttyACM0',
                    '/dev/ttyACM1'
                ];

                defaultDevices.forEach(device => {
                    const option = document.createElement('option');
                    option.value = device;
                    option.textContent = device;
                    deviceSelect.appendChild(option);
                });
            }

        } catch (error) {
            console.error('Failed to load serial devices:', error);
            // Fall back to showing common devices
            const deviceSelect = document.getElementById('radio-device');
            deviceSelect.innerHTML = '';

            const errorOption = document.createElement('option');
            errorOption.value = '';
            errorOption.textContent = 'Error loading devices';
            deviceSelect.appendChild(errorOption);

            // Add default devices as fallback
            const defaultDevices = [
                '/dev/ttyUSBmodem14201',
                '/dev/ttyUSB0',
                '/dev/ttyUSB1',
                '/dev/ttyACM0',
                '/dev/ttyACM1'
            ];

            defaultDevices.forEach(device => {
                const option = document.createElement('option');
                option.value = device;
                option.textContent = device;
                deviceSelect.appendChild(option);
            });
        }
    }

    async loadAudioDevices() {
        try {
            console.log('Loading audio devices...');
            const response = await fetch('/api/v1/audio/devices');
            if (!response.ok) {
                throw new Error(`HTTP ${response.status}`);
            }

            const data = await response.json();
            console.log('Audio devices response:', data);

            // Update input device dropdown
            const inputSelect = document.getElementById('audio-input');
            if (inputSelect) {
                // Keep the default option and clear the rest
                const defaultOption = inputSelect.querySelector('option[value="default"]');
                inputSelect.innerHTML = '';
                if (defaultOption) {
                    inputSelect.appendChild(defaultOption);
                }

                // Add enumerated input devices
                if (data.input_devices && data.input_devices.length > 0) {
                    data.input_devices.forEach(device => {
                        const option = document.createElement('option');
                        option.value = device;
                        option.textContent = device;
                        inputSelect.appendChild(option);
                    });
                }
            }

            // Update output device dropdown
            const outputSelect = document.getElementById('audio-output');
            if (outputSelect) {
                // Keep the default option and clear the rest
                const defaultOption = outputSelect.querySelector('option[value="default"]');
                outputSelect.innerHTML = '';
                if (defaultOption) {
                    outputSelect.appendChild(defaultOption);
                }

                // Add enumerated output devices
                if (data.output_devices && data.output_devices.length > 0) {
                    data.output_devices.forEach(device => {
                        const option = document.createElement('option');
                        option.value = device;
                        option.textContent = device;
                        outputSelect.appendChild(option);
                    });
                }
            }

            // Update notification output dropdown
            const notificationSelect = document.getElementById('audio-notification-output');
            if (notificationSelect) {
                // Clear all options for notification dropdown
                notificationSelect.innerHTML = '';

                // Add enumerated output devices (since notification uses output devices)
                if (data.output_devices && data.output_devices.length > 0) {
                    data.output_devices.forEach(device => {
                        const option = document.createElement('option');
                        option.value = device;
                        option.textContent = device;
                        notificationSelect.appendChild(option);
                    });
                }
            }

            console.log('Audio devices loaded:', data);

        } catch (error) {
            console.error('Failed to load audio devices:', error);

            // Fall back to static devices
            const staticInputDevices = ['Built-in Microphone', 'USB Audio Device', 'IC-7300', 'External Microphone'];
            const staticOutputDevices = ['Built-in Output', 'USB Audio Device', 'IC-7300', 'External Speakers'];

            // Update input select with fallback
            const inputSelect = document.getElementById('audio-input');
            if (inputSelect && inputSelect.children.length <= 1) {
                staticInputDevices.forEach(device => {
                    const option = document.createElement('option');
                    option.value = device;
                    option.textContent = device;
                    inputSelect.appendChild(option);
                });
            }

            // Update output select with fallback
            const outputSelect = document.getElementById('audio-output');
            if (outputSelect && outputSelect.children.length <= 1) {
                staticOutputDevices.forEach(device => {
                    const option = document.createElement('option');
                    option.value = device;
                    option.textContent = device;
                    outputSelect.appendChild(option);
                });
            }

            // Update notification select with fallback
            const notificationSelect = document.getElementById('audio-notification-output');
            if (notificationSelect && notificationSelect.children.length === 0) {
                staticOutputDevices.forEach(device => {
                    const option = document.createElement('option');
                    option.value = device;
                    option.textContent = device;
                    notificationSelect.appendChild(option);
                });
            }
        }
    }

    populateForm() {
        // Debug log to see what config we received
        console.log('Populating form with config:', this.config);

        // Station Configuration (note capitalized field names from Go)
        this.setFormValue('station-callsign', this.config.Station?.Callsign || '');
        this.setFormValue('station-grid', this.config.Station?.Grid || '');

        // Audio Configuration
        this.setFormValue('audio-input', this.config.Audio?.InputDevice || 'default');
        this.setFormValue('audio-input-channels', this.config.Audio?.InputChannels || 'mono');
        this.setFormValue('audio-output', this.config.Audio?.OutputDevice || 'default');
        this.setFormValue('audio-output-channels', this.config.Audio?.OutputChannels || 'mono');
        this.setFormValue('audio-notification-output', this.config.Audio?.NotificationDevice || 'Built-in Output');
        this.setFormValue('audio-sample-rate', this.config.Audio?.SampleRate || 48000);
        this.setFormValue('audio-buffer-size', this.config.Audio?.BufferSize || 1024);
        this.setFormValue('audio-save-directory', this.config.Audio?.SaveDirectory || '/Users/doug/Library/Application Support/JS8Call/save');
        this.setFormValue('audio-remember-power-tx', this.config.Audio?.RememberPowerTx || false);
        this.setFormValue('audio-remember-power-tune', this.config.Audio?.RememberPowerTune || false);

        // Radio Configuration
        this.setFormValue('radio-use-hamlib', this.config.Radio?.UseHamlib || false);
        this.setFormValue('radio-model', this.config.Radio?.Model || '1');
        this.setFormValue('radio-poll-interval', this.config.Radio?.PollInterval || 1000);

        // Handle Hamlib dependency (do this after setting the values)
        this.handleHamlibChange(this.config.Radio?.UseHamlib || false);

        // CAT Control
        this.setFormValue('radio-device', this.config.Radio?.Device || '/dev/ttyUSBmodem14201');
        this.setFormValue('radio-baud-rate', this.config.Radio?.BaudRate || 115200);
        this.setRadioValue('radio.data_bits', this.config.Radio?.DataBits || 'default');
        this.setRadioValue('radio.stop_bits', this.config.Radio?.StopBits || 'default');
        this.setRadioValue('radio.handshake', this.config.Radio?.Handshake || 'default');
        this.setFormValue('radio-dtr', this.config.Radio?.DTR || 'default');
        this.setFormValue('radio-rts', this.config.Radio?.RTS || 'default');

        // PTT Configuration
        this.setRadioValue('radio.ptt_method', this.config.Radio?.PTTMethod || 'cat');
        this.setRadioValue('radio.mode', this.config.Radio?.Mode || 'data');
        this.setRadioValue('radio.tx_audio_source', this.config.Radio?.TxAudioSource || 'front');
        this.setRadioValue('radio.split_operation', this.config.Radio?.SplitOperation || 'rig');
        this.setFormValue('radio-ptt-command', this.config.Radio?.PTTCommand || '');
        this.setFormValue('radio-tx-delay', this.config.Radio?.TxDelay || 0.2);

        // Web Configuration
        this.setFormValue('web-port', this.config.Web?.Port || 8080);
        this.setFormValue('web-bind-address', this.config.Web?.BindAddress || '0.0.0.0');

        // API Configuration
        this.setFormValue('api-unix-socket', this.config.API?.UnixSocket || '/tmp/js8d.sock');

        // Hardware Configuration
        this.setFormValue('hardware-enable-gpio', this.config.Hardware?.EnableGPIO || false);
        this.setFormValue('hardware-ptt-gpio-pin', this.config.Hardware?.PTTGPIOPin || 18);
        this.setFormValue('hardware-status-led-pin', this.config.Hardware?.StatusLEDPin || 19);
        this.setFormValue('hardware-enable-oled', this.config.Hardware?.EnableOLED || false);
        this.setFormValue('hardware-oled-width', this.config.Hardware?.OLEDWidth || 128);
        this.setFormValue('hardware-oled-height', this.config.Hardware?.OLEDHeight || 64);
    }

    setFormValue(elementId, value) {
        const element = document.getElementById(elementId);
        if (!element) {
            console.warn(`Element not found: ${elementId}`);
            return;
        }

        if (element.type === 'checkbox') {
            element.checked = Boolean(value);
        } else {
            element.value = value;
        }

        // Debug log for important fields
        if (elementId.includes('callsign') || elementId.includes('grid') || elementId.includes('radio')) {
            console.log(`Set ${elementId} = ${value}`);
        }
    }

    setRadioValue(name, value) {
        const radio = document.querySelector(`input[name="${name}"][value="${value}"]`);
        if (radio) {
            radio.checked = true;
        } else {
            console.warn(`Radio button not found: ${name}=${value}`);
        }
    }

    getRadioValue(name) {
        const radio = document.querySelector(`input[name="${name}"]:checked`);
        return radio ? radio.value : null;
    }

    getFormValue(elementId) {
        const element = document.getElementById(elementId);
        if (!element) return null;

        if (element.type === 'checkbox') {
            return element.checked;
        } else if (element.type === 'number') {
            const value = element.value;
            return value === '' ? 0 : parseInt(value);
        } else if (element.tagName === 'SELECT') {
            // Check if this select contains numeric values
            const numericSelects = [
                'audio-sample-rate', 'audio-buffer-size', 'radio-baud-rate', 'radio-poll-interval',
                'web-port', 'hardware-ptt-gpio-pin', 'hardware-status-led-pin',
                'hardware-oled-width', 'hardware-oled-height'
            ];

            if (numericSelects.includes(elementId)) {
                return parseInt(element.value);
            } else {
                return element.value;
            }
        } else {
            return element.value;
        }
    }

    collectFormData() {
        return {
            station: {
                callsign: this.getFormValue('station-callsign'),
                grid: this.getFormValue('station-grid')
            },
            audio: {
                input_device: this.getFormValue('audio-input'),
                input_channels: this.getFormValue('audio-input-channels'),
                output_device: this.getFormValue('audio-output'),
                output_channels: this.getFormValue('audio-output-channels'),
                notification_device: this.getFormValue('audio-notification-output'),
                sample_rate: this.getFormValue('audio-sample-rate'),
                buffer_size: this.getFormValue('audio-buffer-size'),
                save_directory: this.getFormValue('audio-save-directory'),
                remember_power_tx: this.getFormValue('audio-remember-power-tx'),
                remember_power_tune: this.getFormValue('audio-remember-power-tune')
            },
            radio: {
                use_hamlib: this.getFormValue('radio-use-hamlib'),
                model: this.getFormValue('radio-model'),
                poll_interval: this.getFormValue('radio-poll-interval'),
                device: this.getFormValue('radio-device'),
                baud_rate: this.getFormValue('radio-baud-rate'),
                data_bits: this.getRadioValue('radio.data_bits'),
                stop_bits: this.getRadioValue('radio.stop_bits'),
                handshake: this.getRadioValue('radio.handshake'),
                dtr: this.getFormValue('radio-dtr'),
                rts: this.getFormValue('radio-rts'),
                ptt_method: this.getRadioValue('radio.ptt_method'),
                ptt_port: this.getFormValue('radio-device'), // Use serial device for PTT
                mode: this.getRadioValue('radio.mode'),
                tx_audio_source: this.getRadioValue('radio.tx_audio_source'),
                split_operation: this.getRadioValue('radio.split_operation'),
                ptt_command: this.getFormValue('radio-ptt-command'),
                tx_delay: this.getFormValue('radio-tx-delay')
            },
            web: {
                port: this.getFormValue('web-port'),
                bind_address: this.getFormValue('web-bind-address')
            },
            api: {
                unix_socket: this.getFormValue('api-unix-socket')
            },
            hardware: {
                enable_gpio: this.getFormValue('hardware-enable-gpio'),
                ptt_gpio_pin: this.getFormValue('hardware-ptt-gpio-pin'),
                status_led_pin: this.getFormValue('hardware-status-led-pin'),
                enable_oled: this.getFormValue('hardware-enable-oled'),
                oled_width: this.getFormValue('hardware-oled-width'),
                oled_height: this.getFormValue('hardware-oled-height')
            }
        };
    }

    async saveConfig() {
        try {
            this.showStatus('Saving configuration...', 'info');

            const configData = this.collectFormData();

            const response = await fetch('/api/v1/config', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(configData),
            });

            if (response.ok) {
                const data = await response.json();
                this.showStatus(`Configuration saved to ${data.path}`, 'success');
                this.config = configData; // Update local config
            } else {
                const error = await response.json();
                this.showStatus(`Failed to save: ${error.error}`, 'error');
            }

        } catch (error) {
            console.error('Failed to save config:', error);
            this.showStatus('Failed to save configuration', 'error');
        }
    }

    async reloadDaemon() {
        try {
            this.showStatus('Reloading daemon configuration...', 'info');

            const response = await fetch('/api/v1/config/reload', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
            });

            if (response.ok) {
                const data = await response.json();

                // Check for warnings
                if (data.warning) {
                    this.showStatus(`Reloaded with warning: ${data.warning}`, 'error');
                } else {
                    this.showStatus('Daemon configuration reloaded successfully', 'success');
                }

                // Show changes if any
                if (data.old_callsign !== data.new_callsign || data.old_grid !== data.new_grid) {
                    setTimeout(() => {
                        this.showStatus(
                            `Station updated: ${data.old_callsign} (${data.old_grid}) → ${data.new_callsign} (${data.new_grid})`,
                            'info'
                        );
                    }, 3000);
                }
            } else {
                const error = await response.json();
                this.showStatus(`Failed to reload: ${error.error}`, 'error');
            }

        } catch (error) {
            console.error('Failed to reload daemon:', error);
            this.showStatus('Failed to reload daemon configuration', 'error');
        }
    }

    showStatus(message, type) {
        const statusElement = document.getElementById('status-message');
        statusElement.textContent = message;
        statusElement.className = `status-message status-${type}`;
        statusElement.style.display = 'block';

        // Auto-hide success and info messages after 5 seconds
        if (type === 'success' || type === 'info') {
            setTimeout(() => {
                statusElement.style.display = 'none';
            }, 5000);
        }
    }

    async testCAT() {
        const button = document.getElementById('test-cat');
        const originalText = button.textContent;

        // Clear any existing timeouts
        if (this.catTimeout) {
            clearTimeout(this.catTimeout);
            this.catTimeout = null;
        }

        try {
            button.textContent = 'Testing...';
            button.classList.remove('success', 'error');
            button.classList.add('testing');
            button.disabled = true;

            const response = await fetch('/api/v1/radio/test-cat', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    device: this.getFormValue('radio-device'),
                    model: this.getFormValue('radio-model'),
                    baud_rate: this.getFormValue('radio-baud-rate')
                })
            });

            if (response.ok) {
                const data = await response.json();
                button.classList.remove('testing', 'error');
                button.classList.add('success');
                button.textContent = 'CAT OK';

                // Reset to original state after 3 seconds
                this.catTimeout = setTimeout(() => {
                    if (button.classList.contains('success')) {
                        button.classList.remove('success');
                        button.textContent = originalText;
                    }
                    this.catTimeout = null;
                }, 3000);
            } else {
                const error = await response.json();
                button.classList.remove('testing', 'success');
                button.classList.add('error');
                button.textContent = 'CAT Failed';

                // Show detailed error in console for debugging
                let errorMsg = error.error || 'Unknown error';
                if (errorMsg.includes('Communication timed out')) {
                    errorMsg += ' - Check serial port, baud rate, and radio model';
                } else if (errorMsg.includes('IO error')) {
                    errorMsg += ' - Verify serial device and connections';
                } else if (errorMsg.includes('failed to open')) {
                    errorMsg += ' - Check if device exists and is not in use';
                }
                console.log('CAT Test Error:', errorMsg);

                // Reset to original state after 5 seconds
                this.catTimeout = setTimeout(() => {
                    if (button.classList.contains('error')) {
                        button.classList.remove('error');
                        button.textContent = originalText;
                    }
                    this.catTimeout = null;
                }, 5000);
            }

        } catch (error) {
            console.error('CAT test failed:', error);
            button.classList.remove('testing', 'success');
            button.classList.add('error');
            button.textContent = 'CAT Failed';

            // Reset to original state after 5 seconds
            this.catTimeout = setTimeout(() => {
                if (button.classList.contains('error')) {
                    button.classList.remove('error');
                    button.textContent = originalText;
                }
                this.catTimeout = null;
            }, 5000);
        } finally {
            button.disabled = false;
        }
    }

    async testPTT() {
        const button = document.getElementById('test-ptt');
        const originalText = button.textContent;

        // Check if we're currently in TX mode
        if (button.classList.contains('tx-active')) {
            // Turn off TX
            try {
                button.textContent = 'Turning off...';
                button.disabled = true;

                const response = await fetch('/api/v1/radio/test-ptt-off', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    }
                });

                if (response.ok) {
                    button.classList.remove('tx-active', 'error');
                    button.textContent = originalText;
                    console.log('PTT turned off successfully');
                } else {
                    button.classList.add('error');
                    button.textContent = 'TX Error';
                    console.error('Failed to turn off PTT');

                    // Reset after 3 seconds but keep TX active
                    setTimeout(() => {
                        button.classList.remove('error');
                        button.textContent = 'TX ON - Click to stop';
                    }, 3000);
                }
            } catch (error) {
                console.error('PTT off failed:', error);
                button.classList.add('error');
                button.textContent = 'TX Error';

                // Reset after 3 seconds but keep TX active
                setTimeout(() => {
                    button.classList.remove('error');
                    button.textContent = 'TX ON - Click to stop';
                }, 3000);
            } finally {
                button.disabled = false;
            }
        } else {
            // Turn on TX
            try {
                button.textContent = 'Testing...';
                button.classList.remove('success', 'error', 'tx-active');
                button.classList.add('testing');
                button.disabled = true;

                const response = await fetch('/api/v1/radio/test-ptt', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({
                        method: this.getRadioValue('radio.ptt_method'),
                        port: this.getFormValue('radio-device'), // Use serial device for PTT
                        tx_delay: 0.5  // Short delay for toggle mode
                    })
                });

                if (response.ok) {
                    const data = await response.json();
                    button.classList.remove('testing', 'error');
                    button.classList.add('tx-active');
                    button.textContent = 'TX ON - Click to stop';
                    console.log('PTT activated successfully');
                } else {
                    const error = await response.json();
                    button.classList.remove('testing', 'tx-active');
                    button.classList.add('error');
                    button.textContent = 'PTT Failed';
                    console.log('PTT Test Error:', error.error);

                    // Reset to original state after 5 seconds
                    setTimeout(() => {
                        button.classList.remove('error');
                        button.textContent = originalText;
                    }, 5000);
                }

            } catch (error) {
                console.error('PTT test failed:', error);
                button.classList.remove('testing', 'tx-active');
                button.classList.add('error');
                button.textContent = 'PTT Failed';

                // Reset to original state after 5 seconds
                setTimeout(() => {
                    button.classList.remove('error');
                    button.textContent = originalText;
                }, 5000);
            } finally {
                button.disabled = false;
            }
        }
    }

    async retryRadioConnection() {
        const button = document.getElementById('retry-radio-connection');
        const originalText = button.textContent;

        try {
            button.textContent = 'Retrying...';
            button.classList.add('testing');
            button.disabled = true;

            this.showStatus('Attempting to reconnect radio...', 'info');

            const response = await fetch('/api/v1/radio/retry-connection', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
            });

            if (response.ok) {
                const data = await response.json();
                this.showStatus(`Radio Connection: ${data.message}`, 'success');
            } else {
                const error = await response.json();
                this.showStatus(`Radio Retry Failed: ${error.error}`, 'error');
            }

        } catch (error) {
            console.error('Radio retry failed:', error);
            this.showStatus('Radio retry failed: Network error', 'error');
        } finally {
            button.textContent = originalText;
            button.classList.remove('testing');
            button.disabled = false;
        }
    }

    selectSaveDirectory() {
        // This would typically open a file dialog
        // For now, show a simple prompt
        const currentPath = this.getFormValue('audio-save-directory');
        const newPath = prompt('Enter save directory path:', currentPath);
        if (newPath && newPath !== currentPath) {
            this.setFormValue('audio-save-directory', newPath);
        }
    }

    validateForm() {
        const callsign = this.getFormValue('station-callsign');
        if (!callsign || callsign.trim() === '') {
            this.showStatus('Callsign is required', 'error');
            return false;
        }

        const port = this.getFormValue('web-port');
        if (port < 1024 || port > 65535) {
            this.showStatus('Web port must be between 1024 and 65535', 'error');
            return false;
        }

        // Validate radio configuration
        if (this.getFormValue('radio-use-hamlib')) {
            const device = this.getFormValue('radio-device');
            if (!device || device.trim() === '') {
                this.showStatus('Serial device is required when using Hamlib', 'error');
                return false;
            }
        }

        return true;
    }

    async loadStorageStats() {
        try {
            const response = await fetch('/api/v1/messages/stats');
            if (!response.ok) {
                throw new Error(`HTTP ${response.status}`);
            }

            const stats = await response.json();

            // Update statistics display
            document.getElementById('stat-total-messages').textContent = stats.total_messages || 0;
            document.getElementById('stat-total-rx').textContent = stats.total_rx || 0;
            document.getElementById('stat-total-tx').textContent = stats.total_tx || 0;

            // Format last cleanup date
            let cleanupText = 'Never';
            if (stats.last_cleanup && stats.last_cleanup !== '0001-01-01T00:00:00Z') {
                const cleanupDate = new Date(stats.last_cleanup);
                cleanupText = cleanupDate.toLocaleDateString() + ' ' + cleanupDate.toLocaleTimeString();
            }
            document.getElementById('stat-last-cleanup').textContent = cleanupText;

        } catch (error) {
            console.error('Failed to load storage stats:', error);
            // Set error indicators
            const errorText = 'Error';
            document.getElementById('stat-total-messages').textContent = errorText;
            document.getElementById('stat-total-rx').textContent = errorText;
            document.getElementById('stat-total-tx').textContent = errorText;
            document.getElementById('stat-last-cleanup').textContent = errorText;
        }
    }

    async cleanupMessages() {
        if (!confirm('This will permanently delete old messages to free up space. Continue?')) {
            return;
        }

        const button = document.getElementById('cleanup-messages');
        const originalText = button.textContent;

        try {
            button.textContent = 'Cleaning...';
            button.disabled = true;

            this.showStatus('Cleaning up old messages...', 'info');

            const response = await fetch('/api/v1/messages/cleanup', {
                method: 'POST'
            });

            if (response.ok) {
                const result = await response.json();
                this.showStatus(`Cleanup completed: ${result.deleted_count || 0} messages removed`, 'success');
                // Refresh stats after cleanup
                this.loadStorageStats();
            } else {
                const error = await response.json();
                this.showStatus(`Cleanup failed: ${error.error}`, 'error');
            }

        } catch (error) {
            console.error('Cleanup failed:', error);
            this.showStatus('Cleanup failed: Network error', 'error');
        } finally {
            button.textContent = originalText;
            button.disabled = false;
        }
    }


    handleHamlibChange(useHamlib) {
        const radioModel = document.getElementById('radio-model');
        const radioDevice = document.getElementById('radio-device');
        const testCatButton = document.getElementById('test-cat');

        if (useHamlib) {
            // Enable real radio selection
            radioModel.disabled = false;
            radioDevice.disabled = false;
            if (testCatButton) testCatButton.disabled = false;

            // If currently on dummy, suggest a better default
            if (radioModel.value === '1') {
                // Don't auto-change but add a note
                this.showStatus('Hamlib enabled - select your radio model and configure the serial port', 'info');
            }
        } else {
            // Disable and force to dummy
            radioModel.disabled = true;
            radioDevice.disabled = true;
            if (testCatButton) testCatButton.disabled = true;
            radioModel.value = '1'; // Force to Hamlib Dummy
            this.showStatus('Hamlib disabled - using dummy radio for testing', 'info');
        }
    }
}

// Initialize the settings manager when the page loads
document.addEventListener('DOMContentLoaded', () => {
    window.settingsManager = new SettingsManager();
});

// Show initialization message
console.log('js8d Settings Interface loaded');