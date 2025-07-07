package api

import (
	"github.com/Nixie-Tech-LLC/medusa/internal/http/middleware"
	"github.com/Nixie-Tech-LLC/medusa/internal/model"
	"github.com/gin-gonic/gin"
	"net/http"
)

type Error struct {
	Code    int
	Message string
}

type HandlerFuncWithAuth func(ctx *gin.Context, user *model.User) (any, *Error)
type HandlerFunc func(ctx *gin.Context) (any, *Error)

func ResolveEndpointWithAuth(h HandlerFuncWithAuth) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		user, ok := middleware.GetCurrentUser(ctx)
		if !ok {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		result, err := h(ctx, user)
		if err != nil {
			ctx.JSON(err.Code, gin.H{"error": err.Message})
			return
		}

		ctx.JSON(http.StatusOK, result)
	}
}

func ResolveEndpoint(h HandlerFunc) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		result, err := h(ctx)
		if err != nil {
			ctx.JSON(err.Code, gin.H{"error": err.Message})
			return
		}

		ctx.JSON(http.StatusOK, result)
	}
}
