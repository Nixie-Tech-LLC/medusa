package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Nixie-Tech-LLC/medusa/internal/db"
	adminpackets "github.com/Nixie-Tech-LLC/medusa/internal/http/api/admin/packets"
	"github.com/Nixie-Tech-LLC/medusa/internal/http/api/tv/packets"
	"github.com/Nixie-Tech-LLC/medusa/internal/redis"
	"github.com/gin-gonic/gin"
	"net/http"
	"sync"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/rs/zerolog/log"
)

var (
	TvClients   = make(map[string]mqtt.Client)
	ClientMutex sync.RWMutex
	MqttClient  mqtt.Client
	BrokerURL   = "ws://localhost:9001" // Default MQTT broker URL
	BrokerUser  = ""
	BrokerPass  = ""
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

func SetBrokerUser(brokerUser string) {
	BrokerUser = brokerUser
}

func SetBrokerPass(brokerPass string) {
	BrokerPass = brokerPass
}

// CreateMQTTClient connects to the MQTT broker as clientName, sets up handlers,
// then attempts to connect to the broker.
//
// On success, it logs a confirmation and returns the connected mqtt.Client.
// On failure, it logs an error and returns nil with the connection error.
func CreateMQTTClient(clientName string) (mqtt.Client, error) {
	opts := mqtt.NewClientOptions()
	opts.Username = BrokerUser
	opts.Password = BrokerPass
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

// tvWebSocket is an MQTT-based handler for TV device connections
func tvWebSocket(ctx *gin.Context) {
	var request packets.TVRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		log.Error().Err(err).Msg("Error parsing request")
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}

	screen, err := db.GetScreenByDeviceID(&request.DeviceID)
	if err != nil {
		log.Error().Err(err).Str("deviceID", request.DeviceID).Msg("Device ID not found")
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized device"})
		return
	}

	// Check if screen has a device ID assigned
	if screen.DeviceID == nil {
		log.Error().Str("deviceID", request.DeviceID).Msg("Screen found but device ID is nil")
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "screen not properly paired"})
		return
	}

	deviceID := *screen.DeviceID

	// Check if client already exists and disconnect it
	DisconnectTV(deviceID)

	// Create MQTT client for this TV device
	client, err := CreateMQTTClient(fmt.Sprintf("tv-%s", deviceID))
	if err != nil {
		log.Error().Err(err).Str("deviceID", deviceID).Msg("Failed to connect TV to MQTT")
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to connect to MQTT"})
		return
	}

	// Subscribe to device-specific topic
	topic := fmt.Sprintf("tv/%s/commands", deviceID)
	if token := client.Subscribe(topic, 1, nil); token.Wait() && token.Error() != nil {
		log.Error().Err(err).Str("deviceID", deviceID).Str("topic", topic).Msg("Failed to subscribe to topic")
		client.Disconnect(250)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to subscribe to MQTT topic"})
		return
	}

	// Store the client in the global map
	ClientMutex.Lock()
	TvClients[deviceID] = client
	ClientMutex.Unlock()

	log.Info().Str("deviceID", deviceID).Msg("Connected device to MQTT")

	// Check for pending playlist assignments and send them
	go func() {
		// Get playlist content if one is assigned to this screen
		playlistName, contentItems, err := db.GetPlaylistContentForScreen(screen.ID)
		if err == nil && len(contentItems) > 0 {
			log.Info().Str("deviceID", deviceID).Str("playlist_name", playlistName).
				Msg("Sending pending playlist to newly connected device")

			// Create response for TV client
			contentList := make([]adminpackets.TVContentItem, len(contentItems))
			for i, item := range contentItems {
				contentList[i] = adminpackets.TVContentItem{
					URL:      item.URL,
					Duration: item.Duration,
					Type: 	  item.Type,
				}
			}

			response, err := json.Marshal(adminpackets.TVPlaylistResponse{
				PlaylistName: playlistName,
				ContentList:  contentList,
			})
			if err != nil {
				log.Error().Err(err).Str("deviceID", deviceID).
					Msg("Failed to marshal pending playlist response")
				return
			}

			// Send the playlist to the newly connected device
			if err := SendMessageToScreen(deviceID, response); err != nil {
				log.Error().Err(err).Str("deviceID", deviceID).
					Msg("Failed to send pending playlist to device")
			} else {
				log.Info().Str("deviceID", deviceID).Str("playlist_name", playlistName).
					Msg("Successfully sent pending playlist to device")
			}
		}
	}()

	redis.Rdb.Del(ctx, request.PairingCode)
	ctx.JSON(http.StatusOK, gin.H{"success": "device connected successfully"})

	return
}
