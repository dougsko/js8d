// js8d Web Interface JavaScript - REST API Only (No WebSocket)

class JS8DClient {
    constructor() {
        this.connected = false;
        this.messages = [];
        this.pollInterval = 2000; // Poll every 2 seconds
        this.statusInterval = 10000; // Update status every 10 seconds

        this.init();
    }

    init() {
        this.setupEventListeners();
        this.startPolling();
        this.updateStatus();
    }

    setupEventListeners() {
        // Send message button
        document.getElementById('send-message').addEventListener('click', () => {
            this.sendMessage();
        });

        // Send heartbeat button
        document.getElementById('send-heartbeat').addEventListener('click', () => {
            this.sendHeartbeat();
        });

        // Send CQ button
        document.getElementById('send-cq').addEventListener('click', () => {
            this.sendCQ();
        });

        // Abort transmission button
        document.getElementById('abort-tx').addEventListener('click', () => {
            this.abortTransmission();
        });

        // Enter key in message input
        document.getElementById('message-text').addEventListener('keypress', (e) => {
            if (e.key === 'Enter') {
                this.sendMessage();
            }
        });

        // Frequency change
        document.getElementById('frequency').addEventListener('change', (e) => {
            this.setFrequency(parseInt(e.target.value));
        });

        // Spectrum display toggle
        document.getElementById('toggle-spectrum').addEventListener('click', () => {
            this.toggleSpectrum();
        });

        // Initialize spectrum display
        this.initSpectrumDisplay();
    }

    initSpectrumDisplay() {
        this.spectrumActive = false;
        this.spectrumWebSocket = null;
        this.spectrumCanvas = document.getElementById('main-spectrum-canvas');
        this.waterfallCanvas = document.getElementById('main-waterfall-canvas');
        this.spectrumCtx = this.spectrumCanvas?.getContext('2d');
        this.waterfallCtx = this.waterfallCanvas?.getContext('2d');

        // Waterfall history buffer
        this.waterfallHistory = [];
        this.maxWaterfallLines = 100;

        // Transmission progress tracking
        this.transmissionProgress = {
            active: false,
            startTime: 0,
            estimatedDuration: 0,
            messageLength: 0
        };
    }

    toggleSpectrum() {
        const button = document.getElementById('toggle-spectrum');
        if (this.spectrumActive) {
            this.stopSpectrum();
            button.textContent = 'Start Display';
            button.classList.remove('active');
        } else {
            this.startSpectrum();
            button.textContent = 'Stop Display';
            button.classList.add('active');
        }
    }

    startSpectrum() {
        if (this.spectrumActive) return;

        this.spectrumActive = true;
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = `${protocol}//${window.location.host}/ws/audio`;

        this.spectrumWebSocket = new WebSocket(wsUrl);

        this.spectrumWebSocket.onopen = () => {
            console.log('Spectrum WebSocket connected');
        };

        this.spectrumWebSocket.onmessage = (event) => {
            const data = JSON.parse(event.data);
            console.log('Spectrum data received:', {
                type: data.type,
                timestamp: data.timestamp,
                spectrum_bins: data.spectrum?.bins?.length || 0,
                sample_rate: data.sample_rate
            });
            this.updateSpectrum(data);
        };

        this.spectrumWebSocket.onclose = () => {
            console.log('Spectrum WebSocket closed');
            if (this.spectrumActive) {
                // Only update UI if we didn't intentionally close it
                this.spectrumActive = false;
                const button = document.getElementById('toggle-spectrum');
                button.textContent = 'Start Display';
                button.classList.remove('active');
            }
        };

        this.spectrumWebSocket.onerror = (error) => {
            console.error('Spectrum WebSocket error:', error);
        };
    }

    stopSpectrum() {
        if (!this.spectrumActive) return;

        this.spectrumActive = false;
        if (this.spectrumWebSocket) {
            this.spectrumWebSocket.close();
            this.spectrumWebSocket = null;
        }
    }

    updateSpectrum(data) {
        if (!this.spectrumCtx || !this.waterfallCtx) return;

        // Update spectrum display
        this.drawSpectrum(data.spectrum);

        // Update waterfall display
        if (document.getElementById('waterfall-enabled').checked) {
            this.drawWaterfall(data.spectrum);
        }
    }

    drawSpectrum(spectrumData) {
        const canvas = this.spectrumCanvas;
        const ctx = this.spectrumCtx;
        const width = canvas.width;
        const height = canvas.height;

        // Clear canvas
        ctx.fillStyle = '#1a1a1a';
        ctx.fillRect(0, 0, width, height);

        if (!spectrumData || !spectrumData.bins || spectrumData.bins.length === 0) {
            return;
        }

        // Draw spectrum
        ctx.strokeStyle = '#4CAF50';
        ctx.lineWidth = 1;
        ctx.beginPath();

        const binWidth = width / spectrumData.bins.length;
        for (let i = 0; i < spectrumData.bins.length; i++) {
            const x = i * binWidth;
            const magnitude = Math.max(0, Math.min(1, spectrumData.bins[i]));
            const y = height - (magnitude * height);

            if (i === 0) {
                ctx.moveTo(x, y);
            } else {
                ctx.lineTo(x, y);
            }
        }
        ctx.stroke();

        // Draw frequency grid lines
        ctx.strokeStyle = '#333';
        ctx.lineWidth = 0.5;
        for (let i = 1; i < 10; i++) {
            const x = (width / 10) * i;
            ctx.beginPath();
            ctx.moveTo(x, 0);
            ctx.lineTo(x, height);
            ctx.stroke();
        }

        // Draw JS8 frequency markers (centered around 1500 Hz)
        ctx.strokeStyle = '#FF9800';
        ctx.lineWidth = 2;
        const js8Center = width / 2; // Assuming 1500 Hz is center
        ctx.beginPath();
        ctx.moveTo(js8Center, 0);
        ctx.lineTo(js8Center, height);
        ctx.stroke();
    }

    drawWaterfall(spectrumData) {
        const canvas = this.waterfallCanvas;
        const ctx = this.waterfallCtx;
        const width = canvas.width;
        const height = canvas.height;

        if (!spectrumData || !spectrumData.bins || spectrumData.bins.length === 0) {
            return;
        }

        // Shift existing waterfall data down
        if (this.waterfallHistory.length >= this.maxWaterfallLines) {
            this.waterfallHistory.shift();
        }
        this.waterfallHistory.push(spectrumData.bins);

        // Clear canvas
        ctx.fillStyle = '#1a1a1a';
        ctx.fillRect(0, 0, width, height);

        // Draw waterfall
        const lineHeight = height / this.maxWaterfallLines;
        for (let y = 0; y < this.waterfallHistory.length; y++) {
            const line = this.waterfallHistory[y];
            const imageData = ctx.createImageData(width, Math.ceil(lineHeight));

            for (let x = 0; x < width; x++) {
                const binIndex = Math.floor((x / width) * line.length);
                const magnitude = Math.max(0, Math.min(1, line[binIndex] || 0));

                // Convert magnitude to color (blue -> green -> yellow -> red)
                let r, g, b;
                if (magnitude < 0.25) {
                    r = 0;
                    g = 0;
                    b = Math.floor(magnitude * 4 * 255);
                } else if (magnitude < 0.5) {
                    r = 0;
                    g = Math.floor((magnitude - 0.25) * 4 * 255);
                    b = 255;
                } else if (magnitude < 0.75) {
                    r = Math.floor((magnitude - 0.5) * 4 * 255);
                    g = 255;
                    b = 255 - Math.floor((magnitude - 0.5) * 4 * 255);
                } else {
                    r = 255;
                    g = 255 - Math.floor((magnitude - 0.75) * 4 * 255);
                    b = 0;
                }

                for (let py = 0; py < Math.ceil(lineHeight); py++) {
                    const pixelIndex = (py * width + x) * 4;
                    imageData.data[pixelIndex] = r;     // R
                    imageData.data[pixelIndex + 1] = g; // G
                    imageData.data[pixelIndex + 2] = b; // B
                    imageData.data[pixelIndex + 3] = 255; // A
                }
            }

            ctx.putImageData(imageData, 0, y * lineHeight);
        }
    }

    startPolling() {
        // Poll for new messages
        setInterval(async () => {
            await this.loadMessages();
        }, this.pollInterval);

        // Poll for status updates
        setInterval(async () => {
            await this.updateStatus();
        }, this.statusInterval);

        // Initial load
        this.loadMessages();
    }

    async loadMessages() {
        try {
            const response = await fetch('/api/v1/messages?limit=20');
            if (!response.ok) {
                throw new Error(`HTTP ${response.status}`);
            }

            const data = await response.json();

            if (data.messages) {
                // Check for new messages
                const newMessages = data.messages.filter(msg =>
                    !this.messages.find(existing => existing.id === msg.id)
                );

                // Add new messages to display
                newMessages.forEach(msg => {
                    this.addMessage(msg, 'rx');
                });

                // Update message list
                this.messages = data.messages;

                // Update message count
                document.getElementById('message-count').textContent = `${data.count} messages`;
            }

            // Update connection status
            if (!this.connected) {
                this.connected = true;
                this.updateConnectionStatus();
            }

        } catch (error) {
            console.error('Failed to load messages:', error);

            if (this.connected) {
                this.connected = false;
                this.updateConnectionStatus();
            }
        }
    }

    async updateStatus() {
        try {
            const response = await fetch('/api/v1/status');
            if (!response.ok) {
                throw new Error(`HTTP ${response.status}`);
            }

            const data = await response.json();
            this.updateStatusFromData(data);

        } catch (error) {
            console.error('Failed to get status:', error);
        }
    }

    updateStatusFromData(data) {
        if (data.frequency) {
            document.getElementById('frequency').value = data.frequency;
        }
        if (data.status) {
            document.getElementById('daemon-status').textContent = data.status;
        }
        if (data.ptt !== undefined) {
            this.updatePTTStatus(data.ptt);
        }
        if (data.connected !== undefined) {
            // Update any connection indicators
        }
    }

    updateConnectionStatus() {
        const statusElement = document.getElementById('connection-status');
        if (this.connected) {
            statusElement.textContent = 'Connected';
            statusElement.className = 'connected';
        } else {
            statusElement.textContent = 'Disconnected';
            statusElement.className = 'disconnected';
        }
    }

    updatePTTStatus(ptt) {
        const pttElement = document.getElementById('ptt-indicator');
        if (ptt) {
            pttElement.textContent = 'PTT: ON';
            pttElement.className = 'ptt-on';
        } else {
            pttElement.textContent = 'PTT: OFF';
            pttElement.className = 'ptt-off';
        }
    }

    addMessage(msg, type) {
        const messagesContainer = document.getElementById('messages');
        const messageElement = document.createElement('div');
        messageElement.className = `message ${type}`;

        const timestamp = new Date(msg.timestamp).toLocaleTimeString();
        const snrText = msg.snr ? ` (SNR: ${msg.snr.toFixed(1)}dB)` : '';

        messageElement.innerHTML = `
            <div class="message-header">
                ${timestamp} - ${msg.from}${msg.to ? ' → ' + msg.to : ''}${snrText}
            </div>
            <div class="message-content">${this.escapeHtml(msg.message)}</div>
        `;

        // Check if this message already exists (avoid duplicates)
        const existingMessages = messagesContainer.querySelectorAll('.message');
        for (let existing of existingMessages) {
            const existingContent = existing.querySelector('.message-content').textContent;
            const existingHeader = existing.querySelector('.message-header').textContent;

            if (existingContent === msg.message && existingHeader.includes(msg.from)) {
                return; // Don't add duplicate
            }
        }

        messagesContainer.appendChild(messageElement);
        messagesContainer.scrollTop = messagesContainer.scrollHeight;
    }

    async sendMessage() {
        const toCallsign = document.getElementById('to-callsign').value.trim().toUpperCase();
        const messageText = document.getElementById('message-text').value.trim();

        if (!messageText) {
            alert('Please enter a message');
            return;
        }

        // Start transmission progress tracking
        this.startTransmissionProgress(messageText);

        try {
            const response = await fetch('/api/v1/messages', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    to: toCallsign,
                    message: messageText,
                }),
            });

            if (response.ok) {
                const data = await response.json();
                console.log('Message queued:', data);

                // Clear the message input
                document.getElementById('message-text').value = '';

                // Add to display as transmitted message
                if (data.message) {
                    this.addMessage(data.message, 'tx');
                }

                // Force refresh messages to get any updates
                setTimeout(() => this.loadMessages(), 500);

            } else {
                const error = await response.json();
                alert(`Failed to send message: ${error.error}`);
                this.endTransmissionProgress();
            }
        } catch (error) {
            console.error('Failed to send message:', error);
            alert('Failed to send message. Check connection.');
            this.endTransmissionProgress();
        }
    }

    async sendHeartbeat() {
        const callsign = document.querySelector('.callsign').textContent;
        const grid = document.querySelector('.grid').textContent.replace(/[()]/g, '');

        // Send in natural format - preprocessing will handle JS8 formatting
        await this.sendMessageWithText(`${callsign}: HEARTBEAT ${grid}`);
    }

    async sendCQ() {
        const callsign = document.querySelector('.callsign').textContent;

        await this.sendMessageWithText(`CQ CQ DE ${callsign} ${callsign} K`);
    }

    async sendMessageWithText(messageText) {
        try {
            const response = await fetch('/api/v1/messages', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    to: '',
                    message: messageText,
                }),
            });

            if (response.ok) {
                const data = await response.json();
                console.log('Message queued:', data);

                if (data.message) {
                    this.addMessage(data.message, 'tx');
                }

                // Force refresh messages
                setTimeout(() => this.loadMessages(), 500);

            } else {
                const error = await response.json();
                alert(`Failed to send message: ${error.error}`);
            }
        } catch (error) {
            console.error('Failed to send message:', error);
            alert('Failed to send message. Check connection.');
        }
    }

    async setFrequency(frequency) {
        try {
            const response = await fetch('/api/v1/radio/frequency', {
                method: 'PUT',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({ frequency }),
            });

            if (!response.ok) {
                const error = await response.json();
                alert(`Failed to set frequency: ${error.error}`);
                // Revert the input value
                this.updateStatus();
            }
        } catch (error) {
            console.error('Failed to set frequency:', error);
            alert('Failed to set frequency. Check connection.');
        }
    }

    async abortTransmission() {
        try {
            const response = await fetch('/api/v1/abort', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
            });

            if (response.ok) {
                const data = await response.json();
                console.log('Transmission aborted:', data);

                // Silently force refresh status to get updated PTT state
                setTimeout(() => this.updateStatus(), 100);

            } else {
                const error = await response.json();
                console.error('Failed to abort transmission:', error.error);
            }
        } catch (error) {
            console.error('Failed to abort transmission:', error);
        }
    }

    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    startTransmissionProgress(messageText) {
        // Estimate transmission duration (JS8 Normal: ~1.6 seconds per character)
        const estimatedDuration = Math.max(5000, messageText.length * 1600); // Minimum 5 seconds

        this.transmissionProgress.active = true;
        this.transmissionProgress.startTime = Date.now();
        this.transmissionProgress.estimatedDuration = estimatedDuration;
        this.transmissionProgress.messageLength = messageText.length;

        const progressBar = document.getElementById('tx-progress-fill');
        const progressText = document.getElementById('tx-progress-text');

        progressBar.classList.add('transmitting');
        progressText.textContent = 'Queued';

        // Update progress periodically
        this.progressInterval = setInterval(() => {
            this.updateTransmissionProgress();
        }, 100);

        // Auto-end after estimated duration + buffer
        this.progressTimeout = setTimeout(() => {
            this.endTransmissionProgress();
        }, estimatedDuration + 5000);
    }

    updateTransmissionProgress() {
        if (!this.transmissionProgress.active) return;

        const elapsed = Date.now() - this.transmissionProgress.startTime;
        const progress = Math.min(100, (elapsed / this.transmissionProgress.estimatedDuration) * 100);

        const progressBar = document.getElementById('tx-progress-fill');
        const progressText = document.getElementById('tx-progress-text');

        progressBar.style.width = `${progress}%`;

        if (elapsed < 2000) {
            progressText.textContent = 'Queued';
        } else if (elapsed < 4000) {
            progressText.textContent = 'Starting';
        } else if (progress < 90) {
            progressText.textContent = `TX ${Math.round(progress)}%`;
        } else {
            progressText.textContent = 'Finishing';
        }
    }

    endTransmissionProgress() {
        if (!this.transmissionProgress.active) return;

        this.transmissionProgress.active = false;

        const progressBar = document.getElementById('tx-progress-fill');
        const progressText = document.getElementById('tx-progress-text');

        progressBar.classList.remove('transmitting');
        progressBar.style.width = '0%';
        progressText.textContent = 'Ready';

        if (this.progressInterval) {
            clearInterval(this.progressInterval);
            this.progressInterval = null;
        }

        if (this.progressTimeout) {
            clearTimeout(this.progressTimeout);
            this.progressTimeout = null;
        }
    }
}

// Initialize the client when the page loads
document.addEventListener('DOMContentLoaded', () => {
    window.js8dClient = new JS8DClient();
});

// Show polling status in console for debugging
console.log('js8d Web Interface - REST API Mode (polling every 2 seconds)');