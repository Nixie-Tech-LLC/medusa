package endpoints

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/Nixie-Tech-LLC/medusa/internal/db"
	"github.com/Nixie-Tech-LLC/medusa/internal/http/api/admin/packets"
	"github.com/Nixie-Tech-LLC/medusa/internal/http/middleware"
	"github.com/gin-gonic/gin"
)

type ContentController struct {
	store db.Store
}

func NewContentController(store db.Store) *ContentController {
	return &ContentController{store: store}
}

func RegisterContentRoutes(r gin.IRoutes, store db.Store) {
	ctl := NewContentController(store)

	// require admin JWT
	r.GET("/content", ctl.listContent)
	r.GET("/content/:id", ctl.getContent)
	r.POST("/content", ctl.createContent)
}

func (c *ContentController) listContent(ctx *gin.Context) {
	if _, ok := middleware.GetCurrentUser(ctx); !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	all, err := c.store.ListContent()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "could not list content"})
		return
	}
	out := make([]packets.ContentResponse, len(all))
	for i, x := range all {
		out[i] = packets.ContentResponse{
			ID:        x.ID,
			Name:      x.Name,
			Type:      x.Type,
			URL:       x.URL,
			CreatedAt: x.CreatedAt.Format(time.RFC3339),
		}
	}
	ctx.JSON(http.StatusOK, out)
}

func (c *ContentController) getContent(ctx *gin.Context) {
	if _, ok := middleware.GetCurrentUser(ctx); !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	id, _ := strconv.Atoi(ctx.Param("id"))
	x, err := c.store.GetContentByID(id)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "not found"})
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
	if _, ok := middleware.GetCurrentUser(ctx); !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var request packets.CreateContentRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	content, err := c.store.CreateContent(request.Name, request.Type, request.URL)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "could not create content"})
		return
	}

	// if ScreenID provided, assign & signal immediately:
	if request.ScreenID != nil {
		if err := c.store.AssignContentToScreen(*request.ScreenID, content.ID); err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "could not assign content to screen"})
			return
		}
		// fire-and-forget signalling to the TV app:
		go func(screenID int) {
			// first lookup the screen to get its Location
			screen, err := c.store.GetScreenByID(screenID)
			if err != nil || screen.Location == nil {
				return
			}
			// assume the TV app listens on HTTP at this location + "/update"
			signalURL := fmt.Sprintf("%s/update", *screen.Location)
			http.Get(signalURL) // ignore errors
		}(*request.ScreenID)
	}

	ctx.JSON(http.StatusCreated, packets.ContentResponse{
		ID:        content.ID,
		Name:      content.Name,
		Type:      content.Type,
		URL:       content.URL,
		CreatedAt: content.CreatedAt.Format(time.RFC3339),
	})
}
