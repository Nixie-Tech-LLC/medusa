package endpoints

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
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
		log.Error().Err(dbErr).Str("deviceID", deviceID).Msg("Device ID not found")
		ctx.JSON(http.StatusNotFound, gin.H{"error": "device not found"})
		return
	}
	screenID := screen.ID

	now := time.Now().UTC()
	playlist, contentItems, source, err := db.GetEffectivePlaylistForScreen(screenID, now)
	if err != nil {
	    ctx.Header("X-Debug-Why", "no active schedule window and no direct playlist")
	    ctx.JSON(http.StatusNotFound, gin.H{"error": "no playlist active for this screen right now"})
	    return
	}

	// ETag: include playlist ID + updatedAt + items
	currentETag := generatePlaylistETag(playlist.ID, playlist.UpdatedAt, contentItems)

	etagKey := fmt.Sprintf("playlist:%d:etag", playlist.ID)
	storedETag, _ := redis.Rdb.Get(ctx, etagKey).Result()
	if storedETag != currentETag {
		_ = redis.Rdb.Set(ctx, etagKey, currentETag, 0).Err()
	}

	ifNoneMatch := ctx.GetHeader("If-None-Match")
	if v := ctx.GetHeader("X-If-None-Match"); v != "" {
		ifNoneMatch = v
	}
	clientETag := strings.Trim(ifNoneMatch, `"`)

	if clientETag == currentETag {
		ctx.Header("ETag", `"`+currentETag+`"`)
		ctx.Header("X-Content-ETag", currentETag)
		ctx.Header("X-Content-Source", source) // "schedule" or "direct"
		ctx.Status(http.StatusNotModified)
		return
	}

	contentList := make([]adminpackets.TVContentItem, len(contentItems))
	for i, item := range contentItems {
		contentList[i] = adminpackets.TVContentItem{
			URL:      item.URL,
			Duration: item.Duration,
			Type:     item.Type,
		}
	}
	response := adminpackets.TVPlaylistResponse{
		PlaylistName: playlist.Name,
		ContentList:  contentList,
	}

	ctx.Header("ETag", `"`+currentETag+`"`)
	ctx.Header("X-Content-ETag", currentETag)
	ctx.Header("X-Content-Source", source) // nice for debugging
	ctx.Header("Cache-Control", "no-cache")
	ctx.JSON(http.StatusOK, response)
}

func generatePlaylistETag(playlistID int, updatedAt time.Time, contentItems []db.ContentItem) string {
	h := sha256.New()
	var buf []byte

	buf = fmt.Appendf(nil, "pid:%d;upts:%d;", playlistID, updatedAt.Unix())
	h.Write(buf)

	for _, it := range contentItems {
		buf = buf[:0]
		buf = fmt.Appendf(buf, "%s:%d;", it.URL, it.Duration)
		h.Write(buf)
	}
	sum := hex.EncodeToString(h.Sum(nil))
	return sum[:24]
}
