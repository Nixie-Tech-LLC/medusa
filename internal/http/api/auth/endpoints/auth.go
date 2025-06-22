package endpoints

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Nixie-Tech-LLC/medusa/internal/db"
	"github.com/Nixie-Tech-LLC/medusa/internal/http/api/auth/packets"
	"github.com/Nixie-Tech-LLC/medusa/internal/http/middleware"
)

type AccountManager struct {
	jwtSecret string
	store     db.Store
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
	r.PUT("/auth/current_profile", ctl.updateCurrentProfile)
}

// POST /api/admin/auth/signup
func (a *AccountManager) userSignup(c *gin.Context) {
	var request packets.SignupRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		log.Printf("Error binding JSON: %v", err)
		return
	}

	if existing, _ := a.store.GetUserByEmail(request.Email); existing != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Email already registered, please sign up with a different email"})
		log.Printf("Email conflict: %s already registered", request.Email)
		return
	}

	hashed, err := middleware.HashPassword(request.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Something went wrong, please try again"})
		log.Printf("Error hashing password for email %s: %v", request.Email, err)
		return
	}

	userID, err := a.store.CreateUser(request.Email, hashed, request.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Something went wrong, please try again"})
		log.Printf("Could not create user for email %s: %v", request.Email, err)
		return
	}

	token, err := middleware.GenerateJWT(userID, a.jwtSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Something went wrong, please try again"})
		log.Printf("Could not generate JWT for user %d: %v", userID, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"token": token})
}

// POST /api/admin/auth/login
func (a *AccountManager) userLogin(c *gin.Context) {
	var request packets.LoginRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		log.Printf("Error binding JSON: %v", err)
		return
	}

	user, err := a.store.GetUserByEmail(request.Email)
	if err != nil || !middleware.CheckPassword(user.HashedPassword, request.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		log.Printf("Login failed for email %s: %v", request.Email, err)
		return
	}

	token, err := middleware.GenerateJWT(user.ID, a.jwtSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Something went wrong, please try again"})
		log.Printf("Could not generate JWT for user %d: %v", user.ID, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token})
}

// GET /api/admin/auth/current_profile
func (a *AccountManager) getCurrentProfile(c *gin.Context) {
	currentUser, ok := middleware.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not retrieve user from context"})
		return
	}
	c.JSON(http.StatusOK, packets.ProfileResponse{
		ID:        currentUser.ID,
		Email:     currentUser.Email,
		Name:      currentUser.Name,
		CreatedAt: currentUser.CreatedAt.Format(time.RFC3339),
		UpdatedAt: currentUser.UpdatedAt.Format(time.RFC3339),
	})
}

// PATCH /api/admin/auth/current_profile
func (a *AccountManager) updateCurrentProfile(c *gin.Context) {
	currentUser, ok := middleware.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not retrieve user from context"})
		return
	}

	var request packets.UpdateCurrentProfileRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		log.Printf("Error binding JSON: %v", err)
		return
	}

	if request.Email != currentUser.Email {
		if other, _ := a.store.GetUserByEmail(request.Email); other != nil {
			c.JSON(http.StatusConflict, gin.H{"error": "Email already registered"})
			log.Printf("Email conflict: %s already registered", request.Email)
			return
		}
	}

	if err := a.store.UpdateUserProfile(currentUser.ID, request.Email, request.Name); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Something went wrong, please try again"})
		log.Printf("Error updating profile for user %d: %v", currentUser.ID, err)
		return
	}

	updated, err := a.store.GetUserByID(currentUser.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Something went wrong, please try again"})
		log.Printf("Error fetching updated profile for user %d: %v", currentUser.ID, err)
		return
	}

	c.JSON(http.StatusOK, packets.ProfileResponse{
		ID:        updated.ID,
		Email:     updated.Email,
		Name:      updated.Name,
		CreatedAt: updated.CreatedAt.Format(time.RFC3339),
		UpdatedAt: updated.UpdatedAt.Format(time.RFC3339),
	})
}
