package api


import (
	"github.com/gin-gonic/gin"
	"github.com/Nixie-Tech-LLC/medusa/internal/model"
	"github.com/Nixie-Tech-LLC/medusa/internal/http/middleware"
	"net/http"
)

type Error struct {
	Code int 
	Message string
}


type HandlerFuncWithAuth func(ctx *gin.Context, user *model.User) (any, *Error)
type HandlerFunc func(ctx *gin.Context) (any, *Error)

func ResolveEndpointWithAuth (h HandlerFuncWithAuth) gin.HandlerFunc { 
	return func(ctx *gin.Context) {
		user, ok := middleware.GetCurrentUser(ctx)
		if !ok {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		result, error := h(ctx, user)
		if error != nil {
			ctx.JSON(error.Code, gin.H{"error": error.Message})
			return
		}

		ctx.JSON(http.StatusOK, result)
	}
}

func ResolveEndpoint (h HandlerFunc) gin.HandlerFunc { 
	return func(ctx *gin.Context) {
		result, error := h(ctx)
		if error != nil {
			ctx.JSON(error.Code, gin.H{"error": error.Message})
			return
		}

		ctx.JSON(http.StatusOK, result)
	}
}

