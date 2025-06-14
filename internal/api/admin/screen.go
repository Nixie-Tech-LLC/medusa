package admin

import "github.com/gin-gonic/gin"

func RegisterScreenRoutes(r gin.IRoutes) {
    r.GET("/screen", func(c *gin.Context) {
        c.JSON(200, gin.H{"message": "admin screen placeholder"})
    })
}

