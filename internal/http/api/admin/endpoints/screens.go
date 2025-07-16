package endpoints

import (
	"encoding/json"
	"github.com/rs/zerolog/log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Nixie-Tech-LLC/medusa/internal/db"
	"github.com/Nixie-Tech-LLC/medusa/internal/http/api"
	"github.com/Nixie-Tech-LLC/medusa/internal/http/api/admin/packets"
	"github.com/Nixie-Tech-LLC/medusa/internal/http/middleware"
	"github.com/Nixie-Tech-LLC/medusa/internal/model"
	"github.com/Nixie-Tech-LLC/medusa/internal/redis"
)

type TvController struct {
	store db.Store
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

	// screen <-> content
	r.GET("/screens/:id/content", api.ResolveEndpointWithAuth(ctl.getContentForScreen))
	r.POST("/screens/:id/content", api.ResolveEndpointWithAuth(ctl.assignContentToScreen))

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
			ID:        s.ID,
			DeviceID:  s.DeviceID,
			Name:      s.Name,
			Location:  s.Location,
			Paired:    s.Paired,
			CreatedAt: s.CreatedAt.Format(time.RFC3339),
			UpdatedAt: s.UpdatedAt.Format(time.RFC3339),
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

	_, err := middleware.CreateMQTTClient(request.Name)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create MQTT client")
		return nil, &api.Error{Code: http.StatusBadRequest, Message: err.Error()}
	}

	screen, err := t.store.CreateScreen(request.Name, request.Location, user.ID)
	if err != nil {
		return nil, &api.Error{Code: http.StatusInternalServerError, Message: "could not create screen"}
	}

	return packets.ScreenResponse{
		ID:        screen.ID,
		DeviceID:  screen.DeviceID,
		Name:      screen.Name,
		Location:  screen.Location,
		Paired:    screen.Paired,
		CreatedAt: screen.CreatedAt.Format(time.RFC3339),
		UpdatedAt: screen.UpdatedAt.Format(time.RFC3339),
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
		ID:        screen.ID,
		DeviceID:  screen.DeviceID,
		Name:      screen.Name,
		Location:  screen.Location,
		Paired:    screen.Paired,
		CreatedAt: screen.CreatedAt.Format(time.RFC3339),
		UpdatedAt: screen.UpdatedAt.Format(time.RFC3339),
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
		ID:        updated.ID,
		DeviceID:  updated.DeviceID,
		Name:      updated.Name,
		Location:  updated.Location,
		Paired:    updated.Paired,
		CreatedAt: updated.CreatedAt.Format(time.RFC3339),
		UpdatedAt: updated.UpdatedAt.Format(time.RFC3339),
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

func (t *TvController) getContentForScreen(ctx *gin.Context, user *model.User) (any, *api.Error) {

	screenID, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		log.Error().Err(err).Str("route", ctx.FullPath()).
			Msg("Invalid screen ID in URL: could not convert to integer during get content request")
		return nil, &api.Error{Code: http.StatusBadRequest, Message: "invalid id"}
	}

	existing, err := t.store.GetScreenByID(screenID)
	if err != nil {
		log.Error().Err(err).Int("screen_id", screenID).Str("route", ctx.FullPath()).
			Msg("Database query failed: screen not found during get content request")
		return nil, &api.Error{Code: http.StatusNotFound, Message: "screen not found"}
	}

	if existing.CreatedBy != user.ID {
		log.Warn().Int("screen_id", existing.ID).Int("requesting_user_id", user.ID).Int("screen_owner_id", existing.CreatedBy).Str("route", ctx.FullPath()).
			Msg("Unauthorized access attempt: user is not the creator of the screen")
		return nil, &api.Error{Code: http.StatusForbidden, Message: "forbidden"}
	}

	content, err := t.store.GetContentForScreen(screenID)
	if err != nil {
		log.Info().Int("screen_id", screenID).Str("route", ctx.FullPath()).
			Msg("No content assigned to screen")
		return nil, &api.Error{Code: http.StatusNotFound, Message: "no content assigned"}
	}

	return packets.ContentResponse{
		ID:        content.ID,
		Name:      content.Name,
		Type:      content.Type,
		URL:       content.URL,
		CreatedAt: content.CreatedAt.Format(time.RFC3339),
		//do we need a log here?
	}, nil
}

func (t *TvController) assignContentToScreen(ctx *gin.Context, user *model.User) (any, *api.Error) {

	screenID, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		log.Error().Err(err).Str("route", ctx.FullPath()).
			Msg("Invalid screen ID in URL: could not convert to integer during content assignment")
		return nil, &api.Error{Code: http.StatusBadRequest, Message: "invalid id"}
	}

	existingScreen, err := t.store.GetScreenByID(screenID)
	if err != nil {
		log.Error().
			Err(err).Int("screen_id", screenID).Str("route", ctx.FullPath()).
			Msg("Failed to fetch screen: screen not found during content assignment")
		return nil, &api.Error{Code: http.StatusNotFound, Message: "screen not found"}
	}

	if existingScreen.CreatedBy != user.ID {
		log.Warn().Int("screen_id", screenID).Int("requesting_user_id", user.ID).Int("screen_owner_id", existingScreen.CreatedBy).Str("route", ctx.FullPath()).
			Msg("Unauthorized content assignment attempt: user does not own the screen")
		return nil, &api.Error{Code: http.StatusForbidden, Message: "forbidden"}
	}

	var request packets.AssignContentToScreenRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		log.Error().Err(err).Str("route", ctx.FullPath()).
			Msg("Failed to bind JSON body during content assignment")
		return nil, &api.Error{Code: http.StatusBadRequest, Message: err.Error()}
	}

	existingContent, err := t.store.GetContentByID(request.ContentID)
	if err != nil {
		log.Error().Err(err).Int("content_id", request.ContentID).Str("route", ctx.FullPath()).
			Msg("Content ID not found during assignment to screen")
		return nil, &api.Error{Code: http.StatusNotFound, Message: "content not found"}
	}

	if existingContent.CreatedBy != user.ID {
		log.Warn().Int("requesting_user_id", user.ID).Int("content_owner_id", existingContent.CreatedBy).Int("content_id", existingContent.ID).Str("route", ctx.FullPath()).
			Msg("Unauthorized attempt to assign content: user does not own the content")
		return nil, &api.Error{Code: http.StatusForbidden, Message: "forbidden"}
	}

	if err := t.store.AssignContentToScreen(screenID, request.ContentID); err != nil {
		log.Error().Err(err).Int("screen_id", screenID).Int("content_id", request.ContentID).Str("route", ctx.FullPath()).
			Msg("Failed to assign content to screen")
		return nil, &api.Error{Code: http.StatusInternalServerError, Message: err.Error()}
	}

	content, err := t.store.GetContentForScreen(screenID)
	log.Error().Err(err).Int("screen_id", screenID).Str("route", ctx.FullPath()).
		Msg("Failed to retrieve assigned content for screen")
	if err != nil {
		return nil, &api.Error{Code: http.StatusInternalServerError, Message: err.Error()}
	}

	screen, err := t.store.GetScreenByID(screenID)
	if err != nil || screen.DeviceID == nil {
		log.Error().Err(err).Int("screen_id", screenID).Str("route", ctx.FullPath()).
			Msg("Failed to retrieve screen or missing device ID during message dispatch")
		return nil, &api.Error{Code: http.StatusInternalServerError, Message: "screen does not exist"}
	}

	response, err := json.Marshal(packets.ContentResponse{
		ID:        content.ID,
		Name:      content.Name,
		Type:      content.Type,
		URL:       content.URL,
		CreatedAt: content.CreatedAt.String(),
	})

	if err == nil {
		err := middleware.SendMessageToScreen(*screen.DeviceID, response)
		log.Error().Err(err).Msg("Failed to send message to screen")
		if err != nil {
			log.Error().Err(err).Str("route", ctx.FullPath()).Int("screen_id", screenID).
				Msg("Failed to send content update message to screen device")
			return nil, &api.Error{Code: http.StatusInternalServerError, Message: err.Error()}
		}
	}

	return nil, nil
}

func (t *TvController) pairScreen(ctx *gin.Context, _ *model.User) (any, *api.Error) {

	var request packets.PairScreenRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		log.Error().Err(err).Str("route", ctx.FullPath()).
			Msg("Invalid JSON in screen pairing request")
		return nil, &api.Error{Code: http.StatusBadRequest, Message: err.Error()}
	}

	// Assign the deviceID to the screen in Redis
	key := request.PairingCode
	deviceID, err := redis.Rdb.Get(ctx, key).Result()
	if err != nil {
		log.Error().Err(err).Str("pairing_code", key).Str("route", ctx.FullPath()).
			Msg("Could not find device ID for pairing code in Redis")
		return nil, &api.Error{Code: http.StatusInternalServerError, Message: "could not find deviceID for pairing code"}
	}
	redis.Rdb.Del(ctx, key)

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

	return gin.H{"success": "screen paired successfully"}, nil
}
