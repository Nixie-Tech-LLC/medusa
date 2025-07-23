package endpoints

import (
	"encoding/json"
	"fmt"
	"github.com/Nixie-Tech-LLC/medusa/internal/db"
	adminpackets "github.com/Nixie-Tech-LLC/medusa/internal/http/api/admin/packets"
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

	r.GET("/register", ctl.registerPairingCode)
	r.GET("/socket", ctl.tvWebSocket)
}

// registerPairingCode binds a JSON pairing request, checks that the screen isn’t already paired,
// stores the pairing code in Redis, and responds with the device ID or an error.
func (t *TvController) registerPairingCode(c *gin.Context) {
	var request packets.RegisterPairingCodeRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		log.Error().Err(err).Msg("failed to bind JSON")
		return
	}

	isPaired, err := db.IsScreenPairedByDeviceID(&request.DeviceID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		log.Error().Err(err).Msg("failed to check if screen is paired by device")
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

	// Check if screen has a device ID assigned
	if screen.DeviceID == nil {
		log.Error().Str("deviceID", request.DeviceID).Msg("Screen found but device ID is nil")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "screen not properly paired"})
		return
	}

	deviceID := *screen.DeviceID

	// Create MQTT client for this TV device
	client, err := middleware.CreateMQTTClient(fmt.Sprintf("tv-%s", deviceID))
	if err != nil {
		log.Error().Err(err).Str("deviceID", deviceID).Msg("Failed to connect TV to MQTT")
	}

	// Subscribe to device-specific topic
	topic := fmt.Sprintf("tv/%s/commands", deviceID)
	if token := client.Subscribe(topic, 1, nil); token.Wait() && token.Error() != nil {
		log.Error().Err(err).Str("deviceID", deviceID).Str("topic", topic).Msg("Failed to subscribe to topic")
		client.Disconnect(250)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to subscribe to MQTT topic"})
		return
	}

	middleware.ClientMutex.Lock()
	middleware.TvClients[deviceID] = client
	middleware.ClientMutex.Unlock()

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
			if err := middleware.SendMessageToScreen(deviceID, response); err != nil {
				log.Error().Err(err).Str("deviceID", deviceID).
					Msg("Failed to send pending playlist to device")
			} else {
				log.Info().Str("deviceID", deviceID).Str("playlist_name", playlistName).
					Msg("Successfully sent pending playlist to device")
			}
		}
	}()

	c.JSON(http.StatusOK, gin.H{"success": "device connected successfully"})
}
