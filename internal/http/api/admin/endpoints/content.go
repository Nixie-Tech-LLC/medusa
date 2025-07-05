package endpoints

import (
	"fmt"
	"net/http"
	"strconv"
	"time"
	"log"

	"github.com/gin-gonic/gin"

	"github.com/Nixie-Tech-LLC/medusa/internal/db"
	"github.com/Nixie-Tech-LLC/medusa/internal/http/api/admin/packets"
	"github.com/Nixie-Tech-LLC/medusa/internal/http/api"
	"github.com/Nixie-Tech-LLC/medusa/internal/model"

)

type ContentController struct {
	store db.Store
}

func NewContentController(store db.Store) *ContentController {
	return &ContentController{store: store}
}

func RegisterContentRoutes(router gin.IRoutes, store db.Store) {
	ctl := NewContentController(store)
	// require auth for all:
	router.GET("/content/:id", 		api.ResolveEndpointWithAuth(ctl.getContent))
	router.GET("/content", 			api.ResolveEndpointWithAuth(ctl.listContent))
	router.POST("/content", 		api.ResolveEndpointWithAuth(ctl.createContent))
	router.PUT("/content/:id", 		api.ResolveEndpointWithAuth(ctl.updateContent))
	router.DELETE("/content/:id", 	api.ResolveEndpointWithAuth(ctl.deleteContent))
}

func (c *ContentController) listContent(ctx *gin.Context, user *model.User) (any, *api.Error){
	all, err := c.store.ListContent()
	if err != nil {
		log.Printf("[content] listContent DB error: %v", err)
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

	return out, nil
}

func (c *ContentController) getContent(ctx *gin.Context, user *model.User) (any, *api.Error) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
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
		CreatedAt: x.CreatedAt.Format(time.RFC3339),
	}

	return resp, nil
}

func (c *ContentController) createContent(ctx *gin.Context, user *model.User) (any, *api.Error) {
	var req packets.CreateContentRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		log.Printf("[content] CreateContent failed: %v", err)
		return nil, &api.Error{Code: http.StatusForbidden, Message: err.Error()}
	}

	content, err := c.store.CreateContent(
		req.Name,
		req.Type,
		req.URL,
		req.DefaultDuration,
		user.ID,
	)

	if err != nil {
		log.Printf("[content] CreateContent failed: %v", err)
		return nil, &api.Error{Code: http.StatusForbidden, Message: "could not create content"}
	}

	if req.ScreenID != nil {
		if err := c.store.AssignContentToScreen(*req.ScreenID, content.ID); err != nil {
			log.Printf("[content] CreateContent failed: %v", err)
			return nil, &api.Error{Code: http.StatusForbidden, Message: "could not assign content"}
		}
		go func(screenID int) {
			screen, err := c.store.GetScreenByID(screenID)
			if err != nil || screen.Location == nil {
				return
			}
			http.Get(fmt.Sprintf("%s/update", *screen.Location))
		}(*req.ScreenID)
	}

	resp := packets.ContentResponse{
		ID:        content.ID,
		Name:      content.Name,
		Type:      content.Type,
		URL:       content.URL,
		CreatedAt: content.CreatedAt.Format(time.RFC3339),
	}

	return resp, nil
}

// updateContent handles PUT /content/:id
func (c *ContentController) updateContent(ctx *gin.Context, user *model.User) (any, *api.Error) {
	contentID, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
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

	var req packets.UpdateContentRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		return nil, &api.Error{Code: http.StatusForbidden, Message: err.Error()}
	}

	if err := c.store.UpdateContent(
		contentID,
		req.Name,
		req.URL,
		req.DefaultDuration,
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

