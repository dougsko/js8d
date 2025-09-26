# js8d REST API Documentation

js8d provides a comprehensive REST API for integration with external applications and automated control.

## Table of Contents

- [Base URL and Authentication](#base-url-and-authentication)
- [Response Format](#response-format)
- [Messages API](#messages-api)
- [Radio Control API](#radio-control-api)
- [Status API](#status-api)
- [Configuration API](#configuration-api)
- [WebSocket API](#websocket-api)
- [Error Handling](#error-handling)
- [Rate Limiting](#rate-limiting)
- [Examples](#examples)

## Base URL and Authentication

**Base URL:** `http://localhost:8080/api/v1`

**Authentication:** Optional API key via header:
```http
Authorization: Bearer your-api-key
```

Configure API key in `config.yaml`:
```yaml
web:
  api_key: "your-secret-api-key"
```

## Response Format

All API responses use JSON format:

```json
{
  "success": true,
  "data": { ... },
  "message": "Operation completed successfully"
}
```

Error responses:
```json
{
  "success": false,
  "error": "Error description",
  "code": "ERROR_CODE"
}
```

## Messages API

### Send Message

Send a JS8 message for transmission.

**Endpoint:** `POST /api/v1/messages`

**Request Body:**
```json
{
  "to": "N0CALL",
  "message": "Hello from js8d!",
  "priority": "normal"
}
```

**Parameters:**
- `to` (string, optional): Target callsign. Empty for broadcast
- `message` (string, required): Message text (max 80 characters)
- `priority` (string, optional): Priority level (`high`, `normal`, `low`)

**Response:**
```json
{
  "success": true,
  "data": {
    "id": "msg_12345",
    "message": {
      "id": "msg_12345",
      "from": "W1ABC",
      "to": "N0CALL",
      "message": "Hello from js8d!",
      "timestamp": "2024-01-15T10:30:00Z",
      "status": "queued"
    }
  },
  "message": "Message queued for transmission"
}
```

### Get Messages

Retrieve recent messages.

**Endpoint:** `GET /api/v1/messages`

**Query Parameters:**
- `limit` (int, optional): Maximum number of messages (default: 50, max: 500)
- `offset` (int, optional): Pagination offset (default: 0)
- `since` (string, optional): ISO 8601 timestamp for messages after this time
- `from` (string, optional): Filter by sender callsign
- `to` (string, optional): Filter by recipient callsign

**Response:**
```json
{
  "success": true,
  "data": {
    "messages": [
      {
        "id": "msg_12345",
        "from": "N0CALL",
        "to": "W1ABC",
        "message": "Hello World!",
        "timestamp": "2024-01-15T10:30:00Z",
        "snr": 12.5,
        "frequency": 14078000,
        "type": "received"
      }
    ],
    "count": 1,
    "total": 156
  }
}
```

### Get Single Message

Retrieve a specific message by ID.

**Endpoint:** `GET /api/v1/messages/{id}`

**Response:**
```json
{
  "success": true,
  "data": {
    "message": {
      "id": "msg_12345",
      "from": "N0CALL",
      "to": "W1ABC",
      "message": "Hello World!",
      "timestamp": "2024-01-15T10:30:00Z",
      "snr": 12.5,
      "frequency": 14078000,
      "type": "received"
    }
  }
}
```

### Delete Message

Delete a message from storage.

**Endpoint:** `DELETE /api/v1/messages/{id}`

**Response:**
```json
{
  "success": true,
  "message": "Message deleted successfully"
}
```

## Radio Control API

### Get Radio Status

Get current radio status and settings.

**Endpoint:** `GET /api/v1/radio/status`

**Response:**
```json
{
  "success": true,
  "data": {
    "connected": true,
    "frequency": 14078000,
    "mode": "USB",
    "bandwidth": 3000,
    "ptt": false,
    "power": 50,
    "swr": 1.2,
    "model": "IC-7300",
    "device": "/dev/ttyUSB0"
  }
}
```

### Set Frequency

Change radio frequency.

**Endpoint:** `PUT /api/v1/radio/frequency`

**Request Body:**
```json
{
  "frequency": 14078000
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "frequency": 14078000
  },
  "message": "Frequency set successfully"
}
```

### Set Mode

Change radio operating mode.

**Endpoint:** `PUT /api/v1/radio/mode`

**Request Body:**
```json
{
  "mode": "USB",
  "bandwidth": 3000
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "mode": "USB",
    "bandwidth": 3000
  },
  "message": "Mode set successfully"
}
```

### PTT Control

Control Push-to-Talk (PTT).

**Endpoint:** `POST /api/v1/radio/ptt`

**Request Body:**
```json
{
  "state": true
}
```

**Parameters:**
- `state` (boolean): PTT state (true = transmit, false = receive)

**Response:**
```json
{
  "success": true,
  "data": {
    "ptt": true
  },
  "message": "PTT activated"
}
```

## Status API

### Get System Status

Get overall system status and statistics.

**Endpoint:** `GET /api/v1/status`

**Response:**
```json
{
  "success": true,
  "data": {
    "status": "running",
    "uptime": 3600,
    "version": "1.0.0",
    "build": "abc123",
    "frequency": 14078000,
    "ptt": false,
    "connected": true,
    "audio": {
      "input_device": "hw:1,0",
      "output_device": "hw:1,0",
      "sample_rate": 48000,
      "buffer_size": 1024
    },
    "radio": {
      "connected": true,
      "model": "IC-7300",
      "device": "/dev/ttyUSB0"
    },
    "statistics": {
      "messages_received": 42,
      "messages_transmitted": 15,
      "total_runtime": 7200
    }
  }
}
```

### Get Health Check

Simple health check endpoint.

**Endpoint:** `GET /api/v1/health`

**Response:**
```json
{
  "success": true,
  "data": {
    "status": "healthy",
    "timestamp": "2024-01-15T10:30:00Z"
  }
}
```

## Configuration API

### Get Configuration

Retrieve current configuration.

**Endpoint:** `GET /api/v1/config`

**Response:**
```json
{
  "success": true,
  "data": {
    "station": {
      "callsign": "W1ABC",
      "grid": "FN42aa"
    },
    "audio": {
      "input_device": "hw:1,0",
      "output_device": "hw:1,0",
      "sample_rate": 48000
    },
    "radio": {
      "use_hamlib": true,
      "model": "311",
      "device": "/dev/ttyUSB0",
      "baud_rate": 9600
    }
  }
}
```

### Update Configuration

Update configuration settings.

**Endpoint:** `PUT /api/v1/config`

**Request Body:**
```json
{
  "station": {
    "callsign": "W1ABC",
    "grid": "FN42aa"
  },
  "audio": {
    "input_device": "hw:1,0",
    "output_device": "hw:1,0"
  }
}
```

**Response:**
```json
{
  "success": true,
  "message": "Configuration updated successfully"
}
```

### Reload Configuration

Reload configuration from file.

**Endpoint:** `POST /api/v1/config/reload`

**Response:**
```json
{
  "success": true,
  "message": "Configuration reloaded successfully"
}
```

### Get Audio Devices

List available audio devices.

**Endpoint:** `GET /api/v1/config/audio/devices`

**Response:**
```json
{
  "success": true,
  "data": {
    "input_devices": [
      {
        "id": "hw:0,0",
        "name": "Built-in Audio",
        "channels": 2
      },
      {
        "id": "hw:1,0",
        "name": "USB Audio Device",
        "channels": 1
      }
    ],
    "output_devices": [
      {
        "id": "hw:0,0",
        "name": "Built-in Audio",
        "channels": 2
      },
      {
        "id": "hw:1,0",
        "name": "USB Audio Device",
        "channels": 1
      }
    ]
  }
}
```

## WebSocket API

### Real-time Messages

Connect to receive real-time message updates.

**Endpoint:** `ws://localhost:8080/ws/messages`

**Message Format:**
```json
{
  "type": "message",
  "data": {
    "id": "msg_12345",
    "from": "N0CALL",
    "to": "W1ABC",
    "message": "Hello World!",
    "timestamp": "2024-01-15T10:30:00Z",
    "snr": 12.5
  }
}
```

### Audio Spectrum Data

Connect to receive real-time audio spectrum data for display.

**Endpoint:** `ws://localhost:8080/ws/audio`

**Message Format:**
```json
{
  "type": "spectrum",
  "data": {
    "timestamp": "2024-01-15T10:30:00Z",
    "spectrum": {
      "bins": [0.1, 0.2, 0.3, ...],
      "sample_rate": 48000,
      "fft_size": 1024
    }
  }
}
```

### Status Updates

Connect to receive real-time status updates.

**Endpoint:** `ws://localhost:8080/ws/status`

**Message Format:**
```json
{
  "type": "status",
  "data": {
    "ptt": false,
    "frequency": 14078000,
    "connected": true,
    "timestamp": "2024-01-15T10:30:00Z"
  }
}
```

## Error Handling

### HTTP Status Codes

- `200 OK`: Request successful
- `201 Created`: Resource created successfully
- `400 Bad Request`: Invalid request format or parameters
- `401 Unauthorized`: Missing or invalid API key
- `404 Not Found`: Resource not found
- `409 Conflict`: Resource conflict (e.g., duplicate message)
- `429 Too Many Requests`: Rate limit exceeded
- `500 Internal Server Error`: Server error

### Error Response Format

```json
{
  "success": false,
  "error": "Detailed error description",
  "code": "ERROR_CODE",
  "details": {
    "field": "Additional error context"
  }
}
```

### Common Error Codes

- `INVALID_CALLSIGN`: Invalid amateur radio callsign format
- `MESSAGE_TOO_LONG`: Message exceeds 80 character limit
- `RADIO_NOT_CONNECTED`: Radio control not available
- `AUDIO_DEVICE_ERROR`: Audio device not accessible
- `FREQUENCY_OUT_OF_RANGE`: Frequency outside valid range
- `CONFIGURATION_ERROR`: Configuration validation failed

## Rate Limiting

API requests are rate-limited to prevent abuse:

- **Default**: 100 requests per minute
- **Burst**: 10 requests per second
- **Headers**: Rate limit status included in response headers

```http
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1642248600
```

Configure rate limiting in `config.yaml`:
```yaml
api:
  rate_limit_enabled: true
  rate_limit_requests: 100
  rate_limit_burst: 10
```

## Examples

### Python Client Example

```python
import requests
import json

class JS8DClient:
    def __init__(self, base_url="http://localhost:8080", api_key=None):
        self.base_url = base_url
        self.headers = {"Content-Type": "application/json"}
        if api_key:
            self.headers["Authorization"] = f"Bearer {api_key}"

    def send_message(self, to_call, message):
        url = f"{self.base_url}/api/v1/messages"
        data = {"to": to_call, "message": message}
        response = requests.post(url, json=data, headers=self.headers)
        return response.json()

    def get_messages(self, limit=50):
        url = f"{self.base_url}/api/v1/messages?limit={limit}"
        response = requests.get(url, headers=self.headers)
        return response.json()

    def get_status(self):
        url = f"{self.base_url}/api/v1/status"
        response = requests.get(url, headers=self.headers)
        return response.json()

    def set_frequency(self, freq):
        url = f"{self.base_url}/api/v1/radio/frequency"
        data = {"frequency": freq}
        response = requests.put(url, json=data, headers=self.headers)
        return response.json()

# Usage
client = JS8DClient()
client.send_message("N0CALL", "Hello from Python!")
messages = client.get_messages(limit=10)
status = client.get_status()
```

### JavaScript/Node.js Example

```javascript
const axios = require('axios');

class JS8DClient {
    constructor(baseUrl = 'http://localhost:8080', apiKey = null) {
        this.baseUrl = baseUrl;
        this.headers = { 'Content-Type': 'application/json' };
        if (apiKey) {
            this.headers['Authorization'] = `Bearer ${apiKey}`;
        }
    }

    async sendMessage(toCall, message) {
        const url = `${this.baseUrl}/api/v1/messages`;
        const data = { to: toCall, message: message };
        const response = await axios.post(url, data, { headers: this.headers });
        return response.data;
    }

    async getMessages(limit = 50) {
        const url = `${this.baseUrl}/api/v1/messages?limit=${limit}`;
        const response = await axios.get(url, { headers: this.headers });
        return response.data;
    }

    async getStatus() {
        const url = `${this.baseUrl}/api/v1/status`;
        const response = await axios.get(url, { headers: this.headers });
        return response.data;
    }
}

// Usage
const client = new JS8DClient();
client.sendMessage('N0CALL', 'Hello from Node.js!');
```

### curl Examples

```bash
# Send a message
curl -X POST http://localhost:8080/api/v1/messages \
  -H "Content-Type: application/json" \
  -d '{"to": "N0CALL", "message": "Hello from curl!"}'

# Get recent messages
curl http://localhost:8080/api/v1/messages?limit=10

# Get system status
curl http://localhost:8080/api/v1/status

# Set frequency
curl -X PUT http://localhost:8080/api/v1/radio/frequency \
  -H "Content-Type: application/json" \
  -d '{"frequency": 14078000}'

# Get configuration
curl http://localhost:8080/api/v1/config

# Health check
curl http://localhost:8080/api/v1/health
```

### WebSocket Client Example (JavaScript)

```javascript
// Connect to real-time messages
const messagesWs = new WebSocket('ws://localhost:8080/ws/messages');

messagesWs.onmessage = (event) => {
    const data = JSON.parse(event.data);
    if (data.type === 'message') {
        console.log('New message:', data.data);
    }
};

// Connect to audio spectrum data
const audioWs = new WebSocket('ws://localhost:8080/ws/audio');

audioWs.onmessage = (event) => {
    const data = JSON.parse(event.data);
    if (data.type === 'spectrum') {
        updateSpectrumDisplay(data.data.spectrum);
    }
};

// Connect to status updates
const statusWs = new WebSocket('ws://localhost:8080/ws/status');

statusWs.onmessage = (event) => {
    const data = JSON.parse(event.data);
    if (data.type === 'status') {
        updateStatusDisplay(data.data);
    }
};
```

## Integration Notes

1. **Polling vs WebSockets**: Use WebSockets for real-time updates, REST API for one-time queries
2. **Authentication**: API key authentication is optional but recommended for network deployments
3. **Error Handling**: Always check the `success` field in responses
4. **Rate Limits**: Respect rate limits to avoid being blocked
5. **CORS**: Configure `allowed_origins` for browser-based clients
6. **Timeouts**: Set appropriate timeouts for HTTP requests (recommended: 30 seconds)

## See Also

- [CONFIGURATION.md](CONFIGURATION.md) - API configuration options
- [INSTALLATION.md](INSTALLATION.md) - Setup and installation
- [TROUBLESHOOTING.md](TROUBLESHOOTING.md) - Common API issues