package main

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"html/template"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/Nixie-Tech-LLC/medusa/internal/db"
	"github.com/Nixie-Tech-LLC/medusa/internal/storage"
	"github.com/Nixie-Tech-LLC/medusa/internal/http/api"
	adminapi 	"github.com/Nixie-Tech-LLC/medusa/internal/http/api/admin/control/endpoints"
	authapi  	"github.com/Nixie-Tech-LLC/medusa/internal/http/api/admin/auth/endpoints"
	clientapi 	"github.com/Nixie-Tech-LLC/medusa/internal/http/api/tv/endpoints"
)


// RegisterRoutes sets up all application routes
func RegisterRoutes(r *gin.Engine, env Environment, store db.Store, storageSystem storage.Storage, tmpl *template.Template) {
	r.SetHTMLTemplate(tmpl)
	// CORS
	r.Use(cors.New(cors.Config{
		AllowOriginFunc: func(origin string) bool { return true },
		AllowMethods: []string{
			"GET", 
			"POST", 
			"PUT", 
			"PATCH", 
			"DELETE", 
			"OPTIONS", 
			"HEAD",
		},
		AllowHeaders: []string{
			"Origin", 
			"Content-Type", 
			"Authorization", 
			"Accept", 
			"If-None-Match", 
			"X-If-None-Match",
		},
		ExposeHeaders:[]string{
			"Content-Length",
			"ETag",
			"X-Content-ETag",
		},
		AllowCredentials: false,
	}))

	api.MountGroup(r, api.GroupConfig{
		Prefix: "/api/admin",
		Auth:   false,
	}, 
		authapi.AuthPublicModule(env.SecretKey, store),
	)

	api.MountGroup(r, api.GroupConfig{
		Prefix:    "/api/admin",
		Auth:      true,
		SecretKey: env.SecretKey,
	}, 
		// control modules
		adminapi.ContentModule(store, storageSystem),
		adminapi.ScreenModule(store),
		adminapi.PlaylistModule(store),
		// session endpoints that require auth
		authapi.AuthSessionModule(env.SecretKey, store),
		adminapi.ScheduleModule(store),
	)

	api.MountGroup(r, api.GroupConfig{
		Prefix: "/api/tv",
	}, 
		clientapi.PairingModule(store),
		clientapi.IntegrationsModule(),
	)

	// Static content
	if !env.UseSpaces {
		r.Static("/uploads", "./uploads")
		r.GET("/integrations/*filepath", func(c *gin.Context) {
			rel := strings.TrimPrefix(c.Param("filepath"), "/")
			full := filepath.Join("integrations", rel)

			info, err := os.Stat(full)
			if err != nil {
				c.Status(http.StatusNotFound)
				return
			}

			if info.IsDir() {
				full = filepath.Join(full, "index.html")
				if _, err := os.Stat(full); err != nil {
					c.Status(http.StatusNotFound)
					return
				}
			}

			c.File(full)
		})
	}
}
