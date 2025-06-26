# MQTT Socket Implementation

## Configuration

### Environment Variables

- `MQTT_BROKER_URL`: MQTT broker URL (default: `ws://medusa-mqtt:9001`)

### Broker Setup

You'll need an MQTT broker running. Popular options include:
- Mosquitto (open source)
- Eclipse HiveMQ
- AWS IoT Core
- Azure IoT Hub

Docker compose contains the setup for a broker

## Usage

### Server Side

The server automatically initializes MQTT on startup. TV devices connect via the `/api/tv/socket` endpoint with a `device_id` parameter.

### TV Device Connection

1. TV devices should connect to the MQTT broker
2. Subscribe to their device-specific topic: `tv/{device_id}/commands`
3. Listen for messages and handle them accordingly

### Example Communication

1. Create screen via POST `admin/screens`
2. Create a pair request via POST `tv/pair`
3. Pair the created screen with the tv via POST `admin/screens/pair`
4. Create content via POST `admin/content`
5. Simulate a TV websocket connection via GET `tv/socket` passing in the deviceID from step 3
6. Assign content to the screen via POST `admin/screens/:screenID/content`

In the medusa-app container logs, you should see something like:

```bash
--:--:-- Message sent to TV device DEVICE_ID via MQTT
--:--:-- Received message: {"id": CONTENT_ID,"name":"CONTENT_NAME","type":"CONTENT_TYPE","url":"CONTENT_URL","created_at":"CONTENT_DATETIME"} from topic: tv/DEVICE_ID/commands
```


## API Functions

- `InitMQTTClient(clientName)`: Initialize an MQTT client with clientName
- `SetBrokerURL(url)`: Configure the MQTT broker URL
- `SendMessageToScreen(deviceID, message)`: Send message to specific device
- `SendMessageToAllScreens(message)`: Broadcast message to all devices
- `DisconnectTV(deviceID)`: Disconnect a specific device
- `GetConnectedTVs()`: Get list of connected device IDs
- `CleanupMQTT()`: Clean up all connections

## Topics

- `tv/{device_id}/commands`: Device-specific commands
- Each device subscribes to its own topic for receiving commands