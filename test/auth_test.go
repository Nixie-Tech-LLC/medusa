package test

import (
    "bytes"
    "encoding/json"
    "os"
    "testing"

    "github.com/gin-gonic/gin"

    "net/http"
    "net/http/httptest"

    "github.com/Nixie-Tech-LLC/medusa/internal/db"
    adminEndpoints "github.com/Nixie-Tech-LLC/medusa/internal/http/api/admin/endpoints"
    "github.com/Nixie-Tech-LLC/medusa/internal/http/middleware"
)

var router *gin.Engine

// TestMain runs once for the whole package.
func TestMain(m *testing.M) {
    gin.SetMode(gin.TestMode)

    db.InitTestDB()

    mustCreateTestUser("test@example.com", "testpassword")

    router = gin.New()
    router.Use(middleware.InjectStore(db.TestStore))
    authGroup := router.Group("/api/admin/auth")
    {
        adminEndpoints.RegisterAuthRoutes(authGroup)
    }

    os.Exit(m.Run())
}

// mustCreateTestUser inserts a user into the test DB and panics on error.
func mustCreateTestUser(email, pass string) {
    if _, err := db.TestStore.CreateUser(email, pass, nil); err != nil {
        panic("could not seed test user: " + err.Error())
    }
}

// TestLogin_Success checks that valid credentials return a token.
func TestLogin_Success(t *testing.T) {
    // build request payload
    payload := map[string]string{
        "email":    "test@example.com",
        "password": "testpassword",
    }
    body, _ := json.Marshal(payload)

    req := httptest.NewRequest(http.MethodPost, "/api/admin/auth/login", bytes.NewReader(body))
    req.Header.Set("Content-Type", "application/json")

    // record
    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)

    assert.Equal(t, http.StatusOK, w.Code)

    // parse response
    var resp struct {
        Token string `json:"token"`
    }
    err := json.Unmarshal(w.Body.Bytes(), &resp)
    assert.NoError(t, err, "response should be valid JSON")
    assert.NotEmpty(t, resp.Token, "token should not be empty")
}

