// js8d Settings Page JavaScript

class SettingsManager {
    constructor() {
        this.config = {};
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

        // File select button
        const fileSelectButton = document.querySelector('.file-select-button');
        if (fileSelectButton) {
            fileSelectButton.addEventListener('click', () => {
                this.selectSaveDirectory();
            });
        }
    }

    async loadConfig() {
        try {
            const response = await fetch('/api/v1/config');
            if (!response.ok) {
                throw new Error(`HTTP ${response.status}`);
            }

            this.config = await response.json();
            this.populateForm();
            this.showStatus('Configuration loaded successfully', 'success');

        } catch (error) {
            console.error('Failed to load config:', error);
            this.showStatus('Failed to load configuration', 'error');
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
        this.setFormValue('radio-model', this.config.Radio?.Model || '10001');
        this.setFormValue('radio-poll-interval', this.config.Radio?.PollInterval || 1000);

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
        this.setFormValue('radio-ptt-port', this.config.Radio?.PTTPort || '/dev/ttyUSBmodem14201');
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
                'audio-sample-rate', 'audio-buffer-size', 'radio-baud-rate',
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
                ptt_port: this.getFormValue('radio-ptt-port'),
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
                this.showStatus('Daemon configuration reloaded successfully', 'success');

                // Show changes if any
                if (data.old_callsign !== data.new_callsign || data.old_grid !== data.new_grid) {
                    setTimeout(() => {
                        this.showStatus(
                            `Station updated: ${data.old_callsign} (${data.old_grid}) → ${data.new_callsign} (${data.new_grid})`,
                            'info'
                        );
                    }, 2000);
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

        try {
            button.textContent = 'Testing...';
            button.classList.add('testing');
            button.disabled = true;

            this.showStatus('Testing CAT connection...', 'info');

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
                this.showStatus(`CAT Test Success: ${data.message}`, 'success');
            } else {
                const error = await response.json();
                this.showStatus(`CAT Test Failed: ${error.error}`, 'error');
            }

        } catch (error) {
            console.error('CAT test failed:', error);
            this.showStatus('CAT test failed: Network error', 'error');
        } finally {
            button.textContent = originalText;
            button.classList.remove('testing');
            button.disabled = false;
        }
    }

    async testPTT() {
        const button = document.getElementById('test-ptt');
        const originalText = button.textContent;

        try {
            button.textContent = 'Testing...';
            button.classList.add('testing');
            button.disabled = true;

            this.showStatus('Testing PTT...', 'info');

            const response = await fetch('/api/v1/radio/test-ptt', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    method: this.getRadioValue('radio.ptt_method'),
                    port: this.getFormValue('radio-ptt-port'),
                    tx_delay: this.getFormValue('radio-tx-delay')
                })
            });

            if (response.ok) {
                const data = await response.json();
                this.showStatus(`PTT Test Success: ${data.message}`, 'success');
            } else {
                const error = await response.json();
                this.showStatus(`PTT Test Failed: ${error.error}`, 'error');
            }

        } catch (error) {
            console.error('PTT test failed:', error);
            this.showStatus('PTT test failed: Network error', 'error');
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
}

// Initialize the settings manager when the page loads
document.addEventListener('DOMContentLoaded', () => {
    window.settingsManager = new SettingsManager();
});

// Show initialization message
console.log('js8d Settings Interface loaded');