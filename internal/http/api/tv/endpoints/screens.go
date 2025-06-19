package endpoints

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Nixie-Tech-LLC/medusa/internal/db"
	"github.com/Nixie-Tech-LLC/medusa/internal/http/api/tv/packets"
	"github.com/Nixie-Tech-LLC/medusa/internal/http/middleware"
	redisclient "github.com/Nixie-Tech-LLC/medusa/internal/redis"
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
	r.GET("/screens", ctl.listScreens)
	r.POST("/screens", ctl.createScreen)
	r.GET("/screens/:id", ctl.getScreen)
	r.PUT("/screens/:id", ctl.updateScreen)
	r.DELETE("/screens/:id", ctl.deleteScreen)

	// screen <-> content
	r.GET("/screens/:id/content", ctl.getContentForScreen)
	r.POST("/screens/:id/content", ctl.assignContentToScreen)

	// pairing
	r.POST("/screens/pair", ctl.pairScreen)

	// assignment
	r.POST("/screens/:id/assign", ctl.assignScreenToUser)
}

// GET /api/admin/screens
func (t *TvController) listScreens(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	all, err := t.store.ListScreens()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    var out []packets.ScreenResponse
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

    c.JSON(http.StatusOK, out)
}

// POST /api/admin/screens
func (t *TvController) createScreen(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req packets.CreateScreenRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    // pass user.ID as Creator
    screen, err := t.store.CreateScreen(req.Name, req.Location, user.ID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create screen"})
        return
    }

    c.JSON(http.StatusCreated, packets.ScreenResponse{
        ID:        screen.ID,
        DeviceID:  screen.DeviceID,
        Name:      screen.Name,
        Location:  screen.Location,
        Paired:    screen.Paired,
        CreatedAt: screen.CreatedAt.Format(time.RFC3339),
        UpdatedAt: screen.UpdatedAt.Format(time.RFC3339),
    })
}

// GET /api/admin/screens/:id
func (t *TvController) getScreen(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	id, err := strconv.Atoi(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
        return
    }

    s, err := t.store.GetScreenByID(id)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "screen not found"})
        return
    }
    if s.CreatedBy != user.ID {
        c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
        return
    }

    c.JSON(http.StatusOK, packets.ScreenResponse{
        ID:        s.ID,
        DeviceID:  s.DeviceID,
        Name:      s.Name,
        Location:  s.Location,
        Paired:    s.Paired,
        CreatedAt: s.CreatedAt.Format(time.RFC3339),
        UpdatedAt: s.UpdatedAt.Format(time.RFC3339),
    })
}

// PUT /api/admin/screens/:id
func (t *TvController) updateScreen(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	id, err := strconv.Atoi(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
        return
    }

    // ownership check
    existing, err := t.store.GetScreenByID(id)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "screen not found"})
        return
    }
    if existing.CreatedBy != user.ID {
        c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
        return
    }

    var req packets.UpdateScreenRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    if err := t.store.UpdateScreen(id, req.Name, req.Location); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "could not update screen"})
        return
    }

    updated, _ := t.store.GetScreenByID(id)
    c.JSON(http.StatusOK, packets.ScreenResponse{
        ID:        updated.ID,
        DeviceID:  updated.DeviceID,
        Name:      updated.Name,
        Location:  updated.Location,
        Paired:    updated.Paired,
        CreatedAt: updated.CreatedAt.Format(time.RFC3339),
        UpdatedAt: updated.UpdatedAt.Format(time.RFC3339),
    })
}

// DELETE /api/admin/screens/:id
func (t *TvController) deleteScreen(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
    id, err := strconv.Atoi(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
        return
    }
    existing, err := t.store.GetScreenByID(id)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "screen not found"})
        return
    }
    if existing.CreatedBy != user.ID {
        c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
        return
    }

    if err := t.store.DeleteScreen(id); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "could not delete screen"})
        return
    }
    c.Status(http.StatusNoContent)
}

// POST /api/admin/screens/:id/assign
func (t *TvController) assignScreenToUser(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	screenID, err := strconv.Atoi(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
        return
    }
    existing, err := t.store.GetScreenByID(screenID)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "screen not found"})
        return
    }
    if existing.CreatedBy != user.ID {
        c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
        return
    }

    var req packets.AssignScreenRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    if err := t.store.AssignScreenToUser(screenID, req.UserID); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "could not assign screen"})
        return
    }
    c.Status(http.StatusOK)
}

func (t *TvController) getContentForScreen(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	screenID, _ := strconv.Atoi(c.Param("id"))

    existing, err := t.store.GetScreenByID(screenID)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "screen not found"})
        return
    }
    if existing.CreatedBy != user.ID {
        c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
        return
    }

	content, err := t.store.GetContentForScreen(screenID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no content assigned"})
		return
	}
	c.JSON(http.StatusOK, packets.ContentResponse{
		ID:        content.ID,
		Name:      content.Name,
		Type:      content.Type,
		URL:       content.URL,
		CreatedAt: content.CreatedAt.Format(time.RFC3339),
	})
}

func (t *TvController) assignContentToScreen(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	screenID, _ := strconv.Atoi(c.Param("id"))

    existingScreen, err := t.store.GetScreenByID(screenID)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "screen not found"})
        return
    }
    if existingScreen.CreatedBy != user.ID {
        c.JSON(http.StatusForbidden, gin.H{"error": "user does not have access to screen"})
        return
    }


	var request packets.AssignContentToScreenRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	existingContent, err := t.store.GetContentByID(request.ContentID)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "content not found"})
        return
    }
    if existingContent.CreatedBy != user.ID {
        c.JSON(http.StatusForbidden, gin.H{"error": "user does not have access to content"})
        return
    }


	if err := db.AssignContentToScreen(screenID, request.ContentID); err != nil {
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

	var request packets.PairScreenRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	key := "pairing:" + request.PairingCode

	deviceID, err := redisclient.Rdb.Get(c, key).Result()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could find deviceID for pairing code"})
		return
	}

	redisclient.Rdb.Del(c, key)

	if err := db.AssignDeviceIDToScreen(request.ScreenID, &deviceID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not update screen device ID"})
		return
	}

	if err := db.PairScreen(request.ScreenID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not update screen"})
		return
	}

	c.JSON(200, gin.H{"success": "screen paired successfully"})
}
