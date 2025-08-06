package endpoints

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog/log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Nixie-Tech-LLC/medusa/internal/db"
	"github.com/Nixie-Tech-LLC/medusa/internal/http/api"
	"github.com/Nixie-Tech-LLC/medusa/internal/http/api/admin/packets"
	"github.com/Nixie-Tech-LLC/medusa/internal/model"
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

func RegisterScreenRoutes(r gin.IRoutes, store db.Store) {
	ctl := NewTvController(store)
	// all admin screens routes require a valid admin JWT
	r.GET("/screens", api.ResolveEndpointWithAuth(ctl.listScreens))
	r.POST("/screens", api.ResolveEndpointWithAuth(ctl.createScreen))
	r.GET("/screens/:id", api.ResolveEndpointWithAuth(ctl.getScreen))
	r.PUT("/screens/:id", api.ResolveEndpointWithAuth(ctl.updateScreen))
	r.DELETE("/screens/:id", api.ResolveEndpointWithAuth(ctl.deleteScreen))

	// screen <-> playlist
	r.GET("/screens/:id/playlist", api.ResolveEndpointWithAuth(ctl.getPlaylistForScreen))
	r.POST("/screens/:id/playlist", api.ResolveEndpointWithAuth(ctl.assignPlaylistToScreen))

	// pairing
	r.POST("/screens/pair", api.ResolveEndpointWithAuth(ctl.pairScreen))

	// assignment
	r.POST("/screens/:id/assign", api.ResolveEndpointWithAuth(ctl.assignScreenToUser))
}

// GET /api/admin/screens
func (t *TvController) listScreens(ctx *gin.Context, user *model.User) (any, *api.Error) {
	all, err := t.store.ListScreens()
	if err != nil {
		return nil, &api.Error{Code: http.StatusInternalServerError, Message: err.Error()}
	}

	out := make([]packets.ScreenResponse, 0, len(all))
	for _, s := range all {
		if s.CreatedBy != user.ID {
			continue
		}
		out = append(out, packets.ScreenResponse{
			ID:                s.ID,
			DeviceID:          s.DeviceID,
			ClientInformation: s.ClientInformation,
			ClientWidth:       s.ClientWidth,
			ClientHeight:      s.ClientHeight,
			Name:              s.Name,
			Location:          s.Location,
			Paired:            s.Paired,
			CreatedAt:         s.CreatedAt.Format(time.RFC3339),
			UpdatedAt:         s.UpdatedAt.Format(time.RFC3339),
		})
	}

	return out, nil
}

// POST /api/admin/screens
func (t *TvController) createScreen(ctx *gin.Context, user *model.User) (any, *api.Error) {
	var request packets.CreateScreenRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		return nil, &api.Error{Code: http.StatusBadRequest, Message: err.Error()}
	}

	screen, err := t.store.CreateScreen(request.Name, request.Location, user.ID)
	if err != nil {
		return nil, &api.Error{Code: http.StatusInternalServerError, Message: "could not create screen"}
	}

	return packets.ScreenResponse{
		ID:                screen.ID,
		DeviceID:          screen.DeviceID,
		ClientInformation: screen.ClientInformation,
		ClientWidth:       screen.ClientWidth,
		ClientHeight:      screen.ClientHeight,
		Name:              screen.Name,
		Location:          screen.Location,
		Paired:            screen.Paired,
		CreatedAt:         screen.CreatedAt.Format(time.RFC3339),
		UpdatedAt:         screen.UpdatedAt.Format(time.RFC3339),
	}, nil
}

// GET /api/admin/screens/:id
func (t *TvController) getScreen(ctx *gin.Context, user *model.User) (any, *api.Error) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		log.Error().Err(err).Int("id", id).Msg("Invalid id in request")
		return nil, &api.Error{Code: http.StatusBadRequest, Message: "invalid id"}
	}
	log.Info().Int("id", id).Msg("Valid id received in request") // example of information log

	screen, err := t.store.GetScreenByID(id)
	if err != nil {
		return nil, &api.Error{Code: http.StatusNotFound, Message: "screen not found"}
	}

	if screen.CreatedBy != user.ID {
		// TODO: add an error log after you answer the question plainly
		log.Printf("forbidden access: user %d tried to access screen created by user %d", user.ID, screen.CreatedBy)
		return nil, &api.Error{Code: http.StatusForbidden, Message: "forbidden"}
	}

	return packets.ScreenResponse{
		ID:                screen.ID,
		DeviceID:          screen.DeviceID,
		ClientInformation: screen.ClientInformation,
		ClientWidth:       screen.ClientWidth,
		ClientHeight:      screen.ClientHeight,
		Name:              screen.Name,
		Location:          screen.Location,
		Paired:            screen.Paired,
		CreatedAt:         screen.CreatedAt.Format(time.RFC3339),
		UpdatedAt:         screen.UpdatedAt.Format(time.RFC3339),
	}, nil
}

// PUT /api/admin/screens/:id
func (t *TvController) updateScreen(ctx *gin.Context, user *model.User) (any, *api.Error) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		log.Error().Err(err).Int("id", id).Msg("Invalid screen ID in URL: failed to convert to integer")
		return nil, &api.Error{Code: http.StatusBadRequest, Message: "invalid id"}
	}
	existing, err := t.store.GetScreenByID(id)
	if err != nil {
		log.Error().Err(err).Int("screen_id", id).Msg("Database lookup failed: could not retrieve screen by ID")
		return nil, &api.Error{Code: http.StatusNotFound, Message: "screen not found"}
	}
	if existing.CreatedBy != user.ID {
		log.Error().Err(err).Int("screen_id", id).Msg("Database lookup failed: user ID to screen does not match the Screen ID")
		return nil, &api.Error{Code: http.StatusForbidden, Message: "forbidden"}
	}
	var req packets.UpdateScreenRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		log.Error().Err(err).Msg("Invalid or malformed JSON in update screen request body")
		log.Info().Str("request_path", ctx.FullPath()).Msg("User submitted invalid JSON when attempting to update screen")
		return nil, &api.Error{Code: http.StatusBadRequest, Message: err.Error()}
	}
	if err := t.store.UpdateScreen(id, req.Name, req.Location); err != nil {
		log.Error().Err(err).Int("screen_id", id).
			Msg("Database update failed for screen")
		return nil, &api.Error{Code: http.StatusInternalServerError, Message: "could not update screen"}
	}
	updated, _ := t.store.GetScreenByID(id)

	return packets.ScreenResponse{
		ID:                updated.ID,
		DeviceID:          updated.DeviceID,
		ClientInformation: updated.ClientInformation,
		ClientWidth:       updated.ClientWidth,
		ClientHeight:      updated.ClientHeight,
		Name:              updated.Name,
		Location:          updated.Location,
		Paired:            updated.Paired,
		CreatedAt:         updated.CreatedAt.Format(time.RFC3339),
		UpdatedAt:         updated.UpdatedAt.Format(time.RFC3339),
	}, nil
}

// DELETE /api/admin/screens/:id
func (t *TvController) deleteScreen(ctx *gin.Context, user *model.User) (any, *api.Error) {

	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		log.Error().Err(err).Int("screen_id", id).Msg("Invalid screen ID in DELETE request: could not convert to integer")

		return nil, &api.Error{Code: http.StatusBadRequest, Message: "invalid id"}
	}

	existing, err := t.store.GetScreenByID(id)
	if err != nil {
		log.Error().
			Err(err).Int("screen_id", id).Msg("Store query failed: screen not found or inaccessible during DELETE request")
		log.Info().Int("screen_id", id).Msg("User attempted to delete a screen that does not exist or is unavailable")
		return nil, &api.Error{Code: http.StatusNotFound, Message: "screen not found"}
	}

	if existing.CreatedBy != user.ID {
		log.Error().
			Int("user_id", user.ID).
			Int("screen_created_by", existing.CreatedBy).
			Int("screen_id", existing.ID).
			Msg("Permission denied: user does not own the screen and cannot delete it")
		return nil, &api.Error{Code: http.StatusForbidden, Message: "forbidden"}
	}

	if err := t.store.DeleteScreen(id); err != nil {
		log.Error().Msg("Store delete failed: could not delete screen")
		return nil, &api.Error{Code: http.StatusInternalServerError, Message: "could not delete screen"}
	}

	return nil, nil
}

// POST /api/admin/screens/:id/assign
func (t *TvController) assignScreenToUser(ctx *gin.Context, user *model.User) (any, *api.Error) {

	screenID, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		log.Error().Err(err).Str("param_screen_id", ctx.Param("id")).
			Msg("Failed to convert screen ID from URL to integer during screen assignment")
		return nil, &api.Error{Code: http.StatusBadRequest, Message: "invalid id"}
	}

	existing, err := t.store.GetScreenByID(screenID)
	if err != nil {
		log.Error().Err(err).Int("screen_id", screenID).
			Msg("Database query failed: screen not found during assignment request")
		return nil, &api.Error{Code: http.StatusNotFound, Message: "screen not found"}
	}

	if existing.CreatedBy != user.ID {
		log.Warn().
			Int("screen_id", screenID).
			Int("requesting_user_id", user.ID).
			Int("screen_owner_id", existing.CreatedBy).
			Msg("Unauthorized screen assignment attempt: user does not own the screen")
		return nil, &api.Error{Code: http.StatusForbidden, Message: "forbidden"}
	}

	var req packets.AssignScreenRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		log.Error().Err(err).Str("route", ctx.FullPath()).
			Msg("Invalid JSON: failed to bind AssignScreenRequest")
		return nil, &api.Error{Code: http.StatusBadRequest, Message: err.Error()}
	}

	if err := t.store.AssignScreenToUser(screenID, req.UserID); err != nil {
		log.Error().Err(err).Int("screen_id", screenID).Int("target_user_id", req.UserID).
			Msg("Database error: failed to assign screen to user")
		return nil, &api.Error{Code: http.StatusInternalServerError, Message: "could not assign screen"}
	}

	return nil, nil
}

func (t *TvController) getPlaylistForScreen(ctx *gin.Context, user *model.User) (any, *api.Error) {
	screenID, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		log.Error().Err(err).Str("route", ctx.FullPath()).
			Msg("Invalid screen ID in URL: could not convert to integer during playlist retrieval")
		return nil, &api.Error{Code: http.StatusBadRequest, Message: "invalid id"}
	}

	existingScreen, err := t.store.GetScreenByID(screenID)
	if err != nil {
		log.Error().Err(err).Int("screen_id", screenID).Str("route", ctx.FullPath()).
			Msg("Failed to fetch screen: screen not found during playlist retrieval")
		return nil, &api.Error{Code: http.StatusNotFound, Message: "screen not found"}
	}

	// Check if the user is the creator of the screen
	if existingScreen.CreatedBy != user.ID {
		log.Warn().Int("user_id", user.ID).Int("screen_id", screenID).
			Msg("User attempted to access playlist for screen they do not own")
		return nil, &api.Error{Code: http.StatusUnauthorized, Message: "unauthorized"}
	}

	// Get the currently assigned playlist for this screen
	playlist, err := t.store.GetPlaylistForScreen(screenID)
	if err != nil {
		// If no playlist is assigned, return null instead of error
		if err.Error() == "sql: no rows in result set" {
			return map[string]interface{}{
				"playlist": nil,
			}, nil
		}
		log.Error().Err(err).Int("screen_id", screenID).Str("route", ctx.FullPath()).
			Msg("Failed to fetch playlist for screen")
		return nil, &api.Error{Code: http.StatusInternalServerError, Message: "failed to fetch playlist"}
	}

	return map[string]interface{}{
		"playlist": playlist,
	}, nil
}

func (t *TvController) assignPlaylistToScreen(ctx *gin.Context, user *model.User) (any, *api.Error) {
	screenID, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		log.Error().Err(err).Str("route", ctx.FullPath()).
			Msg("Invalid screen ID in URL: could not convert to integer during playlist assignment")
		return nil, &api.Error{Code: http.StatusBadRequest, Message: "invalid id"}
	}

	existingScreen, err := t.store.GetScreenByID(screenID)
	if err != nil {
		log.Error().Err(err).Int("screen_id", screenID).Str("route", ctx.FullPath()).
			Msg("Failed to fetch screen: screen not found during playlist assignment")
		return nil, &api.Error{Code: http.StatusNotFound, Message: "screen not found"}
	}

	if existingScreen.CreatedBy != user.ID {
		log.Warn().Int("screen_id", screenID).Int("requesting_user_id", user.ID).Int("screen_owner_id", existingScreen.CreatedBy).Str("route", ctx.FullPath()).
			Msg("Unauthorized playlist assignment attempt: user does not own the screen")
		return nil, &api.Error{Code: http.StatusForbidden, Message: "forbidden"}
	}

	var request packets.AssignPlaylistToScreenRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		log.Error().Err(err).Str("route", ctx.FullPath()).
			Msg("Failed to bind JSON body during playlist assignment")
		return nil, &api.Error{Code: http.StatusBadRequest, Message: err.Error()}
	}

	existingPlaylist, err := t.store.GetPlaylistByID(request.PlaylistID)
	if err != nil {
		log.Error().Err(err).Int("playlist_id", request.PlaylistID).Str("route", ctx.FullPath()).
			Msg("Playlist ID not found during assignment to screen")
		return nil, &api.Error{Code: http.StatusNotFound, Message: "playlist not found"}
	}

	if existingPlaylist.CreatedBy != user.ID {
		log.Warn().Int("requesting_user_id", user.ID).Int("playlist_owner_id", existingPlaylist.CreatedBy).Int("playlist_id", existingPlaylist.ID).Str("route", ctx.FullPath()).
			Msg("Unauthorized attempt to assign playlist: user does not own the playlist")
		return nil, &api.Error{Code: http.StatusForbidden, Message: "forbidden"}
	}

	// Get the old playlist before assignment for ETag invalidation
	oldPlaylist, oldErr := t.store.GetPlaylistForScreen(screenID)

	if err := t.store.AssignPlaylistToScreen(screenID, request.PlaylistID); err != nil {
		log.Error().Err(err).Int("screen_id", screenID).Int("playlist_id", request.PlaylistID).Str("route", ctx.FullPath()).
			Msg("Failed to assign playlist to screen")
		return nil, &api.Error{Code: http.StatusInternalServerError, Message: err.Error()}
	}

	// Invalidate ETag cache for both old and new playlists since assignments have changed
	if oldErr == nil {
		oldEtagKey := fmt.Sprintf("playlist:%d:etag", oldPlaylist.ID)
		if err := redis.Rdb.Del(ctx, oldEtagKey).Err(); err != nil {
			log.Warn().Err(err).Int("old_playlist_id", oldPlaylist.ID).Str("etag_key", oldEtagKey).
				Msg("Failed to invalidate old playlist ETag cache after assignment")
		} else {
			log.Debug().Int("old_playlist_id", oldPlaylist.ID).Str("etag_key", oldEtagKey).
				Msg("Invalidated old playlist ETag cache after assignment")
		}
	}

	// Invalidate ETag for the new playlist
	newEtagKey := fmt.Sprintf("playlist:%d:etag", request.PlaylistID)
	if err := redis.Rdb.Del(ctx, newEtagKey).Err(); err != nil {
		log.Warn().Err(err).Int("new_playlist_id", request.PlaylistID).Str("etag_key", newEtagKey).
			Msg("Failed to invalidate new playlist ETag cache after assignment")
	} else {
		log.Debug().Int("new_playlist_id", request.PlaylistID).Str("etag_key", newEtagKey).
			Msg("Invalidated new playlist ETag cache after assignment")
	}

	log.Info().Int("screen_id", screenID).Int("playlist_id", request.PlaylistID).
		Msg("Successfully assigned playlist to screen")

	return gin.H{"message": "playlist assigned successfully"}, nil
}

func (t *TvController) pairScreen(ctx *gin.Context, _ *model.User) (any, *api.Error) {
	var request packets.PairScreenRequest
	var pairingData PairingData
	if err := ctx.ShouldBindJSON(&request); err != nil {
		log.Error().Err(err).Str("route", ctx.FullPath()).
			Msg("Invalid JSON in screen pairing request")
		return nil, &api.Error{Code: http.StatusBadRequest, Message: err.Error()}
	}

	key := request.PairingCode
	redis.GetUnmarshalledJSON(ctx, key, &pairingData)
	deviceID := pairingData.DeviceID

	pairingData.IsPaired = true
	updatedPairingData, _ := json.Marshal(pairingData)

	redis.Rdb.Set(ctx, key, updatedPairingData, 7*24*time.Hour)

	// Assign the deviceID to the screen in database
	if err := db.AssignDeviceIDToScreen(request.ScreenID, &deviceID); err != nil {
		log.Error().Err(err).Int("screen_id", request.ScreenID).Str("device_id", deviceID).Str("route", ctx.FullPath()).
			Msg("Failed to assign device ID to screen during pairing")
		return nil, &api.Error{Code: http.StatusInternalServerError, Message: "could not update screen device ID"}
	}

	if err := db.PairScreen(request.ScreenID); err != nil {
		log.Error().Err(err).Int("screen_id", request.ScreenID).Str("route", ctx.FullPath()).
			Msg("Failed to mark screen as paired in database")
		return nil, &api.Error{Code: http.StatusInternalServerError, Message: "could not update screen"}
	}

	log.Info().Str("device_id", deviceID).Int("screen_id", request.ScreenID).
		Msg("Successfully paired screen and stored device mapping in Redis")

	return gin.H{"success": "screen paired successfully"}, nil
}
