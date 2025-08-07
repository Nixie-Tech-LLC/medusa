package endpoints

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/Nixie-Tech-LLC/medusa/internal/db"
	"github.com/Nixie-Tech-LLC/medusa/internal/http/api"
	"github.com/Nixie-Tech-LLC/medusa/internal/http/api/admin/auth/packets"
	"github.com/Nixie-Tech-LLC/medusa/internal/http/middleware"
	"github.com/Nixie-Tech-LLC/medusa/internal/model"
)

// AuthPublicModule mounts public auth endpoints (/auth/signup, /auth/login)
func AuthPublicModule(jwtSecret string, store db.Store) api.Module {
	ctl := newAccountManager(jwtSecret, store)
	return api.ModuleFunc(func(c *api.Controller) {
		c.PUBLIC_POST("/auth/signup", 	ctl.userSignup)
		c.PUBLIC_POST("/auth/login", 	ctl.userLogin)
	})
}

// AuthSessionModule mounts private session/profile endpoints (JWT required)
func AuthSessionModule(jwtSecret string, store db.Store) api.Module {
	ctl := newAccountManager(jwtSecret, store)
	return api.ModuleFunc(func(c *api.Controller) {
		c.GET("/auth/current_profile", ctl.getCurrentProfile)
		c.PUT("/auth/current_profile", ctl.updateCurrentProfile)
	})
}

type AccountManager struct {
	jwtSecret string
	store     db.Store
}

func newAccountManager(secret string, store db.Store) *AccountManager {
	return &AccountManager{jwtSecret: secret, store: store}
}

// POST /api/admin/auth/signup
func (a *AccountManager) userSignup(ctx *gin.Context) (any, *api.APIError) {
	var request packets.SignupRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		return nil, &api.APIError{Code: http.StatusBadRequest, Message: err.Error()}
	}

	if existing, _ := a.store.GetUserByEmail(request.Email); existing != nil {
		log.Warn().Str("email", request.Email).Msg("signup email already registered")
		return nil, &api.APIError{Code: http.StatusConflict, Message: "email already registered"}
	}

	hashed, err := middleware.HashPassword(request.Password)
	if err != nil {
		return nil, &api.APIError{Code: http.StatusInternalServerError, Message: "could not hash password"}
	}

	userID, err := a.store.CreateUser(request.Email, hashed, request.Name)
	if err != nil {
		return nil, &api.APIError{Code: http.StatusInternalServerError, Message: "could not create user"}
	}

	token, err := middleware.GenerateJWT(userID, a.jwtSecret)
	if err != nil {
		return nil, &api.APIError{Code: http.StatusInternalServerError, Message: "could not generate token"}
	}

	return gin.H{"token": token}, nil
}

// POST /api/admin/auth/login
func (a *AccountManager) userLogin(ctx *gin.Context) (any, *api.APIError) {
	var request packets.LoginRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		return nil, &api.APIError{Code: http.StatusBadRequest, Message: err.Error()}
	}

	foundUser, err := a.store.GetUserByEmail(request.Email)
	if err != nil || foundUser == nil || !middleware.CheckPassword(foundUser.HashedPassword, request.Password) {
		return nil, &api.APIError{Code: http.StatusUnauthorized, Message: "invalid credentials"}
	}

	token, err := middleware.GenerateJWT(foundUser.ID, a.jwtSecret)
	if err != nil {
		return nil, &api.APIError{Code: http.StatusInternalServerError, Message: "could not generate token"}
	}

	return gin.H{"token": token}, nil
}

// GET /api/admin/auth/current_profile
func (a *AccountManager) getCurrentProfile(ctx *gin.Context, user *model.User) (any, *api.APIError) {
	return packets.ProfileResponse{
		ID:        user.ID,
		Email:     user.Email,
		Name:      user.Name,
		CreatedAt: user.CreatedAt.Format(time.RFC3339),
		UpdatedAt: user.UpdatedAt.Format(time.RFC3339),
	}, nil
}

// PUT /api/admin/auth/current_profile
func (a *AccountManager) updateCurrentProfile(ctx *gin.Context, user *model.User) (any, *api.APIError) {
	var request packets.UpdateCurrentProfileRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		return nil, &api.APIError{Code: http.StatusBadRequest, Message: err.Error()}
	}

	if request.Email != user.Email {
		if other, _ := a.store.GetUserByEmail(request.Email); other != nil {
			return nil, &api.APIError{Code: http.StatusConflict, Message: "email already in use"}
		}
	}

	if err := a.store.UpdateUserProfile(user.ID, request.Email, request.Name); err != nil {
		return nil, &api.APIError{Code: http.StatusInternalServerError, Message: "could not update profile"}
	}

	updated, err := a.store.GetUserByID(user.ID)
	if err != nil {
		return nil, &api.APIError{Code: http.StatusInternalServerError, Message: "could not fetch updated profile"}
	}

	return packets.ProfileResponse{
		ID:        updated.ID,
		Email:     updated.Email,
		Name:      updated.Name,
		CreatedAt: updated.CreatedAt.Format(time.RFC3339),
		UpdatedAt: updated.UpdatedAt.Format(time.RFC3339),
	}, nil
}

