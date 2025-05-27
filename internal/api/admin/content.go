package admin

import "github.com/gin-gonic/gin"

func RegisterContentRoutes(r gin.IRoutes) {
    r.GET("/content", func(c *gin.Context) {
        c.JSON(200, gin.H{"message": "admin content placeholder"})
    })
}

