package tv

import "github.com/gin-gonic/gin"

func RegisterScreenRoutes(r gin.IRoutes) {
    r.GET("/screens", func(c *gin.Context) {
        c.JSON(200, gin.H{"message": "tv screens placeholder"})
    })
}
