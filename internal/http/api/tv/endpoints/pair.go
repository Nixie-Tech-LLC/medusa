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

// PairingModule mounts public TV endpoints using api helpers.
func PairingModule(store db.Store) api.Module {
	ctl := newTvController(store)
	return api.ModuleFunc(func(c *api.Controller) {
		// Public routes via helpers
		c.PUBLIC_POST("/register", ctl.registerPairingCode)
		c.PUBLIC_GET("/ping", ctl.pingServer)

		// HEAD isn’t exposed on Controller; wire it via api.Public wrapper.
		c.Group.HEAD("/ping", api.Public(ctl.pingServer))

		c.PUBLIC_GET("/content", ctl.getContent)
	})
}

type PairingData struct {
	DeviceID string `json:"device_id"`
	IsPaired bool   `json:"is_paired"`
}

// generateContentETag hashes playlist name + ordered items (url:duration).
func generateContentETag(playlistName string, contentItems []db.ContentItem) string {
	hasher := sha256.New()
	hasher.Write([]byte(playlistName))
	buf := make([]byte, 0, 64)
	for _, item := range contentItems {
		buf = buf[:0]
		buf = fmt.Appendf(buf, "%s:%d", item.URL, item.Duration)
		hasher.Write(buf)
	}
	sum := hex.EncodeToString(hasher.Sum(nil))
	return sum[:24] // short but stable; quotes added when emitting headers
}

// POST /api/tv/register
func (t *TvController) registerPairingCode(ctx *gin.Context) (any, *api.APIError) {
	var req packets.TVRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		log.Error().Err(err).Str("path", ctx.FullPath()).Msg("[tv] registerPairingCode: invalid JSON")
		return nil, &api.APIError{Code: http.StatusBadRequest, Message: "invalid request body"}
	}
	if req.DeviceID == "" || req.PairingCode == "" {
		log.Warn().Str("device_id", req.DeviceID).Str("pairing_code", req.PairingCode).Msg("[tv] registerPairingCode: missing fields")
		return nil, &api.APIError{Code: http.StatusBadRequest, Message: "device_id and pairing_code are required"}
	}

	isPaired, err := t.store.IsScreenPairedByDeviceID(&req.DeviceID)
	if err != nil {
		log.Error().Err(err).Str("device_id", req.DeviceID).Msg("[tv] registerPairingCode: store.IsScreenPairedByDeviceID failed")
		return nil, &api.APIError{Code: http.StatusInternalServerError, Message: "could not verify pairing state"}
	}
	if isPaired {
		log.Warn().Str("device_id", req.DeviceID).Msg("[tv] registerPairingCode: already paired")
		return nil, &api.APIError{Code: http.StatusBadRequest, Message: "screen is already paired"}
	}

	payload := PairingData{DeviceID: req.DeviceID, IsPaired: false}
	raw, _ := json.Marshal(payload)

	redis.Set(ctx, req.PairingCode, raw, 7*24*time.Hour)
	
	log.Info().
	    Str("device_id", req.DeviceID).
	    Str("pairing_code", req.PairingCode).
	    Msg("[tv] registerPairingCode: pairing session created")
	
	return packets.TVRequest{DeviceID: req.DeviceID}, nil
}

// GET/HEAD /api/tv/ping?code=XXXX
func (t *TvController) pingServer(ctx *gin.Context) (any, *api.APIError) {
	code := ctx.Query("code")
	if code == "" {
		log.Warn().Msg("[tv] pingServer: missing pairing code")
		return nil, &api.APIError{Code: http.StatusBadRequest, Message: "code is required"}
	}

	var pairing PairingData

	redis.GetUnmarshalledJSON(ctx, code, &pairing)

	if pairing.IsPaired {
		log.Info().Str("path", ctx.FullPath()).Bool("paired", true).Msg("[tv] pingServer: paired")
		// api helpers always return 200 on success; surface paired state in body.
		return gin.H{"paired": true}, nil
	}

	log.Info().Str("path", ctx.FullPath()).Bool("paired", false).Msg("[tv] pingServer: not paired")
	return gin.H{"paired": false}, nil
}

// GET /api/tv/content?device_id=UUID
func (t *TvController) getContent(ctx *gin.Context) (any, *api.APIError) {
	deviceID := ctx.Query("device_id")
	if deviceID == "" {
		log.Warn().Msg("[tv] getContent: missing device_id")
		return nil, &api.APIError{Code: http.StatusBadRequest, Message: "device_id is required"}
	}

	screen, err := t.store.GetScreenByDeviceID(&deviceID)
	if err != nil {
		log.Error().Err(err).Str("device_id", deviceID).Msg("[tv] getContent: screen not found")
		return nil, &api.APIError{Code: http.StatusNotFound, Message: "device not found"}
	}
	screenID := screen.ID

	playlist, err := t.store.GetPlaylistForScreen(screenID)
	if err != nil {
		log.Warn().Err(err).Int("screen_id", screenID).Msg("[tv] getContent: no playlist assigned")
		return nil, &api.APIError{Code: http.StatusNotFound, Message: "no playlist assigned to screen"}
	}

	etagKey := fmt.Sprintf("playlist:%d:etag", playlist.ID)
	storedETag, redisErr := redis.Rdb.Get(ctx, etagKey).Result()

	ifNoneMatch := ctx.GetHeader("If-None-Match")
	if v := ctx.GetHeader("X-If-None-Match"); v != "" {
		ifNoneMatch = v
	}
	clientETag := trimETagQuotes(ifNoneMatch)

	if redisErr == nil && storedETag != "" && clientETag == storedETag {
		// Set headers and return 304 through APIError (logs as error, but honors helpers).
		quoted := quoteETag(storedETag)
		ctx.Header("ETag", quoted)
		ctx.Header("X-Content-ETag", storedETag)
		ctx.Header("Cache-Control", "no-cache")
		log.Info().
			Str("device_id", deviceID).
			Int("screen_id", screenID).
			Int("playlist_id", playlist.ID).
			Str("etag_match", storedETag).
			Msg("[tv] getContent: 304 via stored ETag")
		return nil, &api.APIError{Code: http.StatusNotModified, Message: "not modified"}
	}

	playlistName, contentItems, err := t.store.GetPlaylistContentForScreen(screenID)
	if err != nil {
		log.Error().Err(err).Int("screen_id", screenID).Msg("[tv] getContent: failed to load content")
		return nil, &api.APIError{Code: http.StatusNotFound, Message: "no content found"}
	}

	currentETag := generateContentETag(playlistName, contentItems)
	if err := redis.Rdb.Set(ctx, etagKey, currentETag, 0).Err(); err != nil {
		log.Warn().Err(err).Int("playlist_id", playlist.ID).Str("etag", currentETag).Msg("[tv] getContent: failed to cache ETag")
	}

	if clientETag == currentETag {
		quoted := quoteETag(currentETag)
		ctx.Header("ETag", quoted)
		ctx.Header("X-Content-ETag", currentETag)
		ctx.Header("Cache-Control", "no-cache")
		log.Info().
			Str("device_id", deviceID).
			Int("screen_id", screenID).
			Int("playlist_id", playlist.ID).
			Str("etag_match", currentETag).
			Msg("[tv] getContent: 304 via generated ETag")
		return nil, &api.APIError{Code: http.StatusNotModified, Message: "not modified"}
	}

	// Build response
	items := make([]adminpackets.TVContentItem, len(contentItems))
	for i, it := range contentItems {
		items[i] = adminpackets.TVContentItem{
			URL:      it.URL,
			Duration: it.Duration,
			Type:     it.Type,
		}
	}
	resp := adminpackets.TVPlaylistResponse{
		PlaylistName: playlistName,
		ContentList:  items,
	}

	quoted := quoteETag(currentETag)
	ctx.Header("ETag", quoted)
	ctx.Header("X-Content-ETag", currentETag)
	ctx.Header("Cache-Control", "no-cache")

	log.Info().
		Str("device_id", deviceID).
		Int("screen_id", screenID).
		Int("playlist_id", playlist.ID).
		Str("etag", currentETag).
		Int("items", len(contentItems)).
		Msg("[tv] getContent: returning playlist content")

	return resp, nil
}

func trimETagQuotes(v string) string {
	if len(v) >= 2 && v[0] == '"' && v[len(v)-1] == '"' {
		return v[1 : len(v)-1]
	}
	return v
}

func quoteETag(v string) string {
	return fmt.Sprintf(`"%s"`, v)
}

