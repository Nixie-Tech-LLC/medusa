# MQTT Socket Implementation

This package provides MQTT-based communication for TV devices in the Medusa system, replacing the previous WebSocket implementation.

## Overview

The MQTT implementation provides:
- Real-time communication between the server and TV devices
- Automatic reconnection handling
- Topic-based messaging for device-specific commands
- Support for broadcasting messages to all connected devices

## Configuration

### Environment Variables

- `MQTT_BROKER_URL`: MQTT broker URL (default: `tcp://localhost:1883`)

### Broker Setup

You'll need an MQTT broker running. Popular options include:
- Mosquitto (open source)
- Eclipse HiveMQ
- AWS IoT Core
- Azure IoT Hub

## Usage

### Server Side

The server automatically initializes MQTT on startup. TV devices connect via the `/api/tv/socket` endpoint with a `device_id` parameter.

### TV Device Connection

1. TV devices should connect to the MQTT broker
2. Subscribe to their device-specific topic: `tv/{device_id}/commands`
3. Listen for messages and handle them accordingly

### Sending Messages

```go
// Send message to specific device
err := middleware.SendMessageToScreen("device123", []byte(`{"type": "content_update", "content_id": 1}`))

// Send message to all connected devices
err := middleware.SendMessageToAllScreens([]byte(`{"type": "broadcast", "message": "Hello all TVs!"}`))
```

### Message Format

Messages are sent as JSON with the following structure:

```json
{
  "type": "content_update",
  "content_id": 123,
  "content_name": "Video Title",
  "content_type": "video",
  "content_url": "https://example.com/video.mp4",
  "timestamp": 1640995200
}
```

## API Functions

- `InitMQTT()`: Initialize the MQTT client
- `SetBrokerURL(url)`: Configure the MQTT broker URL
- `SendMessageToScreen(deviceID, message)`: Send message to specific device
- `SendMessageToAllScreens(message)`: Broadcast message to all devices
- `DisconnectTV(deviceID)`: Disconnect a specific device
- `GetConnectedTVs()`: Get list of connected device IDs
- `CleanupMQTT()`: Clean up all connections

## Topics

- `tv/{device_id}/commands`: Device-specific commands
- Each device subscribes to its own topic for receiving commands

## Error Handling

The implementation includes automatic reconnection and error logging. Failed message sends are logged but don't crash the application. 