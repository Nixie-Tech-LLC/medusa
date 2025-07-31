package test

import (
	"bytes"
	"context"
	"github.com/Nixie-Tech-LLC/medusa/internal/db"
	tvapi "github.com/Nixie-Tech-LLC/medusa/internal/http/api/tv/endpoints"
	"github.com/Nixie-Tech-LLC/medusa/internal/redis"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestServerStartsAndPortIsOpen(t *testing.T) {
	// Start the server (just like running `go run main.go`)
	cmd := exec.Command("go", "run", "main.go")

	// Start the server process
	err := cmd.Start()
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}

	// Stop it at the end no matter what
	defer func() {
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
	}()

	// Give the server time to start
	time.Sleep(2 * time.Second)

	// Check if the port is open
	address := "localhost:8080" // <- this MUST match your env.SERVER_ADDRESS

	conn, err := net.DialTimeout("tcp", address, 2*time.Second)
	if err != nil {
		t.Fatalf("Server not listening on %s: %v", address, err)
	}
	conn.Close()
}

func TestDBInitFailsWithInvalidURL(t *testing.T) {
	// Use an obviously invalid URL that will always fail
	badURL := "postgres://fakeuser:fakepass@localhost:9999/nodb"

	start := time.Now()
	err := db.Init(badURL)
	duration := time.Since(start)

	// It should return an error
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "could not connect to database")

	// It should take at least 20 seconds due to retries
	expected := 20 * time.Second
	assert.GreaterOrEqual(t, int(duration.Seconds()), int(expected.Seconds())-1)
}
func TestRunMigrationsWithMissingPath(t *testing.T) {
	// Provide a path that definitely doesn't exist
	fakePath := "./does-not-exist"

	err := db.RunMigrations(fakePath)

	// Should NOT return an error — it's valid for zero .up.sql files
	assert.NoError(t, err, "Expected no error even if migration path is empty")
}

func TestInitRedisSetsClient(t *testing.T) {
	redis.InitRedis("localhost:6379", "", "")

	assert.NotNil(t, redis.Rdb, "Redis client should not be nil after InitRedis")
}

func TestInitRedisAndPing(t *testing.T) {
	redis.InitRedis("localhost:6379", "", "")

	assert.NotNil(t, redis.Rdb, "Redis client should not be nil after InitRedis")

	err := redis.Rdb.Ping(context.Background()).Err()
	assert.NoError(t, err, "Redis Ping should succeed")

}
func TestUploadAndSaveFile(t *testing.T) {
	router := gin.Default()
	router.POST("/upload", func(ctx *gin.Context) {
		file, err := ctx.FormFile("source")
		assert.NoError(t, err)

		savePath := "./uploads/" + file.Filename
		err = ctx.SaveUploadedFile(file, savePath)
		assert.NoError(t, err)

		ctx.String(http.StatusOK, "uploaded")
	})

	// Set up multipart form data with a dummy file
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("source", "testfile.txt")
	assert.NoError(t, err)

	// Write file contents
	testContent := "this is test file content"
	_, err = part.Write([]byte(testContent))
	assert.NoError(t, err)
	writer.Close()

	// Make the request
	req := httptest.NewRequest(http.MethodPost, "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp := httptest.NewRecorder()

	// Serve the request
	router.ServeHTTP(resp, req)

	// Assert the upload worked
	assert.Equal(t, http.StatusOK, resp.Code)

	// Confirm file exists
	savedPath := "./uploads/testfile.txt"
	_, err = os.Stat(savedPath)
	assert.NoError(t, err, "Saved file should exist")

	// Optionally: read file and confirm content
	content, err := os.ReadFile(savedPath)
	assert.NoError(t, err)
	assert.Equal(t, testContent, string(content))

	// Cleanup: remove the file
	_ = os.Remove(savedPath)
}
func TestDummyStorageSaveFile(t *testing.T) {
	type DummyStorage struct {
		Saved map[string]string
	}

	// Simulate a SaveFile method
	saveFile := func(ds *DummyStorage, path string, content string) error {
		if ds.Saved == nil {
			ds.Saved = make(map[string]string)
		}
		ds.Saved[path] = content
		return nil
	}

	// Create dummy storage instance
	ds := &DummyStorage{}

	// Simulate file upload
	err := saveFile(ds, "dummy.txt", "hello world")
	assert.NoError(t, err)

	// Assert it was saved
	assert.Contains(t, ds.Saved, "dummy.txt")
	assert.Equal(t, "hello world", ds.Saved["dummy.txt"])
}
func TestTVPingRoute(t *testing.T) {
	// Init Redis (assumes Redis is running locally)
	redis.InitRedis("localhost:6379", "", "")

	router := gin.Default()

	// You can pass nil store if ping doesn't use DB
	store := db.NewStore(nil)
	tvapi.RegisterPairingRoutes(router.Group("/api/tv"), store)

	req := httptest.NewRequest(http.MethodGet, "/api/tv/ping", nil)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code, "Expected 200 OK from /api/tv/ping")
}
