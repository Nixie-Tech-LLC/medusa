package endpoints

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"net/http"

	"github.com/Nixie-Tech-LLC/medusa/internal/db"
	adminpackets "github.com/Nixie-Tech-LLC/medusa/internal/http/api/admin/packets"
	"github.com/Nixie-Tech-LLC/medusa/internal/http/api/tv/packets"
	"github.com/Nixie-Tech-LLC/medusa/internal/redis"
)

type TvController struct {
	store db.Store
}

type PairingData struct {
	DeviceID string `json:"device_id"`
	IsPaired bool   `json:"is_paired"`
}

// generateContentETag creates an ETag based on playlist name and content items
func generateContentETag(playlistName string, contentItems []db.ContentItem) string {
	hasher := sha256.New()
	hasher.Write([]byte(playlistName))

	for _, item := range contentItems {
		hasher.Write([]byte(fmt.Sprintf("%s:%d", item.URL, item.Duration)))
	}

	hash := hex.EncodeToString(hasher.Sum(nil))
	return fmt.Sprintf(`"%s"`, hash[:16]) // Use first 16 chars for shorter ETag
}

func NewTvController(store db.Store) *TvController {
	return &TvController{store: store}
}

func RegisterPairingRoutes(r gin.IRoutes, store db.Store) {
	ctl := NewTvController(store)

	r.POST("/register", ctl.registerPairingCode)

	r.HEAD("/ping", ctl.pingServer)
	r.GET("/ping", ctl.pingServer)

	r.GET("/content", ctl.getContent)
}

// registerPairingCode binds a JSON pairing request, checks that the screen isn’t already paired,
// stores the pairing code in Redis, and responds with the device ID or an error.
func (t *TvController) registerPairingCode(ctx *gin.Context) {
	var request packets.TVRequest

	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		log.Error().Err(err).Msg("failed to bind JSON")
		return
	}

	isPaired, err := db.IsScreenPairedByDeviceID(&request.DeviceID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		log.Error().Err(err).Msg("failed to check if screen is paired by device")
		return
	}

	if isPaired == true {
		log.Error().Err(err).Msg("Screen is already paired")
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Screen is already paired"})
		return
	}

	pairingData := PairingData{
		DeviceID: request.DeviceID,
		IsPaired: false,
	}

	marshalledPairingData, _ := json.Marshal(pairingData)

	redis.Set(ctx, request.PairingCode, marshalledPairingData, 7*24*time.Hour)

	ctx.JSON(http.StatusOK, packets.TVRequest{DeviceID: request.DeviceID})
}

// pingServer
func (t *TvController) pingServer(ctx *gin.Context) {
	pairingCode := ctx.Query("code")
	var pairingData PairingData

	redis.GetUnmarshalledJSON(ctx, pairingCode, &pairingData)

	if pairingData.IsPaired == true {
		log.Info().Str("pairingCode", pairingCode).Bool("value", pairingData.IsPaired)
		ctx.JSON(http.StatusCreated, gin.H{})
		return
	} else {
		log.Info().Str("pairingCode", pairingCode).Bool("value", pairingData.IsPaired)
		ctx.JSON(http.StatusOK, gin.H{})
		return
	}
}

// getContent retrieves content for a TV device with ETag support
func (t *TvController) getContent(ctx *gin.Context) {
	deviceID := ctx.Query("device_id")
	if deviceID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "device_id is required"})
		return
	}

	screen, dbErr := db.GetScreenByDeviceID(&deviceID)
	if dbErr != nil {
		log.Error().Err(dbErr).Str("deviceID", deviceID).
			Msg("Device ID not found in both Redis and database")
		ctx.JSON(http.StatusNotFound, gin.H{"error": "device not found"})
		return
	}
	screenID := screen.ID

	// Get the playlist assigned to this screen first
	playlist, err := db.GetPlaylistForScreen(screenID)
	if err != nil {
		log.Error().Err(err).Int("screen_id", screenID).Msg("Failed to get playlist for screen")
		ctx.JSON(http.StatusNotFound, gin.H{"error": "no playlist assigned to screen"})
		return
	}

	// Check for persistent ETag in Redis using playlist ID
	etagKey := fmt.Sprintf("playlist:%d:etag", playlist.ID)
	storedETag, err := redis.Rdb.Get(ctx, etagKey).Result()

	// Check If-None-Match header against stored ETag
	ifNoneMatch := ctx.GetHeader("If-None-Match")
	if err == nil && storedETag != "" && ifNoneMatch == storedETag {
		log.Debug().Str("deviceID", deviceID).Int("screen_id", screenID).Int("playlist_id", playlist.ID).Str("etag", storedETag).
			Msg("Content not modified (cached playlist ETag), returning 304")
		ctx.Header("ETag", storedETag)
		ctx.Status(http.StatusNotModified)
		return
	}

	playlistName, contentItems, err := db.GetPlaylistContentForScreen(screenID)
	if err != nil {
		log.Error().Err(err).Int("screen_id", screenID).Msg("Failed to retrieve playlist content for screen")
		ctx.JSON(http.StatusNotFound, gin.H{"error": "no content found"})
		return
	}

	// Generate ETag based on current playlist content
	currentETag := generateContentETag(playlistName, contentItems)

	// Store the new ETag in Redis for future requests using playlist ID
	if err := redis.Rdb.Set(ctx, etagKey, currentETag, 0).Err(); err != nil {
		log.Warn().Err(err).Int("playlist_id", playlist.ID).Str("etag", currentETag).
			Msg("Failed to store playlist ETag in Redis")
	}

	// Final check against newly generated ETag (in case content hasn't changed but ETag wasn't cached)
	if ifNoneMatch == currentETag {
		log.Debug().Str("deviceID", deviceID).Int("screen_id", screenID).Int("playlist_id", playlist.ID).Str("etag", currentETag).
			Msg("Content not modified (generated playlist ETag), returning 304")
		ctx.Header("ETag", currentETag)
		ctx.Status(http.StatusNotModified)
		return
	}

	// Create response for TV client
	contentList := make([]adminpackets.TVContentItem, len(contentItems))
	for i, item := range contentItems {
		contentList[i] = adminpackets.TVContentItem{
			URL:      item.URL,
			Duration: item.Duration,
		}
	}

	response := adminpackets.TVPlaylistResponse{
		PlaylistName: playlistName,
		ContentList:  contentList,
	}

	// Set ETag header and return content
	ctx.Header("ETag", currentETag)
	ctx.Header("Cache-Control", "no-cache") // Allow caching but require revalidation

	log.Debug().Str("deviceID", deviceID).Int("screen_id", screenID).Int("playlist_id", playlist.ID).Str("etag", currentETag).
		Int("content_items", len(contentItems)).
		Msg("Returning content with playlist ETag")

	ctx.JSON(http.StatusOK, response)
}
