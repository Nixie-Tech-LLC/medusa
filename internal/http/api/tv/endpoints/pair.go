package endpoints

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/Nixie-Tech-LLC/medusa/internal/db"
	adminpackets "github.com/Nixie-Tech-LLC/medusa/internal/http/api/admin/control/packets"
	"github.com/Nixie-Tech-LLC/medusa/internal/http/api"
	"github.com/Nixie-Tech-LLC/medusa/internal/http/api/tv/packets"
	"github.com/Nixie-Tech-LLC/medusa/internal/redis"
)

type TvController struct {
	store db.Store
}

func newTvController(store db.Store) *TvController {
	return &TvController{store: store}
}

// PairingModule mounts public TV endpoints: /register, /ping, /content
func PairingModule(store db.Store) api.Module {
	ctl := newTvController(store)
	return api.ModuleFunc(func(c *api.Controller) {
		c.Group.POST("/register", ctl.registerPairingCode)

		// support both HEAD/GET for ping
		c.Group.HEAD("/ping", ctl.pingServer)
		c.Group.GET("/ping", ctl.pingServer)

		c.Group.GET("/content", ctl.getContent)
	})
}


type PairingData struct {
	DeviceID string `json:"device_id"`
	IsPaired bool   `json:"is_paired"`
}

// generateContentETag creates an ETag based on playlist name and content items.
func generateContentETag(playlistName string, contentItems []db.ContentItem) string {
	hasher := sha256.New()
	hasher.Write([]byte(playlistName))
	buf := make([]byte, 0, 64) // pick a sensible capacity
	for _, item := range contentItems {
	    buf = buf[:0]
	    buf = fmt.Appendf(buf, "%s:%d", item.URL, item.Duration)
	    hasher.Write(buf)
	}
	hash := hex.EncodeToString(hasher.Sum(nil))
	// Use 24 chars (unquoted). We'll add quotes in the HTTP header.
	return hash[:24]
}

// POST /api/tv/register
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
	if isPaired {
		log.Error().Msg("screen is already paired")
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Screen is already paired"})
		return
	}

	pairingData := PairingData{DeviceID: request.DeviceID, IsPaired: false}
	marshalledPairingData, _ := json.Marshal(pairingData)
	redis.Set(ctx, request.PairingCode, marshalledPairingData, 7*24*time.Hour)

	ctx.JSON(http.StatusOK, packets.TVRequest{DeviceID: request.DeviceID})
}

// HEAD/GET /api/tv/ping?code=XXXX
func (t *TvController) pingServer(ctx *gin.Context) {
	pairingCode := ctx.Query("code")
	var pairingData PairingData
	redis.GetUnmarshalledJSON(ctx, pairingCode, &pairingData)

	if pairingData.IsPaired {
		log.Info().Str("pairingCode", pairingCode).Bool("value", pairingData.IsPaired).Msg("paired")
		ctx.JSON(http.StatusCreated, gin.H{})
		return
	}
	log.Info().Str("pairingCode", pairingCode).Bool("value", pairingData.IsPaired).Msg("not paired")
	ctx.JSON(http.StatusOK, gin.H{})
}

// GET /api/tv/content?device_id=UUID
func (t *TvController) getContent(ctx *gin.Context) {
	deviceID := ctx.Query("device_id")
	if deviceID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "device_id is required"})
		return
	}

	screen, dbErr := db.GetScreenByDeviceID(&deviceID)
	if dbErr != nil {
		log.Error().Err(dbErr).Str("deviceID", deviceID).
			Msg("Device ID not found")
		ctx.JSON(http.StatusNotFound, gin.H{"error": "device not found"})
		return
	}
	screenID := screen.ID

	// Fetch playlist assigned to this screen
	playlist, err := db.GetPlaylistForScreen(screenID)
	if err != nil {
		log.Error().Err(err).Int("screen_id", screenID).Msg("Failed to get playlist for screen")
		ctx.JSON(http.StatusNotFound, gin.H{"error": "no playlist assigned to screen"})
		return
	}

	etagKey := fmt.Sprintf("playlist:%d:etag", playlist.ID)
	storedETag, err := redis.Rdb.Get(ctx, etagKey).Result()

	// Respect both standard If-None-Match and custom X-If-None-Match (proxy-safe)
	ifNoneMatch := ctx.GetHeader("If-None-Match")
	if v := ctx.GetHeader("X-If-None-Match"); v != "" {
		ifNoneMatch = v
	}
	clientETag := ifNoneMatch
	if len(clientETag) >= 2 && clientETag[0] == '"' && clientETag[len(clientETag)-1] == '"' {
		clientETag = clientETag[1 : len(clientETag)-1]
	}

	if err == nil && storedETag != "" && clientETag == storedETag {
		quotedStored := fmt.Sprintf(`"%s"`, storedETag)
		log.Info().Str("deviceID", deviceID).Int("screen_id", screenID).Int("playlist_id", playlist.ID).
			Str("stored_etag", storedETag).Str("client_etag", ifNoneMatch).
			Msg("304 via stored ETag")
		ctx.Header("ETag", quotedStored)
		ctx.Header("X-Content-ETag", storedETag)
		ctx.Status(http.StatusNotModified)
		return
	}

	playlistName, contentItems, err := db.GetPlaylistContentForScreen(screenID)
	if err != nil {
		log.Error().Err(err).Int("screen_id", screenID).Msg("Failed to retrieve playlist content")
		ctx.JSON(http.StatusNotFound, gin.H{"error": "no content found"})
		return
	}

	// Compute and persist the new ETag
	currentETag := generateContentETag(playlistName, contentItems)
	if err := redis.Rdb.Set(ctx, etagKey, currentETag, 0).Err(); err != nil {
		log.Warn().Err(err).Int("playlist_id", playlist.ID).Str("etag", currentETag).
			Msg("Failed to store playlist ETag in Redis")
	}

	// If client already has the same ETag (but cache missed), still 304
	if clientETag == currentETag {
		quoted := fmt.Sprintf(`"%s"`, currentETag)
		log.Info().Str("deviceID", deviceID).Int("screen_id", screenID).Int("playlist_id", playlist.ID).
			Str("generated_etag", currentETag).Str("client_etag", ifNoneMatch).
			Msg("304 via generated ETag")
		ctx.Header("ETag", quoted)
		ctx.Header("X-Content-ETag", currentETag)
		ctx.Status(http.StatusNotModified)
		return
	}

	// Build response payload
	contentList := make([]adminpackets.TVContentItem, len(contentItems))
	for i, item := range contentItems {
		contentList[i] = adminpackets.TVContentItem{
			URL:      item.URL,
			Duration: item.Duration,
			Type:     item.Type,
		}
	}
	response := adminpackets.TVPlaylistResponse{
		PlaylistName: playlistName,
		ContentList:  contentList,
	}

	quoted := fmt.Sprintf(`"%s"`, currentETag)
	ctx.Header("ETag", quoted)
	ctx.Header("X-Content-ETag", currentETag) // proxy-safe
	ctx.Header("Cache-Control", "no-cache")

	log.Info().Str("deviceID", deviceID).Int("screen_id", screenID).Int("playlist_id", playlist.ID).
		Str("etag", quoted).Str("raw_etag", currentETag).
		Int("content_items", len(contentItems)).
		Msg("Returning content with playlist ETag")

	ctx.JSON(http.StatusOK, response)
}

