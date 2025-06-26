package admin_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	adminapi "github.com/Nixie-Tech-LLC/medusa/internal/api/admin"
	"github.com/Nixie-Tech-LLC/medusa/internal/auth"
	"github.com/Nixie-Tech-LLC/medusa/internal/db"
	"github.com/gin-gonic/gin"
)

func setupRouter(secret string, store db.Store) *gin.Engine {
	r := gin.Default()

	group := r.Group("/api/admin")

	adminapi.RegisterAuthRoutes(group, secret, store)

	protected := group.Group("/")

	protected.Use(auth.JWTMiddleware(secret))
	adminapi.RegisterScreenRoutes(protected)
	adminapi.RegisterContentRoutes(protected)
	adminapi.RegisterScheduleRoutes(protected)

	return r
}

func TestSignupLoginAndProfile(t *testing.T) {
	jwtSecret := "supersecret"
	mockStore := db.NewStore()
	router := setupRouter(jwtSecret, mockStore)

	signupBody := map[string]interface{}{
		"email":    "test@example.com",
		"password": "12345678",
	}
	signupJSON, _ := json.Marshal(signupBody)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/admin/auth/signup", bytes.NewReader(signupJSON))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Signup failed: %s", w.Body.String())
	}
	var signupResp map[string]string
	json.Unmarshal(w.Body.Bytes(), &signupResp)
	token := signupResp["token"]

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/admin/auth/current_profile", nil)
	router.ServeHTTP(w, req)
	if w.Code == http.StatusOK {
		t.Fatalf("Expected unauthorized without token")
	}

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/admin/auth/current_profile", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Current profile failed: %s", w.Body.String())
	}
}
