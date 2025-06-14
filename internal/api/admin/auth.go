package admin

import (
    "net/http"
    "time"

    "github.com/gin-gonic/gin"

    "github.com/Nixie-Tech-LLC/medusa/internal/auth"
    "github.com/Nixie-Tech-LLC/medusa/internal/db"
)

// body for registering
type signupRequest struct {
    Email    string  `json:"email" binding:"required,email"`
    Password string  `json:"password" binding:"required,min=8"`
    Name     *string `json:"name"`
}

// body for logging in
type loginRequest struct {
    Email    string `json:"email" binding:"required,email"`
    Password string `json:"password" binding:"required"`
}

// returned for profile endpoints
type profileResponse struct {
    ID        int     `json:"id"`
    Email     string  `json:"email"`
    Name      *string `json:"name"`
    CreatedAt string  `json:"created_at"`
    UpdatedAt string  `json:"updated_at"`
}

type AccountManager struct {
	jwtSecret string 
	store 	db.Store
}

func accountManagementController(secret string, store db.Store) *AccountManager {
	return &AccountManager{jwtSecret: secret, store: store}
}

// mounts auth‐related routes under /api/admin/auth
func RegisterAuthRoutes(r gin.IRoutes, jwtSecret string, store db.Store) {
	ctl := accountManagementController(jwtSecret, store)

    r.POST("/auth/signup", ctl.userSignup)
	r.POST("/auth/login", ctl.userLogin)
    r.GET("/auth/current_profile", ctl.getCurrentProfile)
	r.PATCH("/auth/current_profile", ctl.updateCurrentProfile)
}

// POST /api/admin/auth/signup
func (a *AccountManager) userSignup(c *gin.Context) {
    var req signupRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    if existing, _ := a.store.GetUserByEmail(req.Email); existing != nil {
        c.JSON(http.StatusConflict, gin.H{"error": "email already registered"})
        return
    }

    hashed, err := auth.HashPassword(req.Password)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "could not hash password"})
        return
    }

    userID, err := a.store.CreateUser(req.Email, hashed, req.Name)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create user"})
        return
    }

    token, err := auth.GenerateJWT(userID, a.jwtSecret)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "could not generate token"})
        return
    }

    c.JSON(http.StatusCreated, gin.H{"token": token})
}

// POST /api/admin/auth/login
func (a *AccountManager) userLogin(c *gin.Context) {
    var req loginRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    user, err := a.store.GetUserByEmail(req.Email)
    if err != nil || !auth.CheckPassword(user.HashedPassword, req.Password) {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
        return
    }

    token, err := auth.GenerateJWT(user.ID, a.jwtSecret)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "could not generate token"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"token": token})
}

// GET /api/admin/auth/current_profile
func (a *AccountManager) getCurrentProfile(c *gin.Context) {
    currentUser, ok := auth.GetCurrentUser(c)
    if !ok {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "could not retrieve user from context"})
        return
    }
    c.JSON(http.StatusOK, profileResponse{
        ID:        currentUser.ID,
        Email:     currentUser.Email,
        Name:      currentUser.Name,
        CreatedAt: currentUser.CreatedAt.Format(time.RFC3339),
        UpdatedAt: currentUser.UpdatedAt.Format(time.RFC3339),
    })
}

// PATCH /api/admin/auth/current_profile
func (a *AccountManager) updateCurrentProfile(c *gin.Context) {
    currentUser, ok := auth.GetCurrentUser(c)
    if !ok {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "could not retrieve user from context"})
        return
    }

    var req struct {
        Email string  `json:"email" binding:"required,email"`
        Name  *string `json:"name"`
    }
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    if req.Email != currentUser.Email {
        if other, _ := a.store.GetUserByEmail(req.Email); other != nil {
            c.JSON(http.StatusConflict, gin.H{"error": "email already in use"})
            return
        }
    }

    if err := a.store.UpdateUserProfile(currentUser.ID, req.Email, req.Name); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "could not update profile"})
        return
    }

    updated, err := a.store.GetUserByID(currentUser.ID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "could not fetch updated profile"})
        return
    }

    c.JSON(http.StatusOK, profileResponse{
        ID:        updated.ID,
        Email:     updated.Email,
        Name:      updated.Name,
        CreatedAt: updated.CreatedAt.Format(time.RFC3339),
        UpdatedAt: updated.UpdatedAt.Format(time.RFC3339),
    })
}

