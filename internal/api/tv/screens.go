package admin

import (
    "net/http"
    "strconv"
	"fmt"
    "time"

    "github.com/gin-gonic/gin"

    "github.com/Nixie-Tech-LLC/medusa/internal/auth"
    "github.com/Nixie-Tech-LLC/medusa/internal/db"
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
    ID          int     `json:"id"`
    Name        string  `json:"name"`
    Location    *string `json:"location"`
    Paired      bool    `json:"paired"`
    PairingCode *string `json:"pairing_code,omitempty"`
    CreatedAt   string  `json:"created_at"`
    UpdatedAt   string  `json:"updated_at"`
}

func RegisterScreenRoutes(r gin.IRoutes) {
    // all admin screens routes require a valid admin JWT
    r.GET("/screens", listScreens)
    r.POST("/screens", createScreen)
    r.GET("/screens/:id", getScreen)
    r.PUT("/screens/:id", updateScreen)
    r.DELETE("/screens/:id", deleteScreen)

    // assignment
    r.POST("/screens/:id/assign", assignScreenToUser)
}

// GET /api/admin/screens
func listScreens(c *gin.Context) {
    _ , ok := auth.GetCurrentUser(c)
    if !ok {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
        return
    }
    screens, err := db.ListScreens()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "could not list screens"})
        return
    }
    out := make([]screenResponse, len(screens))
    for i, s := range screens {
        out[i] = screenResponse{
            ID:          s.ID,
            Name:        s.Name,
            Location:    s.Location,
            Paired:      s.Paired,
            PairingCode: s.PairingCode,
            CreatedAt:   s.CreatedAt.Format(time.RFC3339),
            UpdatedAt:   s.UpdatedAt.Format(time.RFC3339),
        }
    }
    c.JSON(http.StatusOK, out)
}

// POST /api/admin/screens
func createScreen(c *gin.Context) {
    _, ok := auth.GetCurrentUser(c)
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
        ID:          screen.ID,
        Name:        screen.Name,
        Location:    screen.Location,
        Paired:      screen.Paired,
        PairingCode: screen.PairingCode,
        CreatedAt:   screen.CreatedAt.Format(time.RFC3339),
        UpdatedAt:   screen.UpdatedAt.Format(time.RFC3339),
    })
}

// GET /api/admin/screens/:id
func getScreen(c *gin.Context) {
    _, ok := auth.GetCurrentUser(c)
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
        ID:          screen.ID,
        Name:        screen.Name,
        Location:    screen.Location,
        Paired:      screen.Paired,
        PairingCode: screen.PairingCode,
        CreatedAt:   screen.CreatedAt.Format(time.RFC3339),
        UpdatedAt:   screen.UpdatedAt.Format(time.RFC3339),
    })
}

// PUT /api/admin/screens/:id
func updateScreen(c *gin.Context) {
    _, ok := auth.GetCurrentUser(c)
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
        ID:          updated.ID,
        Name:        updated.Name,
        Location:    updated.Location,
        Paired:      updated.Paired,
        PairingCode: updated.PairingCode,
        CreatedAt:   updated.CreatedAt.Format(time.RFC3339),
        UpdatedAt:   updated.UpdatedAt.Format(time.RFC3339),
    })
}

// DELETE /api/admin/screens/:id
func deleteScreen(c *gin.Context) {
    _, ok := auth.GetCurrentUser(c)
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
func assignScreenToUser(c *gin.Context) {
    _, ok := auth.GetCurrentUser(c)
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


