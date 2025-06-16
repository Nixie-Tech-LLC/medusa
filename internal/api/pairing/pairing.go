package admin

import (
	"math/rand"
	"time"

	redisclient "github.com/Nixie-Tech-LLC/medusa/internal/redis"
	"github.com/gin-gonic/gin"
)

func RegisterPairingRoutes(r gin.IRoutes) {
	r.POST("/request", requestPairing)
}

func requestPairing(c *gin.Context) {
	var req struct {
		DeviceID string `json:"deviceId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	code := generatePairCode()
	key := "pairing:" + code

	err := redisclient.Rdb.Set(c, key, req.DeviceID, 5*time.Minute).Err()
	if err != nil {
		c.JSON(500, gin.H{"error": "internal error"})
		return
	}

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
