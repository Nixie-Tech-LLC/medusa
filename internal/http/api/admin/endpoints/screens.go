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
	// 	creates id variable, creates err variable
	// 	id = gets the id from the endpoint url and sets it equal to the variable id
	id, err := strconv.Atoi(ctx.Param("id"))
	// if err is equal to something, return STATUSBADREQUEST response (400 error code)
	if err != nil {
		log.Error().Err(err).Int("id", id).Msg("Invalid id in request")
		return nil, &api.Error{Code: http.StatusBadRequest, Message: "invalid id"}
	}
	// Question: What would the log be for when the id is valid?
	// Answer: {level: info, timestamp: ____, id: {id}, message: "Valid id received in request"}
	log.Info().Int("id", id).Msg("Valid id received in request") // example of information log

	// creates variable s and err
	// searches the database, finds the screen and puts it inside the "screen" variable
	screen, err := t.store.GetScreenByID(id)
	if err != nil {
		return nil, &api.Error{Code: http.StatusNotFound, Message: "screen not found"}
	}

	// Question: What does the if statement do in english?
	// Hint: If user with id 3 creates screen with name "x", CreatedBy for screen "x" is set to 3
	// Answer: The if statement checks whether the user who is trying to access or modify the screen is not the same person who originally created it.
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
	// Question: What are we doing with this line in plain enlgish?
	// A: We're taking the "id" part of the URL and turning it into a number so we can use it in the code.
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		log.Error().Err(err).Int("id", id).Msg("Invalid screen ID in URL: failed to convert to integer")
		return nil, &api.Error{Code: http.StatusBadRequest, Message: "invalid id"}
	}

	// Question: Why are we trying to get the screen from the store using this ID?
	// A: We have to check that this screen exists before trying to update it.
	existing, err := t.store.GetScreenByID(id)
	if err != nil {
		log.Error().Err(err).Int("screen_id", id).Msg("Database lookup failed: could not retrieve screen by ID")
		return nil, &api.Error{Code: http.StatusNotFound, Message: "screen not found"}
	}

	// Question: What is this check doing?
	// Hint: We are making sure that only the person who created the screen is allowed to update it.
	if existing.CreatedBy != user.ID {
		log.Error().Err(err).Int("screen_id", id).Msg("Database lookup failed: user ID to screen does not match the Screen ID")
		return nil, &api.Error{Code: http.StatusForbidden, Message: "forbidden"}
	}

	// Question: What is this line doing with the JSON?
	// A: it attempts to bind the incoming JSON data from the HTTP request to a UpdateScreenRequest structure. If the JSON is bad or missing required info, it returns an error.
	var req packets.UpdateScreenRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		log.Error().Err(err).Msg("Invalid or malformed JSON in update screen request body")
		log.Info().Str("request_path", ctx.FullPath()).Msg("User submitted invalid JSON when attempting to update screen")
		return nil, &api.Error{Code: http.StatusBadRequest, Message: err.Error()}
	}

	// Question: Why are we calling this function to update the screen?
	// A: Now that we have the new info from the request, we want to save those changes to the database.
	if err := t.store.UpdateScreen(id, req.Name, req.Location); err != nil {
		log.Error().Err(err).Int("screen_id", id).
			Msg("Database update failed for screen")
		return nil, &api.Error{Code: http.StatusInternalServerError, Message: "could not update screen"}
	}

	// Question: Why are we retrieving the screen again here?
	// A: We want to return the most up-to-date version of the screen after making changes.
	updated, _ := t.store.GetScreenByID(id)

	// Question: What is this return value doing?
	// A: It's sending back the updated screen information in a format the client understands.
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
	// Question: What is this line doing?
	// A: It's pulling the screen ID from the URL and converting it from a string to an integer.
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		log.Error().Err(err).Int("screen_id", id).Msg("Invalid screen ID in DELETE request: could not convert to integer")
		//every time I try to add ID for the param_id to the error code it gives me an error - Im not sure maybe str isnt the right format?

		return nil, &api.Error{Code: http.StatusBadRequest, Message: "invalid id"}
	}

	// Question: Why do we try to get the screen from the store before deleting it?
	// A: We need to make sure the screen actually exists before trying to delete it.
	existing, err := t.store.GetScreenByID(id)
	if err != nil {
		log.Error().
			Err(err).Int("screen_id", id).Msg("Store query failed: screen not found or inaccessible during DELETE request")
		log.Info().Int("screen_id", id).Msg("User attempted to delete a screen that does not exist or is unavailable")
		return nil, &api.Error{Code: http.StatusNotFound, Message: "screen not found"}
	}
	// Question: Why do we check who created the screen?
	// A: Only the user who created the screen should be allowed to delete it.
	if existing.CreatedBy != user.ID {
		log.Error().
			Int("user_id", user.ID).
			Int("screen_created_by", existing.CreatedBy).
			Int("screen_id", existing.ID).
			Msg("Permission denied: user does not own the screen and cannot delete it")
		return nil, &api.Error{Code: http.StatusForbidden, Message: "forbidden"}
	}

	// Question: What does this line do?
	// A: It deletes the screen with the given ID from the database.
	if err := t.store.DeleteScreen(id); err != nil {
		log.Error().Msg("Store delete failed: could not delete screen")
		return nil, &api.Error{Code: http.StatusInternalServerError, Message: "could not delete screen"}
	}
	// Question: What does this return statement mean?
	// A: We're not returning any data—just confirming the deletion was successful.
	return nil, nil
}

// POST /api/admin/screens/:id/assign
func (t *TvController) assignScreenToUser(ctx *gin.Context, user *model.User) (any, *api.Error) {
	// Question: What is this line doing?
	// A: It gets the screen ID from the URL path and tries to convert it into a usable integer.
	screenID, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		log.Error().Err(err).Str("param_screen_id", ctx.Param("id")).
			Msg("Failed to convert screen ID from URL to integer during screen assignment")
		return nil, &api.Error{Code: http.StatusBadRequest, Message: "invalid id"}
	}

	// Question: Why do we need to retrieve the screen by its ID before assigning it?
	// A: We have to make sure the screen exists and check who created it before making changes.
	existing, err := t.store.GetScreenByID(screenID)
	if err != nil {
		log.Error().Err(err).Int("screen_id", screenID).
			Msg("Database query failed: screen not found during assignment request")
		return nil, &api.Error{Code: http.StatusNotFound, Message: "screen not found"}
	}
	// Question: Why do we check if the current user created the screen?
	// A: Only the user who created a screen should be allowed to reassign it to someone else.
	if existing.CreatedBy != user.ID {
		log.Warn().
			Int("screen_id", screenID).
			Int("requesting_user_id", user.ID).
			Int("screen_owner_id", existing.CreatedBy).
			Msg("Unauthorized screen assignment attempt: user does not own the screen")
		return nil, &api.Error{Code: http.StatusForbidden, Message: "forbidden"}
	}

	// Question: What does this line do with the request body?
	// A: It attempts to parse the JSON body into a struct so we can extract the user ID.
	var req packets.AssignScreenRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		log.Error().Err(err).Str("route", ctx.FullPath()).
			Msg("Invalid JSON: failed to bind AssignScreenRequest")
		return nil, &api.Error{Code: http.StatusBadRequest, Message: err.Error()}
	}
	// Question: What does this line do with the data from the request?
	// A: It uses the screen ID and user ID to assign ownership of the screen to a new user.
	if err := t.store.AssignScreenToUser(screenID, req.UserID); err != nil {
		log.Error().Err(err).Int("screen_id", screenID).Int("target_user_id", req.UserID).
			Msg("Database error: failed to assign screen to user")
		return nil, &api.Error{Code: http.StatusInternalServerError, Message: "could not assign screen"}
	}
	// Question: Why do we return nil here?
	// A: no additional data the client needs to receive.
	return nil, nil
}

func (t *TvController) getContentForScreen(ctx *gin.Context, user *model.User) (any, *api.Error) {
	// Question: What does this line do?
	// A: It extracts the screen ID from the URL and converts it into an integer we can use.
	screenID, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		log.Error().Err(err).Str("route", ctx.FullPath()).
			Msg("Invalid screen ID in URL: could not convert to integer during get content request")
		return nil, &api.Error{Code: http.StatusBadRequest, Message: "invalid id"}
	}
	// Question: Why are we fetching the screen before trying to get its content?
	// A: We need to confirm the screen exists before checking what’s assigned to it.
	existing, err := t.store.GetScreenByID(screenID)
	if err != nil {
		log.Error().Err(err).Int("screen_id", screenID).Str("route", ctx.FullPath()).
			Msg("Database query failed: screen not found during get content request")
		return nil, &api.Error{Code: http.StatusNotFound, Message: "screen not found"}
	}
	// Question: Why do we compare the current user's ID to the screen's creator?
	// A: Only the person who created the screen should be allowed to view its content.
	if existing.CreatedBy != user.ID {
		log.Warn().Int("screen_id", existing.ID).Int("requesting_user_id", user.ID).Int("screen_owner_id", existing.CreatedBy).Str("route", ctx.FullPath()).
			Msg("Unauthorized access attempt: user is not the creator of the screen")
		return nil, &api.Error{Code: http.StatusForbidden, Message: "forbidden"}
	}
	// Question: What are we doing here?
	// A: We're retrieving the content that’s currently assigned to this screen.
	content, err := t.store.GetContentForScreen(screenID)
	if err != nil {
		log.Info().Int("screen_id", screenID).Str("route", ctx.FullPath()).
			Msg("No content assigned to screen")
		return nil, &api.Error{Code: http.StatusNotFound, Message: "no content assigned"}
	}
	// Question: What does this return value represent?
	// A: We're sending back the content details to the user in a structured format.
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
	// Question: What does this line do?
	// A: It reads the screen ID from the URL and converts it from a string to an integer.
	screenID, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		log.Error().Err(err).Str("route", ctx.FullPath()).
			Msg("Invalid screen ID in URL: could not convert to integer during content assignment")
		return nil, &api.Error{Code: http.StatusBadRequest, Message: "invalid id"}
	}
	// Question: Why are we checking if the screen exists before assigning content?
	// A: We can’t assign content to a screen that doesn’t exist in the system.
	existingScreen, err := t.store.GetScreenByID(screenID)
	if err != nil {
		log.Error().
			Err(err).Int("screen_id", screenID).Str("route", ctx.FullPath()).
			Msg("Failed to fetch screen: screen not found during content assignment")
		return nil, &api.Error{Code: http.StatusNotFound, Message: "screen not found"}
	}
	// Question: Why do we check if the user is the creator of the screen?
	// A: Only the creator of a screen should be allowed to assign content to it.
	if existingScreen.CreatedBy != user.ID {
		log.Warn().Int("screen_id", screenID).Int("requesting_user_id", user.ID).Int("screen_owner_id", existingScreen.CreatedBy).Str("route", ctx.FullPath()).
			Msg("Unauthorized content assignment attempt: user does not own the screen")
		return nil, &api.Error{Code: http.StatusForbidden, Message: "forbidden"}
	}
	// Question: What does this binding step do?
	// A: It extracts the content ID from the JSON body of the request.
	var request packets.AssignContentToScreenRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		log.Error().Err(err).Str("route", ctx.FullPath()).
			Msg("Failed to bind JSON body during content assignment")
		return nil, &api.Error{Code: http.StatusBadRequest, Message: err.Error()}
	}
	// Question: Why do we fetch the content before assigning it to the screen?
	// A: We need to ensure the content actually exists in the system.
	existingContent, err := t.store.GetContentByID(request.ContentID)
	if err != nil {
		log.Error().Err(err).Int("content_id", request.ContentID).Str("route", ctx.FullPath()).
			Msg("Content ID not found during assignment to screen")
		return nil, &api.Error{Code: http.StatusNotFound, Message: "content not found"}
	}
	// Question: Why do we check if the user created the content?
	// A: Only the user who owns the content should be able to assign it to a screen.
	if existingContent.CreatedBy != user.ID {
		log.Warn().Int("requesting_user_id", user.ID).Int("content_owner_id", existingContent.CreatedBy).Int("content_id", existingContent.ID).Str("route", ctx.FullPath()).
			Msg("Unauthorized attempt to assign content: user does not own the content")
		return nil, &api.Error{Code: http.StatusForbidden, Message: "forbidden"}
	}
	// Question: What is this line doing?
	// A: It links the screen and the content together in the database.
	if err := t.store.AssignContentToScreen(screenID, request.ContentID); err != nil {
		log.Error().Err(err).Int("screen_id", screenID).Int("content_id", request.ContentID).Str("route", ctx.FullPath()).
			Msg("Failed to assign content to screen")
		return nil, &api.Error{Code: http.StatusInternalServerError, Message: err.Error()}
	}
	// Question: Why do we retrieve the content again after assignment?
	// A: We want the most up-to-date version of the content for the response.
	content, err := t.store.GetContentForScreen(screenID)
	log.Error().Err(err).Int("screen_id", screenID).Str("route", ctx.FullPath()).
		Msg("Failed to retrieve assigned content for screen")
	if err != nil {
		return nil, &api.Error{Code: http.StatusInternalServerError, Message: err.Error()}
	}
	// Question: Why do we fetch the screen again here?
	// A: We need to check that the screen still exists and has a device ID before sending a message.
	screen, err := t.store.GetScreenByID(screenID)
	if err != nil || screen.DeviceID == nil {
		log.Error().Err(err).Int("screen_id", screenID).Str("route", ctx.FullPath()).
			Msg("Failed to retrieve screen or missing device ID during message dispatch")
		return nil, &api.Error{Code: http.StatusInternalServerError, Message: "screen does not exist"}
	}
	// Question: Why are we converting the content response into JSON?
	// A: The device expects to receive data in JSON format.
	response, err := json.Marshal(packets.ContentResponse{
		ID:        content.ID,
		Name:      content.Name,
		Type:      content.Type,
		URL:       content.URL,
		CreatedAt: content.CreatedAt.String(),
	})

	// Question: Why are we sending the message to the screen device?
	// A: This pushes the new content update to the actual screen for display.
	if err == nil {
		err := middleware.SendMessageToScreen(*screen.DeviceID, response)
		log.Error().Err(err).Msg("Failed to send message to screen")
		if err != nil {
			log.Error().Err(err).Str("route", ctx.FullPath()).Int("screen_id", screenID).
				Msg("Failed to send content update message to screen device")
			return nil, &api.Error{Code: http.StatusInternalServerError, Message: err.Error()}
		}
	}

	// Question: Why are we returning nil here?
	// A: The operation succeeded and there is nothing further to return to the client.
	return nil, nil
}

func (t *TvController) pairScreen(ctx *gin.Context, _ *model.User) (any, *api.Error) {
	//Q: What does this line do in plain english?
	// A: We are reading the screen ID from the route to know what to delete.
	var request packets.PairScreenRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		log.Error().Err(err).Str("route", ctx.FullPath()).
			Msg("Invalid JSON in screen pairing request")
		return nil, &api.Error{Code: http.StatusBadRequest, Message: err.Error()}
	}

	// Assign the deviceID to the screen in Redis
	// Question: Why is this check important before deletion?
	// A: We want to be sure the screen exists and belongs to the user.
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
