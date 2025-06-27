package middleware

import (
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/Nixie-Tech-LLC/medusa/internal/db"
	tvpackets "github.com/Nixie-Tech-LLC/medusa/internal/http/api/tv/packets"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gin-gonic/gin"
)

var (
	tvClients   = make(map[string]mqtt.Client)
	clientMutex sync.RWMutex
	mqttClient  mqtt.Client
	brokerURL   = "tcp://0.0.0.0:1883" // Default MQTT broker URL
)

// MQTT message handler for TV devices
var messagePubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	log.Printf("Received message: %s from topic: %s\n", msg.Payload(), msg.Topic())
}

// MQTT connection handler
var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	log.Println("Connected to MQTT broker")
}

// MQTT connection lost handler
var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	log.Printf("Connection lost: %v", err)
}

// Initialize MQTT client
func CreateMQTTClient(clientName string) (mqtt.Client, error) {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(brokerURL)
	opts.SetClientID(clientName)
	opts.SetDefaultPublishHandler(messagePubHandler)
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectLostHandler

	mqttClient = mqtt.NewClient(opts)
	if token := mqttClient.Connect(); token.Wait() && token.Error() != nil {
		return nil, fmt.Errorf("failed to connect to MQTT broker: %v", token.Error())
	}

	log.Println("MQTT client initialized successfully")
	return mqttClient, nil
}

// SetBrokerURL allows configuration of the MQTT broker URL
func SetBrokerURL(url string) {
	brokerURL = url
}

// TODO: Poll the broker to send messages to the screen on an interval in order to update the tv when it is turned on and off
// TVWebSocket is now an MQTT-based handler for TV device connections
func TVWebSocket() gin.HandlerFunc {
	return func(c *gin.Context) {
		var request tvpackets.TVRequest
		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		screen, err := db.GetScreenByDeviceID(&request.DeviceID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized device"})
			return
		}

		// Create MQTT client for this TV device
		client, err := CreateMQTTClient(fmt.Sprintf("tv-%s", request.DeviceID))
		if err != nil {
			log.Printf("Failed to connect TV device %s to MQTT: %v", request.DeviceID, err)
		}

		// Subscribe to device-specific topic
		topic := fmt.Sprintf("tv/%s/commands", request.DeviceID)
		if token := client.Subscribe(topic, 1, nil); token.Wait() && token.Error() != nil {
			log.Printf("Failed to subscribe TV device %s to topic %s: %v", request.DeviceID, topic, token.Error())
			client.Disconnect(250)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to subscribe to MQTT topic"})
			return
		}

		// Store the client connection
		clientMutex.Lock()
		tvClients[*screen.DeviceID] = client
		clientMutex.Unlock()

		log.Printf("MQTT connected: screen %d (device: %s)", screen.ID, request.DeviceID)

		// Send connection success response
		c.JSON(http.StatusOK, gin.H{"success": "device connected successsfully"})

		// Keep the connection alive (this is now handled by MQTT client)
		// The client will automatically reconnect if connection is lost
	}
}

// SendMessageToScreen sends a message to a specific TV screen via MQTT
func SendMessageToScreen(deviceID string, message []byte) error {
	clientMutex.RLock()
	client, exists := tvClients[deviceID]
	clientMutex.RUnlock()
	if !exists {
		return fmt.Errorf("TV device %s not connected", deviceID)
	}
	topic := fmt.Sprintf("tv/%s/commands", deviceID)
	token := client.Publish(topic, 1, false, message)
	token.Wait()

	if token.Error() != nil {
		return fmt.Errorf("failed to send message to TV device %s: %v", deviceID, token.Error())
	}

	log.Printf("Message sent to TV device %s via MQTT", deviceID)
	return nil
}

// SendMessageToAllScreens sends a message to all connected TV screens
func SendMessageToAllScreens(message []byte) error {
	clientMutex.RLock()
	defer clientMutex.RUnlock()

	var errors []string
	for deviceID, client := range tvClients {
		topic := fmt.Sprintf("tv/%s/commands", deviceID)
		token := client.Publish(topic, 1, false, message)
		token.Wait()

		if token.Error() != nil {
			errors = append(errors, fmt.Sprintf("device %s: %v", deviceID, token.Error()))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to send messages to some devices: %v", errors)
	}

	log.Printf("Message sent to all %d connected TV devices via MQTT", len(tvClients))
	return nil
}

// DisconnectTV disconnects a specific TV device
func DisconnectTV(deviceID string) {
	clientMutex.Lock()
	defer clientMutex.Unlock()

	if client, exists := tvClients[deviceID]; exists {
		client.Disconnect(250)
		delete(tvClients, deviceID)
		log.Printf("TV device %s disconnected from MQTT", deviceID)
	}
}

// GetConnectedTVs returns a list of connected TV device IDs
func GetConnectedTVs() []string {
	clientMutex.RLock()
	defer clientMutex.RUnlock()

	devices := make([]string, 0, len(tvClients))
	for deviceID := range tvClients {
		devices = append(devices, deviceID)
	}
	return devices
}

// CleanupMQTT disconnects all clients and the main MQTT client
func CleanupMQTT() {
	clientMutex.Lock()
	defer clientMutex.Unlock()

	// Disconnect all TV clients
	for deviceID, client := range tvClients {
		client.Disconnect(250)
		log.Printf("Disconnected TV device %s", deviceID)
	}
	tvClients = make(map[string]mqtt.Client)

	// Disconnect main MQTT client
	if mqttClient != nil {
		mqttClient.Disconnect(250)
		log.Println("Main MQTT client disconnected")
	}
}
