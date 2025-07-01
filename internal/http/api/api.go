package api


import (
	"log"
	"github.com/gin-gonic/gin"
	"github.com/Nixie-Tech-LLC/medusa/internal/model"
	"github.com/Nixie-Tech-LLC/medusa/internal/http/middleware"
	"net/http"
)

type Error struct {
	Code int 
	Message string
}


type HandlerFunc func(ctx *gin.Context, user *model.User) (any, *Error)

func ResolveEndpoint (h HandlerFunc) gin.HandlerFunc { 
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

		log.Printf("[CONTENT] LISTCONTENT NEW ENDPOINT WORKING")

		ctx.JSON(http.StatusOK, result)
	}
}
