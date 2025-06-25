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
	"github.com/Nixie-Tech-LLC/medusa/internal/http/middleware"
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
	router.GET("/content", ctl.listContent)
	router.POST("/content", ctl.createContent)
	router.GET("/content/:id", ctl.getContent)
	router.PUT("/content/:id", ctl.updateContent)
	router.DELETE("/content/:id", ctl.deleteContent)
}

func (c *ContentController) listContent(ctx *gin.Context) {
	user, ok := middleware.GetCurrentUser(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	all, err := c.store.ListContent()
	if err != nil {
    	log.Printf("[content] listContent DB error: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "could not list content"})
		return
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

	ctx.JSON(http.StatusOK, out)
}

func (c *ContentController) getContent(ctx *gin.Context) {
	user, ok := middleware.GetCurrentUser(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	x, err := c.store.GetContentByID(id)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	if x.CreatedBy != user.ID {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	ctx.JSON(http.StatusOK, packets.ContentResponse{
		ID:        x.ID,
		Name:      x.Name,
		Type:      x.Type,
		URL:       x.URL,
		CreatedAt: x.CreatedAt.Format(time.RFC3339),
	})
}

func (c *ContentController) createContent(ctx *gin.Context) {
	user, ok := middleware.GetCurrentUser(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req packets.CreateContentRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		log.Printf("[content] CreateContent failed: %v", err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
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

		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "could not create content"})
		return
	}

	if req.ScreenID != nil {
		if err := c.store.AssignContentToScreen(*req.ScreenID, content.ID); err != nil {
			log.Printf("[content] CreateContent failed: %v", err)

			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "could not assign content"})
			return
		}
		go func(screenID int) {
			screen, err := c.store.GetScreenByID(screenID)
			if err != nil || screen.Location == nil {
				return
			}
			http.Get(fmt.Sprintf("%s/update", *screen.Location))
		}(*req.ScreenID)
	}

	ctx.JSON(http.StatusCreated, packets.ContentResponse{
		ID:        content.ID,
		Name:      content.Name,
		Type:      content.Type,
		URL:       content.URL,
		CreatedAt: content.CreatedAt.Format(time.RFC3339),
	})
}

// updateContent handles PUT /content/:id
func (c *ContentController) updateContent(ctx *gin.Context) {
	user, ok := middleware.GetCurrentUser(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	idParam := ctx.Param("id")
	contentID, err := strconv.Atoi(idParam)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid content id"})
		return
	}

	// verify ownership
	existing, err := c.store.GetContentByID(contentID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if existing.CreatedBy != user.ID {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	var req packets.UpdateContentRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := c.store.UpdateContent(
		contentID,
		req.Name,
		req.URL,
		req.DefaultDuration,
		); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.Status(http.StatusNoContent)
}

// deleteContent handles DELETE /content/:id
func (c *ContentController) deleteContent(ctx *gin.Context) {
	user, ok := middleware.GetCurrentUser(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	idParam := ctx.Param("id")
	contentID, err := strconv.Atoi(idParam)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid content id"})
		return
	}

	// verify ownership
	existing, err := c.store.GetContentByID(contentID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if existing.CreatedBy != user.ID {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	if err := c.store.DeleteContent(contentID); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.Status(http.StatusNoContent)
}

