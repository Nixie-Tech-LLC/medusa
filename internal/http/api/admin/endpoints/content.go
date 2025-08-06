package endpoints

import (
	"github.com/rs/zerolog/log"
	"net/http"
	_ "path/filepath"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Nixie-Tech-LLC/medusa/internal/db"
	"github.com/Nixie-Tech-LLC/medusa/internal/http/api"
	"github.com/Nixie-Tech-LLC/medusa/internal/http/api/admin/packets"
	"github.com/Nixie-Tech-LLC/medusa/internal/model"
	"github.com/Nixie-Tech-LLC/medusa/internal/storage"
)

type ContentController struct {
	store   db.Store
	storage storage.Storage
}

func NewContentController(store db.Store, storage storage.Storage) *ContentController {
	return &ContentController{store: store, storage: storage}
}

func RegisterContentRoutes(router gin.IRoutes, store db.Store, storage storage.Storage) {
	ctl := NewContentController(store, storage)
	// require auth for all:
	router.GET("/content/:id", api.ResolveEndpointWithAuth(ctl.getContent))
	router.GET("/content", api.ResolveEndpointWithAuth(ctl.listContent))
	router.POST("/content", api.ResolveEndpointWithAuth(ctl.createContent))
	router.PUT("/content/:id", api.ResolveEndpointWithAuth(ctl.updateContent))
	router.DELETE("/content/:id", api.ResolveEndpointWithAuth(ctl.deleteContent))
}

func (c *ContentController) listContent(ctx *gin.Context, user *model.User) (any, *api.Error) {
	// Get query parameters - supports multiple values
	nameFilters := ctx.QueryArray("name")
	typeFilters := ctx.QueryArray("type")

	userID := user.ID

	// Use the new SearchContentMultiple method that handles filtering in the database
	all, err := c.store.SearchContentMultiple(nameFilters, typeFilters, &userID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "could not list content"})
		return nil, &api.Error{Code: http.StatusInternalServerError, Message: "could not list content"}
	}

	// Convert to response format
	out := make([]packets.ContentResponse, 0, len(all))
	for _, x := range all {
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

func (c *ContentController) getContent(ctx *gin.Context, user *model.User) (any, *api.Error) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		log.Error().Msg("Failed to get content id")
		return nil, &api.Error{Code: http.StatusBadRequest, Message: "invalid id"}
	}

	x, err := c.store.GetContentByID(id)
	if err != nil {
		return nil, &api.Error{Code: http.StatusNotFound, Message: "not found"}
	}

	if x.CreatedBy != user.ID {
		log.Printf("[content] getContent: %v <==> %v", x.CreatedBy, user.ID)
		return nil, &api.Error{Code: http.StatusForbidden, Message: "forbidden"}
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

func (c *ContentController) createContent(ctx *gin.Context, user *model.User) (any, *api.Error) {
	// PostForm is used here and not ShouldBindJSON because content uploads
	// are done with binary sources (videos and images)
	// bind form fields
	name := ctx.PostForm("name")
	typeVal := ctx.PostForm("type")
	width, err := strconv.Atoi(ctx.PostForm("width"))
	if err != nil {
		log.Error().Err(err).Int("width", width).Msg("[CONTENT] Non-integer resolution width")
		return nil, &api.Error{Code: http.StatusBadRequest, Message: "invalid form fields"}
	}

	height, err := strconv.Atoi(ctx.PostForm("height"))
	if err != nil {
		log.Error().Err(err).Int("height", height).Msg("[CONTENT] Non-integer resolution height")
		return nil, &api.Error{Code: http.StatusBadRequest, Message: "invalid form fields"}
	}

	if name == "" || typeVal == "" {
		log.Printf("[content] CreateContent failed: missing required form fields")
		return nil, &api.Error{Code: http.StatusBadRequest, Message: "missing required form fields"}
	}

	// retrieve uploaded file
	fileHeader, err := ctx.FormFile("source")
	if err != nil {
		log.Printf("[content] CreateContent failed: %v", err)
		return nil, &api.Error{Code: http.StatusBadRequest, Message: "file is required"}
	}

	// save file using storage system
	uploadPath, err := c.storage.SaveFile(fileHeader, fileHeader.Filename)
	if err != nil {
		log.Printf("[content] CreateContent failed: %v", err)
		return nil, &api.Error{Code: http.StatusInternalServerError, Message: "could not save file"}
	}

	// create database record
	content, err := c.store.CreateContent(
		name,
		typeVal,
		uploadPath,
		width,
		height,
		user.ID,
	)

	if err != nil {
		log.Error().Msg("Failed to create content")
		return nil, &api.Error{Code: http.StatusForbidden, Message: "could not create content"}
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

// updateContent handles PUT /content/:id
func (c *ContentController) updateContent(ctx *gin.Context, user *model.User) (any, *api.Error) {
	contentID, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		log.Error().Msg("Failed to update content")
		return nil, &api.Error{Code: http.StatusForbidden, Message: "invalid content id"}
	}

	// verify ownership
	existing, err := c.store.GetContentByID(contentID)
	if err != nil {
		log.Error().Msg("Failed to get content by ID")
		return nil, &api.Error{Code: http.StatusForbidden, Message: "not found"}
	}
	if existing.CreatedBy != user.ID {
		return nil, &api.Error{Code: http.StatusForbidden, Message: "forbidden"}
	}

	var req packets.UpdateContentRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		return nil, &api.Error{Code: http.StatusForbidden, Message: err.Error()}
	}

	if err := c.store.UpdateContent(
		contentID,
		req.Name,
		req.URL,
		req.Width,
		req.Height,
	); err != nil {
		return nil, &api.Error{Code: http.StatusForbidden, Message: err.Error()}
	}

	// no response body
	return nil, nil
}

// deleteContent handles DELETE /content/:id
func (c *ContentController) deleteContent(ctx *gin.Context, user *model.User) (any, *api.Error) {
	contentID, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		log.Error().Msg("Failed to delete content")
		return nil, &api.Error{Code: http.StatusForbidden, Message: "invalid content id"}
	}

	// verify ownership
	existing, err := c.store.GetContentByID(contentID)
	if err != nil {
		return nil, &api.Error{Code: http.StatusForbidden, Message: "not found"}
	}
	if existing.CreatedBy != user.ID {
		return nil, &api.Error{Code: http.StatusForbidden, Message: "forbidden"}
	}

	if err := c.store.DeleteContent(contentID); err != nil {
		return nil, &api.Error{Code: http.StatusForbidden, Message: err.Error()}
	}

	return nil, nil
}
