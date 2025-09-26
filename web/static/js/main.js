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

// Show polling status in console for debugging
console.log('js8d Web Interface - REST API Mode (polling every 2 seconds)');