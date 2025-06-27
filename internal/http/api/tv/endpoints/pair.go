package endpoints

import (
	"math/rand"
	"time"

	"github.com/Nixie-Tech-LLC/medusa/internal/db"
	"github.com/Nixie-Tech-LLC/medusa/internal/http/api/tv/packets"
	"github.com/Nixie-Tech-LLC/medusa/internal/http/middleware"
	redisclient "github.com/Nixie-Tech-LLC/medusa/internal/redis"
	"github.com/gin-gonic/gin"
)

func RegisterPairingRoutes(r gin.IRoutes) {
	r.POST("/pair", requestPairing)
	r.POST("/socket", middleware.TVWebSocket())
}

func requestPairing(c *gin.Context) {
	var request packets.TVRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	db.IsScreenPairedByDeviceID(&request.DeviceID)

	code := generatePairCode()
	key := code

	redisclient.Set(c, key, request.DeviceID, 5*time.Minute)

	c.JSON(200, gin.H{"code": code})
}

func generatePairCode() string {
	const charset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	b := make([]byte, 6)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}
