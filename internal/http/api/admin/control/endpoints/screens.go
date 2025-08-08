package endpoints

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/Nixie-Tech-LLC/medusa/internal/db"
	"github.com/Nixie-Tech-LLC/medusa/internal/http/api"
	"github.com/Nixie-Tech-LLC/medusa/internal/http/api/admin/control/packets"
	"github.com/Nixie-Tech-LLC/medusa/internal/model"
	"github.com/Nixie-Tech-LLC/medusa/internal/redis"
)


type TvController struct {
	store db.Store
}

func newTvController(store db.Store) *TvController {
	return &TvController{store: store}
}

// ScreenModule mounts all authenticated /screens endpoints.
func ScreenModule(store db.Store) api.Module {
	ctl := newTvController(store)
	return api.ModuleFunc(func(c *api.Controller) {
		// CRUD
		c.GET("/screens", 			ctl.listScreens)
		c.POST("/screens", 			ctl.createScreen)
		c.GET("/screens/:id", 		ctl.getScreen)
		c.PUT("/screens/:id", 		ctl.updateScreen)
		// screen <-> playlist
		c.GET("/screens/:id/playlist", 	ctl.getPlaylistForScreen)
		c.POST("/screens/:id/playlist", ctl.assignPlaylistToScreen)

		// pairing & assignment
		c.POST("/screens/pair", 		ctl.pairScreen)
		c.POST("/screens/:id/assign", 	ctl.assignScreenToUser)
	})
}

type PairingData struct {
	DeviceID string `json:"device_id"`
	IsPaired bool   `json:"is_paired"`
}

// GET /api/admin/screens
func (t *TvController) listScreens(ctx *gin.Context, user *model.User) (any, *api.APIError) {
	all, err := t.store.ListScreens()
	if err != nil {
		return nil, &api.APIError{Code: http.StatusInternalServerError, Message: err.Error()}
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
func (t *TvController) createScreen(ctx *gin.Context, user *model.User) (any, *api.APIError) {
	var request packets.CreateScreenRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		return nil, &api.APIError{Code: http.StatusBadRequest, Message: err.Error()}
	}

	screen, err := t.store.CreateScreen(request.Name, request.Location, user.ID)
	if err != nil {
		return nil, &api.APIError{Code: http.StatusInternalServerError, Message: "could not create screen"}
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
func (t *TvController) getScreen(ctx *gin.Context, user *model.User) (any, *api.APIError) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		log.Error().Err(err).Str("id_raw", ctx.Param("id")).Msg("invalid id in request")
		return nil, &api.APIError{Code: http.StatusBadRequest, Message: "invalid id"}
	}
	log.Info().Int("id", id).Msg("valid id received in request")

	screen, err := t.store.GetScreenByID(id)
	if err != nil {
		return nil, &api.APIError{Code: http.StatusNotFound, Message: "screen not found"}
	}

	if screen.CreatedBy != user.ID {
		log.Error().
			Int("user_id", user.ID).
			Int("screen_owner", screen.CreatedBy).
			Msg("forbidden access to screen")
		return nil, &api.APIError{Code: http.StatusForbidden, Message: "forbidden"}
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
func (t *TvController) updateScreen(ctx *gin.Context, user *model.User) (any, *api.APIError) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		log.Error().Err(err).Str("id_raw", ctx.Param("id")).Msg("invalid screen id in URL")
		return nil, &api.APIError{Code: http.StatusBadRequest, Message: "invalid id"}
	}

	existing, err := t.store.GetScreenByID(id)
	if err != nil {
		log.Error().Err(err).Int("screen_id", id).Msg("could not retrieve screen by id")
		return nil, &api.APIError{Code: http.StatusNotFound, Message: "screen not found"}
	}
	if existing.CreatedBy != user.ID {
		log.Error().Int("screen_id", id).Int("owner", existing.CreatedBy).Int("user_id", user.ID).
			Msg("user does not own the screen")
		return nil, &api.APIError{Code: http.StatusForbidden, Message: "forbidden"}
	}

	var req packets.UpdateScreenRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		log.Error().Err(err).Msg("invalid JSON in update screen request")
		return nil, &api.APIError{Code: http.StatusBadRequest, Message: err.Error()}
	}

	if err := t.store.UpdateScreen(id, req.Name, req.Location); err != nil {
		log.Error().Err(err).Int("screen_id", id).Msg("database update failed for screen")
		return nil, &api.APIError{Code: http.StatusInternalServerError, Message: "could not update screen"}
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
func (t *TvController) deleteScreen(ctx *gin.Context, user *model.User) (any, *api.APIError) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		log.Error().Err(err).Str("id_raw", ctx.Param("id")).Msg("invalid screen id in DELETE request")
		return nil, &api.APIError{Code: http.StatusBadRequest, Message: "invalid id"}
	}

	existing, err := t.store.GetScreenByID(id)
	if err != nil {
		log.Error().Err(err).Int("screen_id", id).Msg("screen not found during DELETE")
		return nil, &api.APIError{Code: http.StatusNotFound, Message: "screen not found"}
	}
	if existing.CreatedBy != user.ID {
		log.Error().
			Int("user_id", user.ID).
			Int("screen_created_by", existing.CreatedBy).
			Int("screen_id", existing.ID).
			Msg("permission denied: user does not own the screen")
		return nil, &api.APIError{Code: http.StatusForbidden, Message: "forbidden"}
	}

	if err := t.store.DeleteScreen(id); err != nil {
		log.Error().Err(err).Int("screen_id", id).Msg("could not delete screen")
		return nil, &api.APIError{Code: http.StatusInternalServerError, Message: "could not delete screen"}
	}

	return nil, nil
}

// POST /api/admin/screens/:id/assign
func (t *TvController) assignScreenToUser(ctx *gin.Context, user *model.User) (any, *api.APIError) {
	screenID, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		log.Error().Err(err).Str("param_screen_id", ctx.Param("id")).Msg("invalid screen id in assign")
		return nil, &api.APIError{Code: http.StatusBadRequest, Message: "invalid id"}
	}

	existing, err := t.store.GetScreenByID(screenID)
	if err != nil {
		log.Error().Err(err).Int("screen_id", screenID).Msg("screen not found during assign")
		return nil, &api.APIError{Code: http.StatusNotFound, Message: "screen not found"}
	}
	if existing.CreatedBy != user.ID {
		log.Warn().Int("screen_id", screenID).Int("requesting_user_id", user.ID).
			Int("screen_owner_id", existing.CreatedBy).Msg("unauthorized screen assignment attempt")
		return nil, &api.APIError{Code: http.StatusForbidden, Message: "forbidden"}
	}

	var req packets.AssignScreenRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		log.Error().Err(err).Str("route", ctx.FullPath()).Msg("invalid JSON for assign screen")
		return nil, &api.APIError{Code: http.StatusBadRequest, Message: err.Error()}
	}

	if err := t.store.AssignScreenToUser(screenID, req.UserID); err != nil {
		log.Error().Err(err).Int("screen_id", screenID).Int("target_user_id", req.UserID).
			Msg("failed to assign screen to user")
		return nil, &api.APIError{Code: http.StatusInternalServerError, Message: "could not assign screen"}
	}

	return nil, nil
}

// GET /api/admin/screens/:id/playlist
func (t *TvController) getPlaylistForScreen(ctx *gin.Context, user *model.User) (any, *api.APIError) {
	screenID, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		log.Error().Err(err).Str("route", ctx.FullPath()).Msg("invalid screen id in playlist retrieval")
		return nil, &api.APIError{Code: http.StatusBadRequest, Message: "invalid id"}
	}

	existingScreen, err := t.store.GetScreenByID(screenID)
	if err != nil {
		log.Error().Err(err).Int("screen_id", screenID).Str("route", ctx.FullPath()).
			Msg("failed to fetch screen in playlist retrieval")
		return nil, &api.APIError{Code: http.StatusNotFound, Message: "screen not found"}
	}

	if existingScreen.CreatedBy != user.ID {
		log.Warn().Int("user_id", user.ID).Int("screen_id", screenID).
			Msg("user attempted to access playlist for non-owned screen")
		return nil, &api.APIError{Code: http.StatusUnauthorized, Message: "unauthorized"}
	}

	playlist, err := t.store.GetPlaylistForScreen(screenID)
	if err != nil {
		// If no playlist is assigned, return null instead of error
		if err.Error() == "sql: no rows in result set" {
			return map[string]any{"playlist": nil}, nil
		}
		log.Error().Err(err).Int("screen_id", screenID).Str("route", ctx.FullPath()).
			Msg("failed to fetch playlist for screen")
		return nil, &api.APIError{Code: http.StatusInternalServerError, Message: "failed to fetch playlist"}
	}

	return map[string]any{"playlist": playlist}, nil
}

// POST /api/admin/screens/:id/playlist
func (t *TvController) assignPlaylistToScreen(ctx *gin.Context, user *model.User) (any, *api.APIError) {
	screenID, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		log.Error().Err(err).Str("route", ctx.FullPath()).Msg("invalid screen id in playlist assignment")
		return nil, &api.APIError{Code: http.StatusBadRequest, Message: "invalid id"}
	}

	existingScreen, err := t.store.GetScreenByID(screenID)
	if err != nil {
		log.Error().Err(err).Int("screen_id", screenID).Str("route", ctx.FullPath()).
			Msg("screen not found during playlist assignment")
		return nil, &api.APIError{Code: http.StatusNotFound, Message: "screen not found"}
	}
	if existingScreen.CreatedBy != user.ID {
		log.Warn().Int("screen_id", screenID).Int("requesting_user_id", user.ID).
			Int("screen_owner_id", existingScreen.CreatedBy).Str("route", ctx.FullPath()).
			Msg("unauthorized playlist assignment attempt")
		return nil, &api.APIError{Code: http.StatusForbidden, Message: "forbidden"}
	}

	var request packets.AssignPlaylistToScreenRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		log.Error().Err(err).Str("route", ctx.FullPath()).Msg("failed to bind JSON in playlist assignment")
		return nil, &api.APIError{Code: http.StatusBadRequest, Message: err.Error()}
	}

	existingPlaylist, err := t.store.GetPlaylistByID(request.PlaylistID)
	if err != nil {
		log.Error().Err(err).Int("playlist_id", request.PlaylistID).Str("route", ctx.FullPath()).
			Msg("playlist id not found during assignment")
		return nil, &api.APIError{Code: http.StatusNotFound, Message: "playlist not found"}
	}
	if existingPlaylist.CreatedBy != user.ID {
		log.Warn().Int("requesting_user_id", user.ID).Int("playlist_owner_id", existingPlaylist.CreatedBy).
			Int("playlist_id", existingPlaylist.ID).Str("route", ctx.FullPath()).
			Msg("unauthorized attempt to assign playlist: not owner")
		return nil, &api.APIError{Code: http.StatusForbidden, Message: "forbidden"}
	}

	// Get the old playlist before assignment for ETag invalidation
	oldPlaylist, oldErr := t.store.GetPlaylistForScreen(screenID)

	if err := t.store.AssignPlaylistToScreen(screenID, request.PlaylistID); err != nil {
		log.Error().Err(err).Int("screen_id", screenID).Int("playlist_id", request.PlaylistID).
			Str("route", ctx.FullPath()).Msg("failed to assign playlist to screen")
		return nil, &api.APIError{Code: http.StatusInternalServerError, Message: err.Error()}
	}

	// Invalidate ETag cache for both old and new playlists since assignments changed
	if oldErr == nil {
		oldEtagKey := fmt.Sprintf("playlist:%d:etag", oldPlaylist.ID)
		if err := redis.Rdb.Del(ctx, oldEtagKey).Err(); err != nil {
			log.Warn().Err(err).Int("old_playlist_id", oldPlaylist.ID).Str("etag_key", oldEtagKey).
				Msg("failed to invalidate old playlist ETag cache after assignment")
		} else {
			log.Debug().Int("old_playlist_id", oldPlaylist.ID).Str("etag_key", oldEtagKey).
				Msg("invalidated old playlist ETag cache after assignment")
		}
	}

	// Invalidate ETag for the new playlist
	newEtagKey := fmt.Sprintf("playlist:%d:etag", request.PlaylistID)
	if err := redis.Rdb.Del(ctx, newEtagKey).Err(); err != nil {
		log.Warn().Err(err).Int("new_playlist_id", request.PlaylistID).Str("etag_key", newEtagKey).
			Msg("failed to invalidate new playlist ETag cache after assignment")
	} else {
		log.Debug().Err(err).Int("new_playlist_id", request.PlaylistID).Str("etag_key", newEtagKey).
			Msg("invalidated new playlist ETag cache after assignment")
	}

	log.Info().Int("screen_id", screenID).Int("playlist_id", request.PlaylistID).
		Msg("successfully assigned playlist to screen")

	return gin.H{"message": "playlist assigned successfully"}, nil
}

// POST /api/admin/screens/pair
func (t *TvController) pairScreen(ctx *gin.Context, _ *model.User) (any, *api.APIError) {
	var request packets.PairScreenRequest
	var pairingData PairingData

	if err := ctx.ShouldBindJSON(&request); err != nil {
		log.Error().Err(err).Str("route", ctx.FullPath()).Msg("invalid JSON in screen pairing request")
		return nil, &api.APIError{Code: http.StatusBadRequest, Message: err.Error()}
	}

	key := request.PairingCode
	redis.GetUnmarshalledJSON(ctx, key, &pairingData)
	deviceID := pairingData.DeviceID

	pairingData.IsPaired = true
	updatedPairingData, _ := json.Marshal(pairingData)
	redis.Rdb.Set(ctx, key, updatedPairingData, 7*24*time.Hour)

	// Assign the deviceID to the screen in database
	if err := db.AssignDeviceIDToScreen(request.ScreenID, &deviceID); err != nil {
		log.Error().Err(err).Int("screen_id", request.ScreenID).Str("device_id", deviceID).
			Str("route", ctx.FullPath()).Msg("failed to assign device ID to screen during pairing")
		return nil, &api.APIError{Code: http.StatusInternalServerError, Message: "could not update screen device ID"}
	}

	if err := db.PairScreen(request.ScreenID); err != nil {
		log.Error().Err(err).Int("screen_id", request.ScreenID).Str("route", ctx.FullPath()).
			Msg("failed to mark screen as paired in database")
		return nil, &api.APIError{Code: http.StatusInternalServerError, Message: "could not update screen"}
	}

	log.Info().Str("device_id", deviceID).Int("screen_id", request.ScreenID).
		Msg("successfully paired screen and stored device mapping in Redis")

	return gin.H{"success": "screen paired successfully"}, nil
}

