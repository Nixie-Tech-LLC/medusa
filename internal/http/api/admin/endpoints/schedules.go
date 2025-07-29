package endpoints

import (
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// What does this function do?
// Hint: It's used to set up a simple endpoint related to admin schedules.
// Answer: It registers the /schedules route under the current router, and when accessed via GET,
// it returns a JSON response with a placeholder message.
func RegisterScheduleRoutes(r gin.IRoutes) {
	r.GET("/schedules", func(c *gin.Context) {
		log.Info().
			Str("endpoint", "/schedules").
			Msg("Received GET request to /schedules — responding with placeholder message (admin schedules route not yet implemented)")
		c.JSON(200, gin.H{"message": "admin schedules placeholder"})
	})
}
