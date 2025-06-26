package middleware

import (
	"errors"

	"github.com/Nixie-Tech-LLC/medusa/internal/model"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
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

// retrieves *model.User from Gin context (after JWTMiddleware has run).
func GetCurrentUser(c *gin.Context) (*model.User, bool) {
	u, exists := c.Get("currentUser")
	if !exists {
		return nil, false
	}
	user, ok := u.(*model.User)
	return user, ok
}
