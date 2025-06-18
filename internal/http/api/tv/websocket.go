package admin

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var tvSockets = map[int]*websocket.Conn{} // screen_id => conn
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func TVWebSocket(secret string) gin.HandlerFunc {
	// return func(c *gin.Context) {
	// 	tokenStr := c.Query("token")
	// 	if tokenStr == "" {
	// 		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
	// 		return
	// 	}

	// 	claims, err := middleware.ParseJWT(tokenStr, secret)
	// 	if err != nil {
	// 		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
	// 		return
	// 	}

	// 	screenID := int(claims["screen_id"].(float64)) // jwt.MapClaims is float64

	// 	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	// 	if err != nil {
	// 		log.Println("websocket upgrade failed:", err)
	// 		return
	// 	}

	// 	log.Printf("WebSocket connected: screen %d", screenID)
	// 	tvSockets[screenID] = conn

	// 	defer func() {
	// 		log.Printf("WebSocket disconnected: screen %d", screenID)
	// 		delete(tvSockets, screenID)
	// 		conn.Close()
	// 	}()

	// 	// keep the connection alive
	// 	for {
	// 		if _, _, err := conn.ReadMessage(); err != nil {
	// 			break
	// 		}
	// 	}
	// }
}
