// js8d Web Interface JavaScript

class JS8DClient {
    constructor() {
        this.ws = null;
        this.connected = false;
        this.messages = [];

        this.init();
    }

    init() {
        this.setupEventListeners();
        this.connectWebSocket();
        this.loadMessages();
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

        // Auto-scroll messages
        const messagesContainer = document.getElementById('messages');
        messagesContainer.addEventListener('scroll', () => {
            // TODO: Implement auto-scroll behavior
        });
    }

    connectWebSocket() {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = `${protocol}//${window.location.host}/ws`;

        this.ws = new WebSocket(wsUrl);

        this.ws.onopen = () => {
            console.log('WebSocket connected');
            this.connected = true;
            this.updateConnectionStatus();
        };

        this.ws.onmessage = (event) => {
            const data = JSON.parse(event.data);
            this.handleWebSocketMessage(data);
        };

        this.ws.onclose = () => {
            console.log('WebSocket disconnected');
            this.connected = false;
            this.updateConnectionStatus();

            // Attempt to reconnect after 3 seconds
            setTimeout(() => {
                this.connectWebSocket();
            }, 3000);
        };

        this.ws.onerror = (error) => {
            console.error('WebSocket error:', error);
        };
    }

    handleWebSocketMessage(data) {
        switch (data.type) {
            case 'message_received':
                this.addMessage(data.data, 'rx');
                break;
            case 'message_sent':
                this.addMessage(data.data, 'tx');
                break;
            case 'status':
                this.updateStatusFromData(data.data);
                break;
            case 'frequency_changed':
                document.getElementById('frequency').value = data.data.frequency;
                break;
            case 'ptt_changed':
                this.updatePTTStatus(data.data.ptt);
                break;
        }
    }

    async loadMessages() {
        try {
            const response = await fetch('/api/v1/messages');
            const data = await response.json();

            if (data.messages) {
                data.messages.forEach(msg => {
                    this.addMessage(msg, 'rx');
                });
            }
        } catch (error) {
            console.error('Failed to load messages:', error);
        }
    }

    async updateStatus() {
        try {
            const response = await fetch('/api/v1/status');
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

        messagesContainer.appendChild(messageElement);
        messagesContainer.scrollTop = messagesContainer.scrollHeight;

        // Update message count
        this.messages.push(msg);
        document.getElementById('message-count').textContent = `${this.messages.length} messages`;
    }

    async sendMessage() {
        const toCallsign = document.getElementById('to-callsign').value.trim().toUpperCase();
        const messageText = document.getElementById('message-text').value.trim();

        if (!messageText) {
            alert('Please enter a message');
            return;
        }

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
                this.addMessage(data.message, 'tx');
            } else {
                const error = await response.json();
                alert(`Failed to send message: ${error.error}`);
            }
        } catch (error) {
            console.error('Failed to send message:', error);
            alert('Failed to send message. Check connection.');
        }
    }

    async sendHeartbeat() {
        const callsign = document.querySelector('.callsign').textContent;
        const grid = document.querySelector('.grid').textContent.replace(/[()]/g, '');

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
                this.addMessage(data.message, 'tx');
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

    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }
}

// Initialize the client when the page loads
document.addEventListener('DOMContentLoaded', () => {
    window.js8dClient = new JS8DClient();
});

// Periodic status updates
setInterval(() => {
    if (window.js8dClient) {
        window.js8dClient.updateStatus();
    }
}, 30000); // Every 30 seconds