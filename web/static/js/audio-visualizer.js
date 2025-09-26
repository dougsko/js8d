class AudioVisualizer {
    constructor() {
        this.websocket = null;
        this.isConnected = false;

        // Canvas elements
        this.inputVUCanvas = document.getElementById('input-vu-meter');
        this.outputVUCanvas = document.getElementById('output-vu-meter');
        this.spectrumCanvas = document.getElementById('spectrum-canvas');
        this.waterfallCanvas = document.getElementById('waterfall-canvas');

        // Canvas contexts
        this.inputVUCtx = this.inputVUCanvas?.getContext('2d');
        this.outputVUCtx = this.outputVUCanvas?.getContext('2d');
        this.spectrumCtx = this.spectrumCanvas?.getContext('2d');
        this.waterfallCtx = this.waterfallCanvas?.getContext('2d');

        // Waterfall history buffer - reduced size for better performance
        this.waterfallHistory = [];
        this.maxWaterfallLines = 150;

        // Peak hold values
        this.inputPeakHold = -100;
        this.outputPeakHold = -100;
        this.peakHoldTime = Date.now();

        // Performance optimization - throttle canvas updates
        this.lastUpdateTime = 0;
        this.minUpdateInterval = 50; // Limit updates to 20Hz max

        // Event listeners
        this.setupEventListeners();

        // Initialize canvases
        this.initializeCanvases();
    }

    setupEventListeners() {
        const startBtn = document.getElementById('start-audio-monitoring');
        const stopBtn = document.getElementById('stop-audio-monitoring');

        if (startBtn) {
            startBtn.addEventListener('click', () => this.startMonitoring());
        }

        if (stopBtn) {
            stopBtn.addEventListener('click', () => this.stopMonitoring());
        }

        // Auto-start monitoring when page loads (disabled by default for better performance)
        // Uncomment the following lines to enable auto-start
        // window.addEventListener('load', () => {
        //     setTimeout(() => this.startMonitoring(), 2000);
        // });
    }

    initializeCanvases() {
        // Set high DPI scaling
        const ratio = window.devicePixelRatio || 1;

        [this.inputVUCanvas, this.outputVUCanvas, this.spectrumCanvas, this.waterfallCanvas].forEach(canvas => {
            if (!canvas) return;

            const rect = canvas.getBoundingClientRect();
            canvas.width = rect.width * ratio;
            canvas.height = rect.height * ratio;
            const ctx = canvas.getContext('2d');
            ctx.scale(ratio, ratio);
            canvas.style.width = rect.width + 'px';
            canvas.style.height = rect.height + 'px';
        });

        // Draw initial empty states
        this.drawVUMeter(this.inputVUCtx, -100, -100, false);
        this.drawVUMeter(this.outputVUCtx, -100, -100, false);
        this.drawSpectrum([]);
        this.drawWaterfall();
    }

    startMonitoring() {
        if (this.isConnected) {
            console.log('Audio monitoring already active');
            return;
        }

        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = `${protocol}//${window.location.host}/ws/audio`;

        console.log('Connecting to audio WebSocket:', wsUrl);

        try {
            this.websocket = new WebSocket(wsUrl);

            this.websocket.onopen = () => {
                console.log('Audio WebSocket connected');
                this.isConnected = true;
                this.updateMonitoringStatus(true);
            };

            this.websocket.onmessage = (event) => {
                try {
                    const data = JSON.parse(event.data);
                    this.updateVisualization(data);
                } catch (error) {
                    console.error('Error parsing WebSocket data:', error);
                }
            };

            this.websocket.onclose = () => {
                console.log('Audio WebSocket disconnected');
                this.isConnected = false;
                this.updateMonitoringStatus(false);

                // Auto-reconnect after 5 seconds (less aggressive)
                setTimeout(() => {
                    if (!this.isConnected) {
                        console.log('Attempting WebSocket reconnection...');
                        this.startMonitoring();
                    }
                }, 5000);
            };

            this.websocket.onerror = (error) => {
                console.error('Audio WebSocket error:', error);
                this.isConnected = false;
                this.updateMonitoringStatus(false);
            };

        } catch (error) {
            console.error('Failed to create WebSocket:', error);
            this.updateMonitoringStatus(false);
        }
    }

    stopMonitoring() {
        if (this.websocket) {
            this.websocket.close();
            this.websocket = null;
        }
        this.isConnected = false;
        this.updateMonitoringStatus(false);
    }

    updateMonitoringStatus(active) {
        const startBtn = document.getElementById('start-audio-monitoring');
        const stopBtn = document.getElementById('stop-audio-monitoring');

        if (startBtn && stopBtn) {
            if (active) {
                startBtn.disabled = true;
                startBtn.textContent = 'Monitoring Active';
                startBtn.style.background = '#4CAF50';
                stopBtn.disabled = false;
            } else {
                startBtn.disabled = false;
                startBtn.textContent = 'Start Monitoring';
                startBtn.style.background = '#2196F3';
                stopBtn.disabled = true;
            }
        }
    }

    updateVisualization(data) {
        // Throttle canvas updates to reduce CPU usage
        const now = Date.now();
        if (now - this.lastUpdateTime < this.minUpdateInterval) {
            return; // Skip this update
        }
        this.lastUpdateTime = now;

        // Update VU meters
        this.drawVUMeter(this.inputVUCtx, data.rms, data.peak, data.clipping);
        this.drawVUMeter(this.outputVUCtx, data.rms, data.peak, data.clipping);

        // Update spectrum display
        if (data.spectrum && data.spectrum.length > 0) {
            this.drawSpectrum(data.spectrum);

            // Add to waterfall history
            this.waterfallHistory.push(data.spectrum.slice());
            if (this.waterfallHistory.length > this.maxWaterfallLines) {
                this.waterfallHistory.shift();
            }
            this.drawWaterfall();
        }

        // Update statistics
        this.updateAudioStats(data);
    }

    drawVUMeter(ctx, rms, peak, clipping) {
        if (!ctx) return;

        const canvas = ctx.canvas;
        const width = canvas.clientWidth;
        const height = canvas.clientHeight;

        // Clear canvas
        ctx.fillStyle = '#1e1e1e';
        ctx.fillRect(0, 0, width, height);

        // Convert dB to linear scale (0-1)
        const rmsLinear = Math.max(0, Math.min(1, (rms + 60) / 60)); // -60dB to 0dB
        const peakLinear = Math.max(0, Math.min(1, (peak + 60) / 60));

        // Draw background scale marks
        this.drawVUScale(ctx, width, height);

        // Draw RMS level bar
        const rmsWidth = rmsLinear * (width - 20);
        const barHeight = height - 20;
        const barY = 10;

        // Color based on level
        let rmsColor = '#4CAF50'; // Green
        if (rms > -20) rmsColor = '#FF9800'; // Orange
        if (rms > -6) rmsColor = '#f44336';  // Red
        if (clipping) rmsColor = '#ff0000';  // Bright red

        ctx.fillStyle = rmsColor;
        ctx.fillRect(10, barY, rmsWidth, barHeight);

        // Draw peak hold line
        if (peak > -60) {
            const peakX = 10 + peakLinear * (width - 20);
            ctx.fillStyle = clipping ? '#ff0000' : '#ffffff';
            ctx.fillRect(peakX - 1, barY - 2, 2, barHeight + 4);
        }

        // Draw level text
        ctx.fillStyle = '#ffffff';
        ctx.font = '10px monospace';
        ctx.fillText(`RMS: ${rms.toFixed(1)}dB`, 10, height - 2);
        ctx.fillText(`Peak: ${peak.toFixed(1)}dB`, width - 80, height - 2);

        if (clipping) {
            ctx.fillStyle = '#ff0000';
            ctx.font = 'bold 10px monospace';
            ctx.fillText('CLIP!', width / 2 - 15, height - 2);
        }
    }

    drawVUScale(ctx, width, height) {
        ctx.strokeStyle = '#444';
        ctx.lineWidth = 1;

        // Scale marks at -60, -40, -20, -6, 0 dB
        const marks = [-60, -40, -20, -6, 0];
        marks.forEach(db => {
            const x = 10 + ((db + 60) / 60) * (width - 20);
            ctx.beginPath();
            ctx.moveTo(x, 5);
            ctx.lineTo(x, height - 15);
            ctx.stroke();
        });
    }

    drawSpectrum(spectrum) {
        if (!this.spectrumCtx || !spectrum || spectrum.length === 0) {
            return;
        }

        const canvas = this.spectrumCanvas;
        const ctx = this.spectrumCtx;
        const width = canvas.clientWidth;
        const height = canvas.clientHeight;

        // Clear canvas
        ctx.fillStyle = '#1e1e1e';
        ctx.fillRect(0, 0, width, height);

        // Draw frequency grid
        this.drawFrequencyGrid(ctx, width, height);

        if (spectrum.length === 0) return;

        const binWidth = width / spectrum.length;

        // Draw spectrum bars
        spectrum.forEach((magnitude, i) => {
            const normalizedMag = Math.max(0, Math.min(1, (magnitude + 100) / 80)); // -100dB to -20dB range
            const barHeight = normalizedMag * (height - 40);
            const x = i * binWidth;

            // Color based on magnitude
            const hue = 240 - (normalizedMag * 200); // Blue to red
            ctx.fillStyle = `hsl(${hue}, 100%, 50%)`;
            ctx.fillRect(x, height - 20 - barHeight, binWidth - 0.5, barHeight);
        });

        // Draw frequency labels - focus on JS8 range
        ctx.fillStyle = '#999';
        ctx.font = '10px monospace';
        const freqLabels = ['500Hz', '1000Hz', '1500Hz', '2000Hz', '2500Hz'];
        const labelPositions = [0.1, 0.3, 0.5, 0.7, 0.9];

        freqLabels.forEach((label, i) => {
            const x = labelPositions[i] * width;
            ctx.fillText(label, x - 20, height - 5);
        });
    }

    drawFrequencyGrid(ctx, width, height) {
        ctx.strokeStyle = '#333';
        ctx.lineWidth = 1;

        // Vertical frequency lines
        const freqLines = [0.1, 0.35, 0.5, 0.65, 0.9];
        freqLines.forEach(pos => {
            const x = pos * width;
            ctx.beginPath();
            ctx.moveTo(x, 0);
            ctx.lineTo(x, height - 20);
            ctx.stroke();
        });

        // Horizontal level lines
        for (let i = 1; i < 5; i++) {
            const y = (i / 5) * (height - 20);
            ctx.beginPath();
            ctx.moveTo(0, y);
            ctx.lineTo(width, y);
            ctx.stroke();
        }
    }

    drawWaterfall() {
        if (!this.waterfallCtx || this.waterfallHistory.length === 0) {
            return;
        }

        const canvas = this.waterfallCanvas;
        const ctx = this.waterfallCtx;
        const width = canvas.clientWidth;
        const height = canvas.clientHeight;

        // Clear canvas
        ctx.fillStyle = '#000';
        ctx.fillRect(0, 0, width, height);

        const lineHeight = height / this.maxWaterfallLines;
        const binWidth = width / (this.waterfallHistory[0]?.length || 1);

        // Draw waterfall from newest (top) to oldest (bottom)
        this.waterfallHistory.forEach((spectrum, historyIndex) => {
            const y = historyIndex * lineHeight;

            spectrum.forEach((magnitude, binIndex) => {
                const x = binIndex * binWidth;
                const normalizedMag = Math.max(0, Math.min(1, (magnitude + 100) / 80));

                // Create color from magnitude
                const intensity = Math.floor(normalizedMag * 255);
                const color = this.getWaterfallColor(normalizedMag);

                ctx.fillStyle = color;
                ctx.fillRect(x, y, binWidth, lineHeight + 1);
            });
        });

        // Draw frequency scale at bottom
        ctx.fillStyle = '#666';
        ctx.font = '10px monospace';
        const freqLabels = ['300', '1k', '1.5k', '2k', '3k'];
        const labelPositions = [0.1, 0.35, 0.5, 0.65, 0.9];

        freqLabels.forEach((label, i) => {
            const x = labelPositions[i] * width;
            ctx.fillText(label, x - 10, height - 2);
        });
    }

    getWaterfallColor(intensity) {
        // Create a blue-to-red color scale
        if (intensity < 0.25) {
            // Black to blue
            const blue = Math.floor(intensity * 4 * 128) + 64;
            return `rgb(0, 0, ${blue})`;
        } else if (intensity < 0.5) {
            // Blue to cyan
            const green = Math.floor((intensity - 0.25) * 4 * 255);
            return `rgb(0, ${green}, 192)`;
        } else if (intensity < 0.75) {
            // Cyan to yellow
            const red = Math.floor((intensity - 0.5) * 4 * 255);
            const blue = 192 - Math.floor((intensity - 0.5) * 4 * 192);
            return `rgb(${red}, 255, ${blue})`;
        } else {
            // Yellow to red
            const green = 255 - Math.floor((intensity - 0.75) * 4 * 255);
            return `rgb(255, ${green}, 0)`;
        }
    }

    updateAudioStats(data) {
        // Update audio statistics display
        const sampleRateEl = document.getElementById('audio-sample-rate-stat');
        const bufferSizeEl = document.getElementById('audio-buffer-size-stat');
        const clipRateEl = document.getElementById('audio-clip-rate-stat');
        const peakHoldEl = document.getElementById('audio-peak-hold-stat');

        if (sampleRateEl && data.sample_rate) {
            sampleRateEl.textContent = `${data.sample_rate} Hz`;
        }

        if (bufferSizeEl && data.spectrum) {
            bufferSizeEl.textContent = `${data.spectrum.length * 2} samples`;
        }

        if (clipRateEl && data.clipping !== undefined) {
            clipRateEl.textContent = data.clipping ? 'CLIPPING!' : 'OK';
            clipRateEl.style.color = data.clipping ? '#f44336' : '#4CAF50';
        }

        if (peakHoldEl && data.peak !== undefined) {
            peakHoldEl.textContent = `${data.peak.toFixed(1)} dB`;
        }
    }

    // Method to fetch and display current audio statistics
    async loadAudioStats() {
        try {
            const response = await fetch('/api/v1/audio/stats');
            const data = await response.json();

            if (data && data.statistics) {
                const stats = data.statistics;

                const sampleRateEl = document.getElementById('audio-sample-rate-stat');
                const bufferSizeEl = document.getElementById('audio-buffer-size-stat');
                const clipRateEl = document.getElementById('audio-clip-rate-stat');
                const peakHoldEl = document.getElementById('audio-peak-hold-stat');

                if (sampleRateEl) sampleRateEl.textContent = `${stats.sample_rate || '--'} Hz`;
                if (bufferSizeEl) bufferSizeEl.textContent = `${stats.buffer_samples || '--'} samples`;
                if (clipRateEl) {
                    const clipRate = stats.clip_rate_pct || 0;
                    clipRateEl.textContent = `${clipRate.toFixed(2)}%`;
                    clipRateEl.style.color = clipRate > 1 ? '#f44336' : '#4CAF50';
                }
                if (peakHoldEl) peakHoldEl.textContent = `${(stats.peak_hold_db || -100).toFixed(1)} dB`;
            }
        } catch (error) {
            console.error('Failed to load audio stats:', error);
        }
    }
}

// Global instance
let audioVisualizer = null;

// Initialize when page loads
document.addEventListener('DOMContentLoaded', () => {
    // Only initialize if we're on the settings page with audio monitoring
    if (document.getElementById('input-vu-meter')) {
        audioVisualizer = new AudioVisualizer();

        // Load initial stats
        setTimeout(() => {
            if (audioVisualizer) {
                audioVisualizer.loadAudioStats();
            }
        }, 500);
    }
});