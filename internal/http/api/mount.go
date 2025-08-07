package api

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/Nixie-Tech-LLC/medusa/internal/http/middleware"
)

// Module is a pluggable feature that attaches its endpoints to a Controller (a gin group).
type Module interface {
	Mount(c *Controller)
}

// ModuleFunc lets you define a Module with a simple function.
type ModuleFunc func(c *Controller)

func (f ModuleFunc) Mount(c *Controller) { f(c) }

// GroupConfig tells the api package how to mount a group.
type GroupConfig struct {
	Prefix     string
	Auth       bool
	SecretKey  string             // required if Auth == true
	Middleware []gin.HandlerFunc  // optional additional middleware
}

// MountGroup mounts one or more Modules under a prefix with optional auth.
func MountGroup(parent gin.IRoutes, cfg GroupConfig, modules ...Module) {
	var grp *gin.RouterGroup

	switch v := parent.(type) {
	case *gin.Engine:
		grp = v.Group(cfg.Prefix)
	case *gin.RouterGroup:
		if cfg.Prefix != "" {
			grp = v.Group(cfg.Prefix)
		} else {
			grp = v
		}
	default:
		log.Fatal().Str("type", fmt.Sprintf("%T", parent)).Msg("api.MountGroup: unsupported router type")
	}

	// Apply middleware in a deterministic order.
	for _, mw := range cfg.Middleware {
		grp.Use(mw)
	}
	if cfg.Auth {
		if cfg.SecretKey == "" {
			log.Fatal().Msg("api.MountGroup: Auth enabled but SecretKey is empty")
		}
		grp.Use(middleware.JWTMiddleware(cfg.SecretKey))
	}

	controller := &Controller{Group: grp}

	for _, m := range modules {
		m.Mount(controller)
	}
}

