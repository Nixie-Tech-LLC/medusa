package endpoints

import (
	"fmt"
	"github.com/Nixie-Tech-LLC/medusa/internal/db"
	"github.com/Nixie-Tech-LLC/medusa/internal/http/api/tv/packets"
	"github.com/Nixie-Tech-LLC/medusa/internal/http/middleware"
	"github.com/Nixie-Tech-LLC/medusa/internal/redis"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"net/http"
)

type TvController struct {
	store db.Store
}

func NewTvController(store db.Store) *TvController {
	return &TvController{store: store}
}

func RegisterPairingRoutes(r gin.IRoutes, store db.Store) {
	ctl := NewTvController(store)

	r.POST("/register", ctl.registerPairingCode)
	r.POST("/socket", ctl.tvWebSocket)
}

// registerPairingCode binds a JSON pairing request, checks that the screen isn’t already paired,
// stores the pairing code in Redis, and responds with the device ID or an error.
func (t *TvController) registerPairingCode(c *gin.Context) {
	var request packets.RegisterPairingCodeRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	isPaired, err := db.IsScreenPairedByDeviceID(&request.DeviceID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if isPaired == true {
		log.Error().Err(err).Msg("Screen is already paired")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Screen is already paired"})
		return
	}

	redis.Set(c, request.PairingCode, request.DeviceID, 0)

	c.JSON(http.StatusOK, packets.TVRequest{DeviceID: request.DeviceID})
}

// tvWebSocket is an MQTT-based handler for TV device connections
func (t *TvController) tvWebSocket(c *gin.Context) {
	var request packets.TVRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		log.Error().Err(err).Msg("Error parsing request")
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	screen, err := db.GetScreenByDeviceID(&request.DeviceID)
	if err != nil {
		log.Error().Err(err).Str("deviceID", request.DeviceID).Msg("Device ID not found")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized device"})
		return
	}

	deviceID := screen.DeviceID

	// Create MQTT client for this TV device
	client, err := middleware.CreateMQTTClient(fmt.Sprintf("tv-%s", *deviceID))
	if err != nil {
		log.Error().Err(err).Str("deviceID", *deviceID).Msg("Failed to connect TV to MQTT")
	}

	// Subscribe to device-specific topic
	topic := fmt.Sprintf("tv/%s/commands", *deviceID)
	if token := client.Subscribe(topic, 1, nil); token.Wait() && token.Error() != nil {
		log.Error().Err(err).Str("deviceID", *deviceID).Str("topic", topic).Msg("Failed to subscribe to topic")
		client.Disconnect(250)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to subscribe to MQTT topic"})
		return
	}

	middleware.ClientMutex.Lock()
	middleware.TvClients[*deviceID] = client
	middleware.ClientMutex.Unlock()

	log.Info().Str("deviceID", *deviceID).Msg("Connected device to MQTT")
	c.JSON(http.StatusOK, gin.H{"success": "device connected successfully"})
}
