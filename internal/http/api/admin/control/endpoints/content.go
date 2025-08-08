package endpoints

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/Nixie-Tech-LLC/medusa/internal/db"
	"github.com/Nixie-Tech-LLC/medusa/internal/http/api"
	"github.com/Nixie-Tech-LLC/medusa/internal/http/api/admin/control/packets"
	"github.com/Nixie-Tech-LLC/medusa/internal/model"
	"github.com/Nixie-Tech-LLC/medusa/internal/storage"
)

type ContentController struct {
	store   db.Store
	storage storage.Storage
}

func newContentController(store db.Store, storage storage.Storage) *ContentController {
	return &ContentController{store: store, storage: storage}
}

// ContentModule mounts all authenticated /content endpoints
func ContentModule(store db.Store, storage storage.Storage) api.Module {
	ctl := newContentController(store, storage)
	return api.ModuleFunc(func(c *api.Controller) {
		c.GET("/content/:id", 		ctl.getContent)
		c.GET("/content", 			ctl.listContent)
		c.POST("/content", 			ctl.createContent)
		c.PUT("/content/:id", 		ctl.updateContent)
		c.DELETE("/content/:id", 	ctl.deleteContent)
	})
}

func (c *ContentController) listContent(ctx *gin.Context, user *model.User) (any, *api.APIError) {
	// Get query parameters - supports multiple values
	nameFilters := ctx.QueryArray("name")
	typeFilters := ctx.QueryArray("type")

	userID := user.ID

	// Use the new SearchContentMultiple method that handles filtering in the database
	all, err := c.store.SearchContentMultiple(nameFilters, typeFilters, &userID)
	if err != nil {
		return nil, &api.APIError{Code: http.StatusInternalServerError, Message: "could not list content"}
	}

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
			Width:     x.Width,
			Height:    x.Height,
			CreatedAt: x.CreatedAt.Format(time.RFC3339),
		})
	}

	return out, nil
}

func (c *ContentController) getContent(ctx *gin.Context, user *model.User) (any, *api.APIError) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		log.Error().Str("id", ctx.Param("id")).Msg("invalid content id")
		return nil, &api.APIError{Code: http.StatusBadRequest, Message: "invalid id"}
	}

	x, err := c.store.GetContentByID(id)
	if err != nil {
		return nil, &api.APIError{Code: http.StatusNotFound, Message: "not found"}
	}

	if x.CreatedBy != user.ID {
		log.Warn().Int("owner", x.CreatedBy).Int("user", user.ID).Msg("[content] forbidden getContent")
		return nil, &api.APIError{Code: http.StatusForbidden, Message: "forbidden"}
	}

	resp := packets.ContentResponse{
		ID:        x.ID,
		Name:      x.Name,
		Type:      x.Type,
		URL:       x.URL,
		Width:     x.Width,
		Height:    x.Height,
		CreatedAt: x.CreatedAt.Format(time.RFC3339),
	}

	return resp, nil
}

func (c *ContentController) createContent(ctx *gin.Context, user *model.User) (any, *api.APIError) {
	// binary upload via multipart form
	name := ctx.PostForm("name")
	typeVal := ctx.PostForm("type")

	width, err := strconv.Atoi(ctx.PostForm("width"))
	if err != nil {
		log.Error().Err(err).Str("width", ctx.PostForm("width")).Msg("[content] non-integer width")
		return nil, &api.APIError{Code: http.StatusBadRequest, Message: "invalid form fields"}
	}
	height, err := strconv.Atoi(ctx.PostForm("height"))
	if err != nil {
		log.Error().Err(err).Str("height", ctx.PostForm("height")).Msg("[content] non-integer height")
		return nil, &api.APIError{Code: http.StatusBadRequest, Message: "invalid form fields"}
	}

	if name == "" || typeVal == "" {
		log.Warn().Msg("[content] createContent: missing required form fields")
		return nil, &api.APIError{Code: http.StatusBadRequest, Message: "missing required form fields"}
	}

	fileHeader, err := ctx.FormFile("source")
	if err != nil {
		log.Warn().Err(err).Msg("[content] createContent: missing file")
		return nil, &api.APIError{Code: http.StatusBadRequest, Message: "file is required"}
	}

	uploadPath, err := c.storage.SaveFile(fileHeader, fileHeader.Filename)
	if err != nil {
		log.Error().Err(err).Msg("[content] createContent: save failed")
		return nil, &api.APIError{Code: http.StatusInternalServerError, Message: "could not save file"}
	}

	content, err := c.store.CreateContent(
		name,
		typeVal,
		uploadPath,
		width,
		height,
		user.ID,
	)
	if err != nil {
		log.Error().Err(err).Msg("[content] createContent: db create failed")
		return nil, &api.APIError{Code: http.StatusForbidden, Message: "could not create content"}
	}

	resp := packets.ContentResponse{
		ID:        content.ID,
		Name:      content.Name,
		Type:      content.Type,
		URL:       content.URL,
		Width:     content.Width,
		Height:    content.Height,
		CreatedAt: content.CreatedAt.Format(time.RFC3339),
	}

	return resp, nil
}

func (c *ContentController) updateContent(ctx *gin.Context, user *model.User) (any, *api.APIError) {
	contentID, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		log.Error().Str("id", ctx.Param("id")).Msg("[content] updateContent: invalid id")
		return nil, &api.APIError{Code: http.StatusBadRequest, Message: "invalid content id"}
	}

	existing, err := c.store.GetContentByID(contentID)
	if err != nil {
		log.Error().Int("id", contentID).Msg("[content] updateContent: not found")
		return nil, &api.APIError{Code: http.StatusNotFound, Message: "not found"}
	}
	if existing.CreatedBy != user.ID {
		return nil, &api.APIError{Code: http.StatusForbidden, Message: "forbidden"}
	}

	var req packets.UpdateContentRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return nil, &api.APIError{Code: http.StatusBadRequest, Message: err.Error()}
	}

	if err := c.store.UpdateContent(
		contentID,
		req.Name,
		req.URL,
		req.Width,
		req.Height,
	); err != nil {
		return nil, &api.APIError{Code: http.StatusForbidden, Message: err.Error()}
	}

	return nil, nil
}

func (c *ContentController) deleteContent(ctx *gin.Context, user *model.User) (any, *api.APIError) {
	contentID, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		log.Error().Str("id", ctx.Param("id")).Msg("[content] deleteContent: invalid id")
		return nil, &api.APIError{Code: http.StatusBadRequest, Message: "invalid content id"}
	}

	existing, err := c.store.GetContentByID(contentID)
	if err != nil {
		return nil, &api.APIError{Code: http.StatusNotFound, Message: "not found"}
	}
	if existing.CreatedBy != user.ID {
		return nil, &api.APIError{Code: http.StatusForbidden, Message: "forbidden"}
	}

	if err := c.store.DeleteContent(contentID); err != nil {
		return nil, &api.APIError{Code: http.StatusForbidden, Message: err.Error()}
	}

	return nil, nil
}

