package admin

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Nixie-Tech-LLC/medusa/internal/db"
	"github.com/Nixie-Tech-LLC/medusa/internal/http/middleware"
	redisclient "github.com/Nixie-Tech-LLC/medusa/internal/redis"
)

type createScreenRequest struct {
	Name     string  `json:"name" binding:"required"`
	Location *string `json:"location"`
}

type updateScreenRequest struct {
	Name     *string `json:"name"`
	Location *string `json:"location"`
}

type assignScreenRequest struct {
	UserID int `json:"user_id" binding:"required"`
}

// screenResponse mirrors model.Screen but flattens times to RFC3339
type screenResponse struct {
	ID        int     `json:"id"`
	DeviceID  *string `json:"device_id"`
	Name      string  `json:"name"`
	Location  *string `json:"location"`
	Paired    bool    `json:"paired"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

type contentResponse struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	URL       string `json:"url"`
	CreatedAt string `json:"created_at"`
}

type TvController struct {
	store db.Store
}

func NewTvController(store db.Store) *TvController {
	return &TvController{store: store}
}

func RegisterScreenRoutes(r gin.IRoutes, store db.Store) {
	ctl := NewTvController(store)
	// all admin screens routes require a valid admin JWT
	r.GET("/screens", ctl.listScreens)
	r.POST("/screens", ctl.createScreen)
	r.GET("/screens/:id", ctl.getScreen)
	r.PUT("/screens/:id", ctl.updateScreen)
	r.DELETE("/screens/:id", ctl.deleteScreen)
	r.GET("/screens/:id/content", ctl.getContentForScreen)
	r.POST("/screens/:id/content", ctl.assignContentToScreen)

	// pairing
	r.POST("/screens/pair", ctl.pairScreen)

	// assignment
	r.POST("/screens/:id/assign", ctl.assignScreenToUser)
}

// GET /api/admin/screens
func (t *TvController) listScreens(c *gin.Context) {
	_, ok := middleware.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	screens, err := db.ListScreens()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	out := make([]screenResponse, len(screens))
	for i, s := range screens {
		out[i] = screenResponse{
			ID:        s.ID,
			DeviceID:  s.DeviceID,
			Name:      s.Name,
			Location:  s.Location,
			Paired:    s.Paired,
			CreatedAt: s.CreatedAt.Format(time.RFC3339),
			UpdatedAt: s.UpdatedAt.Format(time.RFC3339),
		}
	}
	c.JSON(http.StatusOK, out)
}

// POST /api/admin/screens
func (t *TvController) createScreen(c *gin.Context) {
	_, ok := middleware.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req createScreenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	screen, err := db.CreateScreen(req.Name, req.Location)
	if err != nil {
		fmt.Println("CreateScreen error:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create screen"})
		return
	}

	c.JSON(http.StatusCreated, screenResponse{
		ID:        screen.ID,
		Name:      screen.Name,
		Location:  screen.Location,
		Paired:    screen.Paired,
		CreatedAt: screen.CreatedAt.Format(time.RFC3339),
		UpdatedAt: screen.UpdatedAt.Format(time.RFC3339),
	})
}

// GET /api/admin/screens/:id
func (t *TvController) getScreen(c *gin.Context) {
	_, ok := middleware.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	id, _ := strconv.Atoi(c.Param("id"))
	screen, err := db.GetScreenByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "screen not found"})
		return
	}
	c.JSON(http.StatusOK, screenResponse{
		ID:        screen.ID,
		Name:      screen.Name,
		Location:  screen.Location,
		Paired:    screen.Paired,
		CreatedAt: screen.CreatedAt.Format(time.RFC3339),
		UpdatedAt: screen.UpdatedAt.Format(time.RFC3339),
	})
}

// PUT /api/admin/screens/:id
func (t *TvController) updateScreen(c *gin.Context) {
	_, ok := middleware.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	id, _ := strconv.Atoi(c.Param("id"))
	var req updateScreenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := db.UpdateScreen(id, req.Name, req.Location); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not update screen"})
		return
	}
	updated, _ := db.GetScreenByID(id)
	c.JSON(http.StatusOK, screenResponse{
		ID:        updated.ID,
		Name:      updated.Name,
		Location:  updated.Location,
		Paired:    updated.Paired,
		CreatedAt: updated.CreatedAt.Format(time.RFC3339),
		UpdatedAt: updated.UpdatedAt.Format(time.RFC3339),
	})
}

// DELETE /api/admin/screens/:id
func (t *TvController) deleteScreen(c *gin.Context) {
	_, ok := middleware.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	id, _ := strconv.Atoi(c.Param("id"))
	if err := db.DeleteScreen(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not delete screen"})
		return
	}
	c.Status(http.StatusNoContent)
}

// POST /api/admin/screens/:id/assign
func (t *TvController) assignScreenToUser(c *gin.Context) {
	_, ok := middleware.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	id, _ := strconv.Atoi(c.Param("id"))
	var req assignScreenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := db.AssignScreenToUser(id, req.UserID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not assign screen"})
		return
	}
	c.Status(http.StatusOK)
}

func (t *TvController) getContentForScreen(ctx *gin.Context) {
	screenID, _ := strconv.Atoi(ctx.Param("id"))
	c, err := t.store.GetContentForScreen(screenID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "no content assigned"})
		return
	}
	ctx.JSON(http.StatusOK, contentResponse{
		ID:        c.ID,
		Name:      c.Name,
		Type:      c.Type,
		URL:       c.URL,
		CreatedAt: c.CreatedAt.Format(time.RFC3339),
	})
}

func (t *TvController) assignContentToScreen(c *gin.Context) {
	if _, ok := middleware.GetCurrentUser(c); !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	screenID, _ := strconv.Atoi(c.Param("id"))

	var req struct {
		ContentID int `json:"content_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := db.AssignContentToScreen(screenID, req.ContentID); err != nil {
		fmt.Printf("AssignContentToScreen error: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// fire-and-forget the TV signal, same as in createContent:
	go func() {
		screen, err := db.GetScreenByID(screenID)
		if err != nil || screen.Location == nil {
			return
		}
		signalURL := fmt.Sprintf("%s/update", *screen.Location)
		http.Get(signalURL)
	}()

	c.Status(http.StatusOK)
}

func (t *TvController) pairScreen(c *gin.Context) {
	if _, ok := middleware.GetCurrentUser(c); !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req struct {
		PairingCode string `json:"code" binding:"required"`
		UserID      int    `json:"user_id" binding:"required"`
		ScreenID    int    `json:"screen_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	key := "pairing:" + req.PairingCode

	deviceID, err := redisclient.Rdb.Get(c, key).Result()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could find deviceID for pairing code"})
		return
	}

	redisclient.Rdb.Del(c, key)

	if err := db.AssignDeviceIDToScreen(req.ScreenID, &deviceID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not update screen device ID"})
		return
	}

	if err := db.PairScreen(req.ScreenID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not update screen"})
		return
	}
	c.JSON(200, gin.H{"success": "screen paired successfully"})
}
