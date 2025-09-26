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
        this.setFormValue('audio-output', this.config.Audio?.OutputDevice || 'default');
        this.setFormValue('audio-sample-rate', this.config.Audio?.SampleRate || 48000);
        this.setFormValue('audio-buffer-size', this.config.Audio?.BufferSize || 1024);

        // Radio Configuration
        this.setFormValue('radio-device', this.config.Radio?.Device || '');
        this.setFormValue('radio-model', this.config.Radio?.Model || 'QDX');
        this.setFormValue('radio-baud-rate', this.config.Radio?.BaudRate || 38400);
        this.setFormValue('radio-use-hamlib', this.config.Radio?.UseHamlib || false);

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
                output_device: this.getFormValue('audio-output'),
                sample_rate: this.getFormValue('audio-sample-rate'),
                buffer_size: this.getFormValue('audio-buffer-size')
            },
            radio: {
                device: this.getFormValue('radio-device'),
                model: this.getFormValue('radio-model'),
                baud_rate: this.getFormValue('radio-baud-rate'),
                use_hamlib: this.getFormValue('radio-use-hamlib')
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

        return true;
    }
}

// Initialize the settings manager when the page loads
document.addEventListener('DOMContentLoaded', () => {
    window.settingsManager = new SettingsManager();
});

// Show initialization message
console.log('js8d Settings Interface loaded');