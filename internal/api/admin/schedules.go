package admin

import "github.com/gin-gonic/gin"

func RegisterScheduleRoutes(r gin.IRoutes) {
    r.GET("/schedules", func(c *gin.Context) {
        c.JSON(200, gin.H{"message": "admin schedules placeholder"})
    })
}

