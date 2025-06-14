package auth

import (
    "errors"
    "net/http"
    "strings"
    "time"

    "github.com/dgrijalva/jwt-go" 
    "github.com/gin-gonic/gin"
    "golang.org/x/crypto/bcrypt"

    "github.com/Nixie-Tech-LLC/medusa/internal/db"
    "github.com/Nixie-Tech-LLC/medusa/internal/model"
)

// is returned when email/password don’t match.
var ErrInvalidCredentials = errors.New("invalid email or password")

// uses bcrypt to hash a plaintext password.
func HashPassword(plain string) (string, error) {
    bytes, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
    return string(bytes), err
}

// compares a bcrypt hash with the plaintext.
func CheckPassword(hash, plain string) bool {
    err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain))
    return err == nil
}

// signs a token embedding userID in the “sub” claim.
func GenerateJWT(userID int, secret string) (string, error) {
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
        "sub": userID,
        "exp": time.Now().Add(72 * time.Hour).Unix(),
    })
    return token.SignedString([]byte(secret))
}

// verifies the JWT and returns the user ID (unexported, only used internally).
func parseToken(tokenString, secret string) (int, error) {
    token, err := jwt.Parse(tokenString, func(t *jwt.Token) (any, error) {
        if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, errors.New("unexpected signing method")
        }
        return []byte(secret), nil
    })
    if err != nil || !token.Valid {
        return 0, errors.New("invalid token")
    }
    claims, ok := token.Claims.(jwt.MapClaims)
    if !ok {
        return 0, errors.New("invalid claims")
    }
    sub, ok := claims["sub"].(float64)
    if !ok {
        return 0, errors.New("invalid sub claim")
    }
    return int(sub), nil
}

// checks “Authorization: Bearer <token>”, verifies it, loads user, and sets “currentUser” in context.
func JWTMiddleware(secret string) gin.HandlerFunc {
    return func(c *gin.Context) {
        header := c.GetHeader("Authorization")
        if header == "" {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing auth header"})
            return
        }

        parts := strings.SplitN(header, " ", 2)
        if len(parts) != 2 || parts[0] != "Bearer" {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid auth header"})
            return
        }

        userID, err := parseToken(parts[1], secret)
        if err != nil {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
            return
        }

        user, err := db.GetUserByID(userID)
        if err != nil {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
            return
        }
        c.Set("currentUser", user)
        c.Next()
    }
}

// retrieves *model.User from Gin context (after JWTMiddleware has run).
func GetCurrentUser(c *gin.Context) (*model.User, bool) {
    u, exists := c.Get("currentUser")
    if !exists {
        return nil, false
    }
    user, ok := u.(*model.User)
    return user, ok
}

