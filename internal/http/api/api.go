package api

import (
	"net/http"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/Nixie-Tech-LLC/medusa/internal/http/middleware"
	"github.com/Nixie-Tech-LLC/medusa/internal/model"
)

// APIError represents a standardized error response.
type APIError struct {
	Code    int    `json:"-"`
	Message string `json:"error"`
}

// HandlerFunc defines a public endpoint handler signature.
type HandlerFunc func(ctx *gin.Context) (any, *APIError)

// AuthHandlerFunc defines an authenticated endpoint handler signature.
type AuthHandlerFunc func(ctx *gin.Context, user *model.User) (any, *APIError)

// sendResponse centralizes JSON responses and error logging.
func sendResponse(ctx *gin.Context, result any, apiErr *APIError) {
	if apiErr != nil {
		evt := log.Error()
		if apiErr.Code != 0 {
			evt = evt.Int("status", apiErr.Code)
		}
		evt.
			Str("path", ctx.FullPath()).
			Str("method", ctx.Request.Method).
			Msg(apiErr.Message)

		ctx.JSON(apiErr.Code, gin.H{"error": apiErr.Message})
		return
	}
	ctx.JSON(http.StatusOK, result)
}

// Public wraps a HandlerFunc for unauthenticated routes.
func Public(h HandlerFunc) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		res, err := h(ctx)
		sendResponse(ctx, res, err)
	}
}

// Private wraps an AuthHandlerFunc, enforcing authentication.
func Private(h AuthHandlerFunc) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		user, ok := middleware.GetCurrentUser(ctx)
		if !ok {
			log.Warn().Str("path", ctx.FullPath()).Msg("unauthorized access attempt")
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		res, err := h(ctx, user)
		sendResponse(ctx, res, err)
	}
}

// Controller offers helper methods for route registration.
type Controller struct {
	Group *gin.RouterGroup
}

// NewController creates a new Controller scoped to the given prefix.
func NewController(parent gin.IRoutes, prefix string) *Controller {
   var grp *gin.RouterGroup
   switch v := parent.(type) {
   case *gin.Engine:
       grp = v.Group(prefix)
   case *gin.RouterGroup:
       if prefix != "" {
           grp = v.Group(prefix)
       } else {
           grp = v
       }
   default:
       log.Fatal().Str("type", fmt.Sprintf("%T", parent)).
           Msg("api.NewController: unsupported router type")
   }
   return &Controller{Group: grp}
}

// GET registers an authenticated GET endpoint.
func (c *Controller) GET(path string, h AuthHandlerFunc) {
	c.Group.GET(path, Private(h))
}

// POST registers an authenticated POST endpoint.
func (c *Controller) POST(path string, h AuthHandlerFunc) {
	c.Group.POST(path, Private(h))
}

// PUT registers an authenticated PUT endpoint.
func (c *Controller) PUT(path string, h AuthHandlerFunc) {
	c.Group.PUT(path, Private(h))
}

// DELETE registers an authenticated DELETE endpoint.
func (c *Controller) DELETE(path string, h AuthHandlerFunc) {
	c.Group.DELETE(path, Private(h))
}

func (c *Controller) PUBLIC_GET(path string, h HandlerFunc) {
	c.Group.GET(path, Public(h))
}

// PUBPOST registers a public POST endpoint.
func (c *Controller) PUBLIC_POST(path string, h HandlerFunc) {
	c.Group.POST(path, Public(h))
}

// PUBPUT registers a public PUT endpoint.
func (c *Controller) PUBLIC_PUT(path string, h HandlerFunc) {
	c.Group.PUT(path, Public(h))
}

// PUBDELETE registers a public DELETE endpoint.
func (c *Controller) PUBLIC_DELETE(path string, h HandlerFunc) {
	c.Group.DELETE(path, Public(h))
}
