package endpoints

import (
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
)

type ContentController struct {
	store db.Store
}

// What does this function do?
// Hint: It initializes a new content controller with access to a store and storage system.
// Answer: It returns a new instance of ContentController using the provided database store and file storage service.
func NewContentController(store db.Store) *ContentController {
	log.Info().Msg("launching new ContentController")

	return &ContentController{store: store}
}

// What does this function do?
// Hint: It sets up the routing for content-related API endpoints.
// Answer: It registers all the routes (GET, POST, PUT, DELETE) for content management and applies authentication.
func RegisterContentRoutes(router gin.IRoutes, store db.Store) {
	log.Info().Msg("Setting up content API routes with authentication")
	ctl := NewContentController(store)
	// require auth for all:
	log.Debug().Msg("Registering route: GET /content/:id (fetch single content)")
	router.GET("/content/:id", api.ResolveEndpointWithAuth(ctl.getContent))
	log.Debug().Msg("Registering route: GET /content (list all user content)")
	router.GET("/content", api.ResolveEndpointWithAuth(ctl.listContent))
	log.Debug().Msg("Registering route: POST /content (create new content)")
	router.POST("/content", api.ResolveEndpointWithAuth(ctl.createContent))
	log.Debug().Msg("Registering route: PUT /content/:id (update existing content)")
	router.PUT("/content/:id", api.ResolveEndpointWithAuth(ctl.updateContent))
	log.Debug().Msg("Registering route: DELETE /content/:id (delete content)")
	router.DELETE("/content/:id", api.ResolveEndpointWithAuth(ctl.deleteContent))
	log.Info().Msg("All Content routes registered successfully")
}

// What does this function do?
// Hint: It retrieves all content created by the current user.
// Answer: It fetches all content from the database, filters it by the user's ID, and returns it in a standardized response format.
func (c *ContentController) listContent(ctx *gin.Context, user *model.User) (any, *api.Error) {
	log.Debug().Msgf("User %d requested their content list", user.ID)
	all, err := c.store.ListContent()
	if err != nil {
		log.Error().Err(err).Msg("Failed to retrieve content list from the database")
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "could not list content"})
		return nil, &api.Error{Code: http.StatusInternalServerError, Message: "could not list content"}
	}

	// only return content owned by this user
	out := make([]packets.ContentResponse, 0, len(all))
	for _, x := range all {
		if x.CreatedBy != user.ID {
			continue
		}
		out = append(out, packets.ContentResponse{
			ID:        x.ID,
			Name:      x.Name,
			Type:      x.Type,
			URL:       x.URL,
			CreatedAt: x.CreatedAt.Format(time.RFC3339),
		})
	}
	log.Info().Msgf("Returned %d content item(s) for user %d", user.ID)
	//this log was suggested by ai
	return out, nil
}

// What does this function do?
// Hint: It fetches a single content item by its ID and checks if the user owns it.
// Answer: It gets the content with the specified ID from the database and returns it if the user is the creator.
func (c *ContentController) getContent(ctx *gin.Context, user *model.User) (any, *api.Error) {
	log.Debug().Msgf("User %d requested content by ID", user.ID)
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		log.Warn().
			Str("provided_id", idStr).
			Msg("Invalid content ID format; must be a valid integer")
		return nil, &api.Error{Code: http.StatusBadRequest, Message: "invalid id"}
	}
	log.Debug().
		Int("content_id", id).
		Msg("Looking up content by ID in the database")

	x, err := c.store.GetContentByID(id)
	if err != nil {
		log.Warn().
			Int("content_id", id).
			Msg("Content not found in database")
		return nil, &api.Error{Code: http.StatusNotFound, Message: "not found"}
	}

	if x.CreatedBy != user.ID {
		log.Warn().
			Int("requested_by_user_id", user.ID).
			Int("content_owner_user_id", x.CreatedBy).
			Int("content_id", x.ID).
			Msg("Access denied: user attempted to access content they do not own")
		return nil, &api.Error{Code: http.StatusForbidden, Message: "forbidden"}
	}
	// Log successful access
	log.Info().
		Int("user_id", user.ID).
		Int("content_id", x.ID).
		Msg("Successfully retrieved user-owned content")

	resp := packets.ContentResponse{
		ID:        x.ID,
		Name:      x.Name,
		Type:      x.Type,
		URL:       x.URL,
		CreatedAt: x.CreatedAt.Format(time.RFC3339),
	}

	return resp, nil
}

// What does this function do?
// Hint: It handles the upload of new content (like video or image files).
// Answer: It reads form data and uploaded file, saves the file to storage, creates a database record, and optionally assigns it to a screen.
func (c *ContentController) createContent(ctx *gin.Context, user *model.User) (any, *api.Error) {
	log.Debug().
		Int("user_id", user.ID).
		Msg("Received request to create new content")
	var req packets.CreateContentRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		log.Warn().
			Int("user_id", user.ID).
			Err(err).
			Msg("Invalid request output when creating content")
		return nil, &api.Error{Code: http.StatusForbidden, Message: err.Error()}
	}
	log.Debug().
		Str("name", req.Name).
		Str("type", req.Type).
		Str("url", req.URL).
		Int("default_duration", req.DefaultDuration).
		Msg("Creating content record in the database")

	// Save the content metadata in the database
	content, err := c.store.CreateContent(
		req.Name,
		req.Type,
		req.URL,
		req.DefaultDuration,
		user.ID,
	)

	if err != nil {
		log.Error().
			Int("user_id", user.ID).
			Str("name", req.Name).
			Msg("Failed to create content record in database")
		return nil, &api.Error{Code: http.StatusForbidden, Message: "could not create content"}
	}

	if req.ScreenID != nil {
		log.Debug().
			Int("screen_id", *req.ScreenID).
			Int("content_id", content.ID).
			Msg("Attempting to assign content to screen has failed")
		if err := c.store.AssignContentToScreen(*req.ScreenID, content.ID); err != nil {
			log.Error().
				Int("screen_id", *req.ScreenID).
				Int("content_id", content.ID).
				Msg("Failed to assign content to screen")
			return nil, &api.Error{Code: http.StatusForbidden, Message: "could not assign content"}
		}
		go func(screenID int) {
			screen, err := c.store.GetScreenByID(screenID)
			if err != nil || screen.Location == nil {
				log.Error().
					Int("screen_id", screenID).
					Err(err).
					Msg("Failed to retrieve screen by ID after content assignment")
				return
			}
			http.Get(fmt.Sprintf("%s/update", *screen.Location))
		}(*req.ScreenID)
	}
	log.Info().
		Int("content_id", content.ID).
		Int("user_id", user.ID).
		Msg("Content created successfully")

	resp := packets.ContentResponse{
		ID:        content.ID,
		Name:      content.Name,
		Type:      content.Type,
		URL:       content.URL,
		CreatedAt: content.CreatedAt.Format(time.RFC3339),
	}

	return resp, nil
}

// What does this function do?
// Hint: It updates existing content information such as name, URL, or duration.
// Answer: It verifies that the content belongs to the user, validates the incoming data, and updates the content record in the database.
// updateContent handles PUT /content/:id
func (c *ContentController) updateContent(ctx *gin.Context, user *model.User) (any, *api.Error) {
	contentID, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		log.Warn().
			Str("provided_id", ctx.Param("id")).
			Msg("Invalid content ID format in update request")
		return nil, &api.Error{Code: http.StatusForbidden, Message: "invalid content id"}
	}

	// verify ownership
	existing, err := c.store.GetContentByID(contentID)
	if err != nil {
		log.Error().
			Int("content_id", contentID).
			Err(err).
			Msg("Could not find content to update")
		return nil, &api.Error{Code: http.StatusForbidden, Message: "not found"}
	}
	if existing.CreatedBy != user.ID {
		log.Warn().
			Int("user_id", user.ID).
			Int("content_owner", existing.CreatedBy).
			Int("content_id", contentID).
			Msg("User attempted to update content they do not own this is forbidden")
		return nil, &api.Error{Code: http.StatusForbidden, Message: "forbidden"}
	}

	var req packets.UpdateContentRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		log.Warn().
			Int("user_id", user.ID).
			Int("content_id", contentID).
			Err(err).
			Msg("Could not parse the update request body — likely missing or invalid fields")
		return nil, &api.Error{Code: http.StatusForbidden, Message: err.Error()}
	}

	if err := c.store.UpdateContent(
		contentID,
		req.Name,
		req.URL,
		req.DefaultDuration,
	); err != nil {
		log.Error().
			Int("content_id", contentID).
			Err(err).
			Msg("Database update failed — could not save updated content information")
		return nil, &api.Error{Code: http.StatusForbidden, Message: err.Error()}
	}
	log.Info().
		Int("content_id", contentID).
		Int("user_id", user.ID).
		Msg("Content updated successfully")
	// no response body
	return nil, nil
}

// What does this function do?
// Hint: It removes content if the user is its creator.
// Answer: It validates the content ID, checks ownership, and deletes the content from the database.
// deleteContent handles DELETE /content/:id
func (c *ContentController) deleteContent(ctx *gin.Context, user *model.User) (any, *api.Error) {
	contentID, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		log.Warn().
			Str("provided_id", ctx.Param("id")).
			Msg("Invalid content ID format in delete request")
		return nil, &api.Error{Code: http.StatusForbidden, Message: "invalid content id"}
	}

	// verify ownership
	existing, err := c.store.GetContentByID(contentID)
	if err != nil {
		log.Warn().
			Int("content_id", contentID).
			Msg("Content not found in database for deletion")
		return nil, &api.Error{Code: http.StatusForbidden, Message: "not found"}
	}
	if existing.CreatedBy != user.ID {
		return nil, &api.Error{Code: http.StatusForbidden, Message: "forbidden"}
	}

	if err := c.store.DeleteContent(contentID); err != nil {
		log.Warn().
			Int("user_id", user.ID).
			Int("content_owner", existing.CreatedBy).
			Int("content_id", contentID).
			Msg("User attempted to delete content they do not own")
		return nil, &api.Error{Code: http.StatusForbidden, Message: err.Error()}
	}
	log.Info().
		Int("content_id", contentID).
		Int("user_id", user.ID).
		Msg("Content deleted successfully")
	return nil, nil
}
