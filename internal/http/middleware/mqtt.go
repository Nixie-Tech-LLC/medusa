package middleware

import (
	"context"
	"fmt"
	"github.com/Nixie-Tech-LLC/medusa/internal/redis"
	"sync"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/rs/zerolog/log"
)

var (
	TvClients   = make(map[string]mqtt.Client)
	ClientMutex sync.RWMutex
	MqttClient  mqtt.Client
	BrokerURL   = "ws://localhost:9001" // Default MQTT broker URL
)

// MQTT message handler for TV devices
var messagePubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	log.Info().Str("topic", msg.Topic()).Msg("Received message")
}

// MQTT connection handler
var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	log.Info().Msg("Client connected to MQTT broker")
}

// MQTT connection lost handler
var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	log.Warn().Err(err).Msg("Connection lost with client")
}

// SetBrokerURL allows configuration of the MQTT broker URL
func SetBrokerURL(url string) {
	BrokerURL = url
}

// CreateMQTTClient connects to the MQTT broker as clientName, sets up handlers,
// then attempts to connect to the broker.
//
// On success, it logs a confirmation and returns the connected mqtt.Client.
// On failure, it logs an error and returns nil with the connection error.
func CreateMQTTClient(clientName string) (mqtt.Client, error) {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(BrokerURL)
	opts.SetClientID(clientName)
	opts.SetDefaultPublishHandler(messagePubHandler)
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectLostHandler

	MqttClient = mqtt.NewClient(opts)
	if token := MqttClient.Connect(); token.Wait() && token.Error() != nil {
		log.Error().Err(token.Error()).Msg("Failed to connect to MQTT broker")
		return nil, token.Error()
	}

	log.Info().Msg("MQTT client created successfully")
	return MqttClient, nil
}

// SendMessageToScreen sends a message to a specific TV screen via MQTT
func SendMessageToScreen(deviceID string, message []byte) error {
	ClientMutex.RLock()
	client, exists := TvClients[deviceID]
	ClientMutex.RUnlock()
	if !exists {
		return fmt.Errorf("TV device %s not connected", deviceID)
	}
	topic := fmt.Sprintf("tv/%s/commands", deviceID)
	token := client.Publish(topic, 1, true, message)
	token.Wait()

	if token.Error() != nil {
		return fmt.Errorf("failed to send message to TV device %s: %v", deviceID, token.Error())
	}

	redis.Set(context.Background(), topic, message, 0)

	log.Printf("Message sent to TV device %s via MQTT", deviceID)
	return nil
}

// SendMessageToAllScreens sends a message to all connected TV screens
func SendMessageToAllScreens(message []byte) error {
	ClientMutex.RLock()
	defer ClientMutex.RUnlock()

	var errors []string
	for deviceID, client := range TvClients {
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

	log.Printf("Message sent to all %d connected TV devices via MQTT", len(TvClients))
	return nil
}

// DisconnectTV disconnects a specific TV device
func DisconnectTV(deviceID string) {
	ClientMutex.Lock()
	defer ClientMutex.Unlock()

	if client, exists := TvClients[deviceID]; exists {
		client.Disconnect(250)
		delete(TvClients, deviceID)
		log.Printf("TV device %s disconnected from MQTT", deviceID)
	}
}

// GetConnectedTVs returns a list of connected TV device IDs
func GetConnectedTVs() []string {
	ClientMutex.RLock()
	defer ClientMutex.RUnlock()

	devices := make([]string, 0, len(TvClients))
	for deviceID := range TvClients {
		devices = append(devices, deviceID)
	}
	return devices
}

// CleanupMQTT disconnects all clients and the main MQTT client
func CleanupMQTT() {
	ClientMutex.Lock()
	defer ClientMutex.Unlock()

	// Disconnect all TV clients
	for deviceID, client := range TvClients {
		client.Disconnect(250)
		log.Printf("Disconnected TV device %s", deviceID)
	}
	TvClients = make(map[string]mqtt.Client)

	// Disconnect main MQTT client
	if MqttClient != nil {
		MqttClient.Disconnect(250)
		log.Info().Msg("MQTT client disconnected")
	}
}
