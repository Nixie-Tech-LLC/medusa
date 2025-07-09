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
		return nil, &api.Error{Code: http.StatusBadRequest, Message: "invalid id"}
	}
	log.Error().Err(err).Msg("Failed to get screen")
	s, err := t.store.GetScreenByID(id)
	if err != nil {
		return nil, &api.Error{Code: http.StatusNotFound, Message: "screen not found"}
		log.Error().Err(err).Msg("Failed to get screen by ID")
	}
	if s.CreatedBy != user.ID {
		return nil, &api.Error{Code: http.StatusForbidden, Message: "forbidden"}
	}

	return packets.ScreenResponse{
		ID:        s.ID,
		DeviceID:  s.DeviceID,
		Name:      s.Name,
		Location:  s.Location,
		Paired:    s.Paired,
		CreatedAt: s.CreatedAt.Format(time.RFC3339),
		UpdatedAt: s.UpdatedAt.Format(time.RFC3339),
	}, nil
}

// PUT /api/admin/screens/:id
func (t *TvController) updateScreen(ctx *gin.Context, user *model.User) (any, *api.Error) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		return nil, &api.Error{Code: http.StatusBadRequest, Message: "invalid id"}
	}

	existing, err := t.store.GetScreenByID(id)
	if err != nil {
		return nil, &api.Error{Code: http.StatusNotFound, Message: "screen not found"}
		log.Error().Err(err).Msg("Failed to get screen by ID")
	}
	if existing.CreatedBy != user.ID {
		return nil, &api.Error{Code: http.StatusForbidden, Message: "forbidden"}
	}

	var req packets.UpdateScreenRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		log.Error().Err(err).Msg("Failed to bind JSON body")
		return nil, &api.Error{Code: http.StatusBadRequest, Message: err.Error()}
	}

	if err := t.store.UpdateScreen(id, req.Name, req.Location); err != nil {
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
		return nil, &api.Error{Code: http.StatusBadRequest, Message: "invalid id"}
		log.Error().Err(err).Msg("Failed delete screen")
	}

	existing, err := t.store.GetScreenByID(id)
	if err != nil {
		return nil, &api.Error{Code: http.StatusNotFound, Message: "screen not found"}
		log.Error().Err(err).Msg("Failed to get screen by ID")
	}
	if existing.CreatedBy != user.ID {
		return nil, &api.Error{Code: http.StatusForbidden, Message: "forbidden"}
	}

	if err := t.store.DeleteScreen(id); err != nil {
		return nil, &api.Error{Code: http.StatusInternalServerError, Message: "could not delete screen"}
	}
	return nil, nil
}

// POST /api/admin/screens/:id/assign
func (t *TvController) assignScreenToUser(ctx *gin.Context, user *model.User) (any, *api.Error) {
	screenID, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		return nil, &api.Error{Code: http.StatusBadRequest, Message: "invalid id"}
		log.Error().Err(err).Msg("Failed assign screen to user")
	}

	existing, err := t.store.GetScreenByID(screenID)
	if err != nil {
		return nil, &api.Error{Code: http.StatusNotFound, Message: "screen not found"}
		log.Error().Err(err).Msg("Failed to get screen by ID")
	}
	if existing.CreatedBy != user.ID {
		return nil, &api.Error{Code: http.StatusForbidden, Message: "forbidden"}
	}

	var req packets.AssignScreenRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		log.Error().Err(err).Msg("Failed to bind JSON body")
		return nil, &api.Error{Code: http.StatusBadRequest, Message: err.Error()}
	}
	if err := t.store.AssignScreenToUser(screenID, req.UserID); err != nil {
		log.Error().Err(err).Msg("Failed to assign screen to user")
		return nil, &api.Error{Code: http.StatusInternalServerError, Message: "could not assign screen"}
	}
	return nil, nil
}

func (t *TvController) getContentForScreen(ctx *gin.Context, user *model.User) (any, *api.Error) {
	screenID, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		return nil, &api.Error{Code: http.StatusBadRequest, Message: "invalid id"}
		log.Error().Err(err).Msg("Failed get content for screen")
	}

	existing, err := t.store.GetScreenByID(screenID)
	if err != nil {
		return nil, &api.Error{Code: http.StatusNotFound, Message: "screen not found"}
		log.Error().Err(err).Msg("Failed to get screen by ID")
	}
	if existing.CreatedBy != user.ID {
		return nil, &api.Error{Code: http.StatusForbidden, Message: "forbidden"}
	}

	content, err := t.store.GetContentForScreen(screenID)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get content for screen")
		return nil, &api.Error{Code: http.StatusNotFound, Message: "no content assigned"}
	}
	return packets.ContentResponse{
		ID:        content.ID,
		Name:      content.Name,
		Type:      content.Type,
		URL:       content.URL,
		CreatedAt: content.CreatedAt.Format(time.RFC3339),
	}, nil
}

func (t *TvController) assignContentToScreen(ctx *gin.Context, user *model.User) (any, *api.Error) {
	screenID, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		return nil, &api.Error{Code: http.StatusBadRequest, Message: "invalid id"}
		log.Error().Err(err).Msg("Failed assign content to screen")
	}

	existingScreen, err := t.store.GetScreenByID(screenID)
	if err != nil {
		return nil, &api.Error{Code: http.StatusNotFound, Message: "screen not found"}
	}
	if existingScreen.CreatedBy != user.ID {
		log.Error().Int("current user", user.ID).Int("screen owner", existingScreen.CreatedBy).Msg("The screen was not created by the user")
		return nil, &api.Error{Code: http.StatusForbidden, Message: "forbidden"}
	}

	var request packets.AssignContentToScreenRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		log.Error().Err(err).Msg("Failed to bind JSON body")
		return nil, &api.Error{Code: http.StatusBadRequest, Message: err.Error()}
	}
	existingContent, err := t.store.GetContentByID(request.ContentID)
	if err != nil {
		return nil, &api.Error{Code: http.StatusNotFound, Message: "content not found"}
		log.Error().Err(err).Msg("Failed to get content for screen")
	}
	if existingContent.CreatedBy != user.ID {
		return nil, &api.Error{Code: http.StatusForbidden, Message: "forbidden"}
	}

	if err := t.store.AssignContentToScreen(screenID, request.ContentID); err != nil {
		log.Error().Err(err).Msg("Failed to assign content to screen")
		return nil, &api.Error{Code: http.StatusInternalServerError, Message: err.Error()}
	}
	content, err := t.store.GetContentForScreen(screenID)
	log.Error().Err(err).Msg("Failed to get content to screen")
	if err != nil {
		return nil, &api.Error{Code: http.StatusInternalServerError, Message: err.Error()}
	}

	screen, err := t.store.GetScreenByID(screenID)
	if err != nil || screen.DeviceID == nil {
		log.Error().Err(err).Msg("Failed to get screen by ID")
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
			return nil, &api.Error{Code: http.StatusInternalServerError, Message: err.Error()}
			log.Error().Err(err).Msg("Failed to send message to screen")
		}
	}

	return nil, nil
}

func (t *TvController) pairScreen(ctx *gin.Context, _ *model.User) (any, *api.Error) {
	var request packets.PairScreenRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		log.Error().Err(err).Msg("Failed to bind JSON body")
		return nil, &api.Error{Code: http.StatusBadRequest, Message: err.Error()}
	}

	// Assign the deviceID to the screen in Redis
	key := request.PairingCode
	deviceID, err := redis.Rdb.Get(ctx, key).Result()
	if err != nil {
		return nil, &api.Error{Code: http.StatusInternalServerError, Message: "could not find deviceID for pairing code"}
		log.Error().Err(err).Msg("Failed to get device ID for pairing code")
	}
	redis.Rdb.Del(ctx, key)

	if err := db.AssignDeviceIDToScreen(request.ScreenID, &deviceID); err != nil {
		log.Error().Err(err).Msg("Failed to assign device ID to screen")
		return nil, &api.Error{Code: http.StatusInternalServerError, Message: "could not update screen device ID"}
	}

	if err := db.PairScreen(request.ScreenID); err != nil {
		log.Error().Err(err).Msg("Failed to pair screen")
		return nil, &api.Error{Code: http.StatusInternalServerError, Message: "could not update screen"}
	}

	return gin.H{"success": "screen paired successfully"}, nil
}
