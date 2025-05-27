package auth

import "github.com/gin-gonic/gin"

// JWTMiddleware returns a Gin middleware for JWT validation using the provided secret.
func JWTMiddleware(secret string) gin.HandlerFunc {
    return func(c *gin.Context) {
        // TODO: implement JWT validation using the secret
        c.Next()
    }
}
