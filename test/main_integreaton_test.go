package test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Nixie-Tech-LLC/medusa/internal/db"
	tvapi "github.com/Nixie-Tech-LLC/medusa/internal/http/api/tv/endpoints"
	"github.com/Nixie-Tech-LLC/medusa/internal/redis"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// TestSetup handles setup for all integration tests
func TestSetup(t *testing.T) {
	// Create uploads directory if it doesn't exist
	uploadsDir := "./uploads"
	if err := os.MkdirAll(uploadsDir, 0755); err != nil {
		t.Fatalf("Failed to create uploads directory: %v", err)
	}
}

// TestTeardown handles cleanup for all integration tests
func TestTeardown(t *testing.T) {
	// Clean up any test files
	uploadsDir := "./uploads"
	if err := os.RemoveAll(uploadsDir); err != nil {
		t.Logf("Warning: Failed to clean up uploads directory: %v", err)
	}
}

// startTestServer starts the server and returns the command for cleanup
func startTestServer(t *testing.T) *exec.Cmd {
	cmd := exec.Command("go", "run", "main.go")
	err := cmd.Start()
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}

	// Give the server time to start
	time.Sleep(2 * time.Second)

	return cmd
}

// cleanupTestServer ensures the server process is killed
func cleanupTestServer(cmd *exec.Cmd) {
	if cmd != nil && cmd.Process != nil {
		cmd.Process.Kill()
		cmd.Wait() // Wait for process to fully terminate
	}
}

// createTestFile creates a temporary test file and returns cleanup function
func createTestFile(t *testing.T, filename, content string) (string, func()) {
	uploadsDir := "./uploads"
	if err := os.MkdirAll(uploadsDir, 0755); err != nil {
		t.Fatalf("Failed to create uploads directory: %v", err)
	}

	filepath := filepath.Join(uploadsDir, filename)
	err := os.WriteFile(filepath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	cleanup := func() {
		os.Remove(filepath)
	}

	return filepath, cleanup
}

// assertResponseBody reads and asserts the response body
func assertResponseBody(t *testing.T, resp *httptest.ResponseRecorder, expectedBody string) {
	body, err := io.ReadAll(resp.Body)
	assert.NoError(t, err, "Failed to read response body")
	assert.Equal(t, expectedBody, string(body), "Response body should match expected")
}

// assertJSONResponse reads and asserts JSON response
func assertJSONResponse(t *testing.T, resp *httptest.ResponseRecorder, expected interface{}) {
	body, err := io.ReadAll(resp.Body)
	assert.NoError(t, err, "Failed to read response body")

	var actual interface{}
	err = json.Unmarshal(body, &actual)
	assert.NoError(t, err, "Response should be valid JSON")

	assert.Equal(t, expected, actual, "JSON response should match expected")
}

func TestIntegration(t *testing.T) {
	// Setup and teardown for all tests
	TestSetup(t)
	defer TestTeardown(t)

	t.Run("Server Infrastructure", func(t *testing.T) {
		t.Run("Server Starts and Port is Open", func(t *testing.T) {
			cmd := startTestServer(t)
			defer cleanupTestServer(cmd)

			address := "localhost:8080"
			conn, err := net.DialTimeout("tcp", address, 2*time.Second)
			assert.NoError(t, err, "Server not listening on %s", address)
			if conn != nil {
				conn.Close()
			}
		})
	})

	t.Run("Database Infrastructure", func(t *testing.T) {
		t.Run("DB Init Fails With Invalid URL", func(t *testing.T) {
			badURL := "postgres://fakeuser:fakepass@localhost:9999/nodb"
			start := time.Now()
			err := db.Init(badURL)
			duration := time.Since(start)

			assert.Error(t, err)
			assert.Contains(t, err.Error(), "could not connect to database")

			// It should take at least 20 seconds due to retries
			expected := 20 * time.Second
			assert.GreaterOrEqual(t, int(duration.Seconds()), int(expected.Seconds())-1)
		})

		t.Run("Run Migrations With Missing Path", func(t *testing.T) {
			fakePath := "./does-not-exist"
			err := db.RunMigrations(fakePath)
			assert.NoError(t, err, "Expected no error even if migration path is empty")
		})
	})

	t.Run("Redis Infrastructure", func(t *testing.T) {
		t.Run("Redis Init and Ping", func(t *testing.T) {
			t.Parallel() // Safe to run in parallel
			redis.InitRedis("localhost:6379", "", "")
			assert.NotNil(t, redis.Rdb, "Redis client should not be nil after InitRedis")

			err := redis.Rdb.Ping(context.Background()).Err()
			assert.NoError(t, err, "Redis Ping should succeed")
		})
	})

	t.Run("File Operations", func(t *testing.T) {
		t.Run("Upload and Save File", func(t *testing.T) {
			t.Parallel() // Safe to run in parallel
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
			assertResponseBody(t, resp, "uploaded")

			// Confirm file exists
			savedPath := "./uploads/testfile.txt"
			_, err = os.Stat(savedPath)
			assert.NoError(t, err, "Saved file should exist")

			// Read file and confirm content
			content, err := os.ReadFile(savedPath)
			assert.NoError(t, err)
			assert.Equal(t, testContent, string(content))

			// Cleanup: remove the file
			os.Remove(savedPath)
		})

		t.Run("File Upload Scenarios", func(t *testing.T) {
			t.Parallel() // Safe to run in parallel
			router := gin.Default()
			router.POST("/upload", func(ctx *gin.Context) {
				file, err := ctx.FormFile("source")
				if err != nil {
					ctx.String(http.StatusBadRequest, "no file provided")
					return
				}

				// Check file size
				if file.Size == 0 {
					ctx.String(http.StatusBadRequest, "empty file")
					return
				}

				savePath := "./uploads/" + file.Filename
				err = ctx.SaveUploadedFile(file, savePath)
				if err != nil {
					ctx.String(http.StatusInternalServerError, "failed to save file")
					return
				}

				ctx.String(http.StatusOK, "uploaded")
			})

			tests := []struct {
				name           string
				filename       string
				content        string
				expectedStatus int
				expectedBody   string
			}{
				{"Valid File", "test.txt", "content", 200, "uploaded"},
				{"Empty File", "empty.txt", "", 400, "empty file"},
				{"Large File", "large.txt", strings.Repeat("a", 1000), 200, "uploaded"},
				{"Special Characters", "test-file_123.txt", "content with spaces", 200, "uploaded"},
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					body := &bytes.Buffer{}
					writer := multipart.NewWriter(body)
					part, err := writer.CreateFormFile("source", tt.filename)
					assert.NoError(t, err)

					_, err = part.Write([]byte(tt.content))
					assert.NoError(t, err)
					writer.Close()

					req := httptest.NewRequest(http.MethodPost, "/upload", body)
					req.Header.Set("Content-Type", writer.FormDataContentType())
					resp := httptest.NewRecorder()

					router.ServeHTTP(resp, req)

					assert.Equal(t, tt.expectedStatus, resp.Code, "Status code should match")
					assertResponseBody(t, resp, tt.expectedBody)

					// Cleanup if file was created
					if tt.expectedStatus == 200 {
						savedPath := "./uploads/" + tt.filename
						os.Remove(savedPath)
					}
				})
			}
		})

		t.Run("Upload with Invalid File", func(t *testing.T) {
			t.Parallel() // Safe to run in parallel
			router := gin.Default()
			router.POST("/upload", func(ctx *gin.Context) {
				file, err := ctx.FormFile("source")
				if err != nil {
					ctx.String(http.StatusBadRequest, "no file provided")
					return
				}

				savePath := "./uploads/" + file.Filename
				err = ctx.SaveUploadedFile(file, savePath)
				if err != nil {
					ctx.String(http.StatusInternalServerError, "failed to save file")
					return
				}

				ctx.String(http.StatusOK, "uploaded")
			})

			// Make request without file
			req := httptest.NewRequest(http.MethodPost, "/upload", strings.NewReader(""))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			resp := httptest.NewRecorder()

			router.ServeHTTP(resp, req)

			// Assert error response
			assert.Equal(t, http.StatusBadRequest, resp.Code)
			assertResponseBody(t, resp, "no file provided")
		})

		t.Run("Dummy Storage Save File", func(t *testing.T) {
			t.Parallel() // Safe to run in parallel
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
		})
	})

	t.Run("API Endpoints", func(t *testing.T) {
		t.Run("TV Ping Route", func(t *testing.T) {
			t.Parallel() // Safe to run in parallel
			// Init Redis (assumes Redis is running locally)
			redis.InitRedis("localhost:6379", "", "")

			router := gin.Default()
			store := db.NewStore(nil)
			tvapi.RegisterPairingRoutes(router.Group("/api/tv"), store)

			req := httptest.NewRequest(http.MethodGet, "/api/tv/ping", nil)
			resp := httptest.NewRecorder()

			router.ServeHTTP(resp, req)

			assert.Equal(t, http.StatusOK, resp.Code, "Expected 200 OK from /api/tv/ping")

			// Assert response body contains expected content
			body, err := io.ReadAll(resp.Body)
			assert.NoError(t, err, "Failed to read response body")
			assert.Contains(t, string(body), "pong", "Response should contain 'pong'")
		})

		t.Run("TV Ping Route with JSON Response", func(t *testing.T) {
			t.Parallel() // Safe to run in parallel
			// Init Redis (assumes Redis is running locally)
			redis.InitRedis("localhost:6379", "", "")

			router := gin.Default()
			store := db.NewStore(nil)
			tvapi.RegisterPairingRoutes(router.Group("/api/tv"), store)

			req := httptest.NewRequest(http.MethodGet, "/api/tv/ping", nil)
			req.Header.Set("Accept", "application/json")
			resp := httptest.NewRecorder()

			router.ServeHTTP(resp, req)

			assert.Equal(t, http.StatusOK, resp.Code, "Expected 200 OK from /api/tv/ping")

			// Assert JSON response structure
			body, err := io.ReadAll(resp.Body)
			assert.NoError(t, err, "Failed to read response body")

			var response map[string]interface{}
			err = json.Unmarshal(body, &response)
			assert.NoError(t, err, "Response should be valid JSON")
			assert.Contains(t, response, "status", "JSON response should contain 'status' field")
		})

		t.Run("Non-existent Route Returns 404", func(t *testing.T) {
			t.Parallel() // Safe to run in parallel
			router := gin.Default()
			store := db.NewStore(nil)
			tvapi.RegisterPairingRoutes(router.Group("/api/tv"), store)

			req := httptest.NewRequest(http.MethodGet, "/api/tv/nonexistent", nil)
			resp := httptest.NewRecorder()

			router.ServeHTTP(resp, req)

			assert.Equal(t, http.StatusNotFound, resp.Code, "Expected 404 for non-existent route")
		})
	})

	t.Run("Performance", func(t *testing.T) {
		t.Run("API Response Time", func(t *testing.T) {
			t.Parallel() // Safe to run in parallel
			redis.InitRedis("localhost:6379", "", "")
			router := gin.Default()
			store := db.NewStore(nil)
			tvapi.RegisterPairingRoutes(router.Group("/api/tv"), store)

			req := httptest.NewRequest(http.MethodGet, "/api/tv/ping", nil)
			resp := httptest.NewRecorder()

			start := time.Now()
			router.ServeHTTP(resp, req)
			duration := time.Since(start)

			assert.Equal(t, http.StatusOK, resp.Code, "Expected 200 OK")
			assert.Less(t, duration, 100*time.Millisecond, "Response should be under 100ms")
		})

		t.Run("File Upload Performance", func(t *testing.T) {
			t.Parallel() // Safe to run in parallel
			router := gin.Default()
			router.POST("/upload", func(ctx *gin.Context) {
				file, err := ctx.FormFile("source")
				assert.NoError(t, err)

				savePath := "./uploads/" + file.Filename
				err = ctx.SaveUploadedFile(file, savePath)
				assert.NoError(t, err)

				ctx.String(http.StatusOK, "uploaded")
			})

			// Create test file
			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)
			part, err := writer.CreateFormFile("source", "perf-test.txt")
			assert.NoError(t, err)

			// Write 1KB of content
			content := strings.Repeat("a", 1024)
			_, err = part.Write([]byte(content))
			assert.NoError(t, err)
			writer.Close()

			req := httptest.NewRequest(http.MethodPost, "/upload", body)
			req.Header.Set("Content-Type", writer.FormDataContentType())
			resp := httptest.NewRecorder()

			start := time.Now()
			router.ServeHTTP(resp, req)
			duration := time.Since(start)

			assert.Equal(t, http.StatusOK, resp.Code, "Expected 200 OK")
			assert.Less(t, duration, 500*time.Millisecond, "File upload should be under 500ms")

			// Cleanup
			os.Remove("./uploads/perf-test.txt")
		})

		t.Run("Redis Performance", func(t *testing.T) {
			t.Parallel() // Safe to run in parallel
			redis.InitRedis("localhost:6379", "", "")

			// Test Redis ping performance
			start := time.Now()
			err := redis.Rdb.Ping(context.Background()).Err()
			duration := time.Since(start)

			assert.NoError(t, err, "Redis ping should succeed")
			assert.Less(t, duration, 10*time.Millisecond, "Redis ping should be under 10ms")

			// Test Redis set/get performance
			key := "perf:test:key"
			value := "test_value"

			start = time.Now()
			err = redis.Rdb.Set(context.Background(), key, value, 0).Err()
			setDuration := time.Since(start)

			assert.NoError(t, err, "Redis set should succeed")
			assert.Less(t, setDuration, 10*time.Millisecond, "Redis set should be under 10ms")

			start = time.Now()
			result, err := redis.Rdb.Get(context.Background(), key).Result()
			getDuration := time.Since(start)

			assert.NoError(t, err, "Redis get should succeed")
			assert.Equal(t, value, result, "Redis get should return correct value")
			assert.Less(t, getDuration, 10*time.Millisecond, "Redis get should be under 10ms")

			// Cleanup
			redis.Rdb.Del(context.Background(), key)
		})
	})

	t.Run("Authentication", func(t *testing.T) {
		t.Run("Unauthorized Access", func(t *testing.T) {
			t.Parallel() // Safe to run in parallel
			router := gin.Default()

			// Add a protected route
			protected := router.Group("/api/admin")
			protected.Use(func(c *gin.Context) {
				authHeader := c.GetHeader("Authorization")
				if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
					c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required. Please provide a valid Bearer token in the Authorization header."})
					c.Abort()
					return
				}
				c.Next()
			})

			protected.GET("/protected", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "Access granted. You are successfully authenticated."})
			})

			// Test without auth header
			req := httptest.NewRequest(http.MethodGet, "/api/admin/protected", nil)
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			assert.Equal(t, http.StatusUnauthorized, resp.Code, "Protected endpoint should return 401 Unauthorized when no authentication token is provided")

			body, err := io.ReadAll(resp.Body)
			assert.NoError(t, err, "Failed to read response body from unauthorized request")
			assert.Contains(t, string(body), "Authentication required", "Error message should clearly explain that authentication is required")
		})

		t.Run("Invalid Token", func(t *testing.T) {
			t.Parallel() // Safe to run in parallel
			router := gin.Default()

			// Add a protected route
			protected := router.Group("/api/admin")
			protected.Use(func(c *gin.Context) {
				authHeader := c.GetHeader("Authorization")
				if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
					c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required. Please provide a valid Bearer token in the Authorization header."})
					c.Abort()
					return
				}

				token := strings.TrimPrefix(authHeader, "Bearer ")
				if token == "invalid-token" {
					c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authentication token. Please provide a valid Bearer token."})
					c.Abort()
					return
				}

				c.Next()
			})

			protected.GET("/protected", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "Access granted. You are successfully authenticated."})
			})

			// Test with invalid token
			req := httptest.NewRequest(http.MethodGet, "/api/admin/protected", nil)
			req.Header.Set("Authorization", "Bearer invalid-token")
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			assert.Equal(t, http.StatusUnauthorized, resp.Code, "Protected endpoint should return 401 Unauthorized when an invalid authentication token is provided")

			body, err := io.ReadAll(resp.Body)
			assert.NoError(t, err, "Failed to read response body from invalid token request")
			assert.Contains(t, string(body), "Invalid authentication token", "Error message should clearly explain that the provided token is invalid")
		})

		t.Run("Valid Token", func(t *testing.T) {
			t.Parallel() // Safe to run in parallel
			router := gin.Default()

			// Add a protected route
			protected := router.Group("/api/admin")
			protected.Use(func(c *gin.Context) {
				authHeader := c.GetHeader("Authorization")
				if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
					c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required. Please provide a valid Bearer token in the Authorization header."})
					c.Abort()
					return
				}

				token := strings.TrimPrefix(authHeader, "Bearer ")
				if token == "invalid-token" {
					c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authentication token. Please provide a valid Bearer token."})
					c.Abort()
					return
				}

				c.Next()
			})

			protected.GET("/protected", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "Access granted. You are successfully authenticated."})
			})

			// Test with valid token
			req := httptest.NewRequest(http.MethodGet, "/api/admin/protected", nil)
			req.Header.Set("Authorization", "Bearer valid-token")
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			assert.Equal(t, http.StatusOK, resp.Code, "Protected endpoint should return 200 OK when a valid authentication token is provided")

			body, err := io.ReadAll(resp.Body)
			assert.NoError(t, err, "Failed to read response body from valid token request")
			assert.Contains(t, string(body), "Access granted", "Success message should clearly indicate that access has been granted")
		})
	})

	t.Run("Load Testing", func(t *testing.T) {
		t.Run("Concurrent API Requests", func(t *testing.T) {
			t.Parallel() // Safe to run in parallel
			redis.InitRedis("localhost:6379", "", "")
			router := gin.Default()
			store := db.NewStore(nil)
			tvapi.RegisterPairingRoutes(router.Group("/api/tv"), store)

			const numRequests = 10
			results := make(chan bool, numRequests)
			errors := make(chan error, numRequests)

			// Launch concurrent requests
			for i := 0; i < numRequests; i++ {
				go func(id int) {
					req := httptest.NewRequest(http.MethodGet, "/api/tv/ping", nil)
					resp := httptest.NewRecorder()
					router.ServeHTTP(resp, req)

					if resp.Code == http.StatusOK {
						results <- true
					} else {
						errors <- fmt.Errorf("concurrent request %d failed with unexpected status code %d", id, resp.Code)
					}
				}(i)
			}

			// Wait for all requests to complete
			successCount := 0
			for i := 0; i < numRequests; i++ {
				select {
				case <-results:
					successCount++
				case err := <-errors:
					t.Errorf("Concurrent API request failed: %v", err)
				case <-time.After(5 * time.Second):
					t.Error("Timeout waiting for concurrent API requests to complete")
					return
				}
			}

			assert.Equal(t, numRequests, successCount, "All concurrent API requests should succeed. Expected %d successful requests, but got %d", numRequests, successCount)
		})

		t.Run("Concurrent File Uploads", func(t *testing.T) {
			t.Parallel() // Safe to run in parallel
			router := gin.Default()
			router.POST("/upload", func(ctx *gin.Context) {
				file, err := ctx.FormFile("source")
				if err != nil {
					ctx.String(http.StatusBadRequest, "No file was provided in the request. Please include a file in the 'source' field.")
					return
				}

				savePath := "./uploads/" + file.Filename
				err = ctx.SaveUploadedFile(file, savePath)
				if err != nil {
					ctx.String(http.StatusInternalServerError, "Failed to save uploaded file to server. Please try again.")
					return
				}

				ctx.String(http.StatusOK, "File uploaded successfully")
			})

			const numUploads = 5
			results := make(chan bool, numUploads)
			errors := make(chan error, numUploads)

			// Launch concurrent uploads
			for i := 0; i < numUploads; i++ {
				go func(id int) {
					body := &bytes.Buffer{}
					writer := multipart.NewWriter(body)
					part, err := writer.CreateFormFile("source", fmt.Sprintf("concurrent-test-%d.txt", id))
					if err != nil {
						errors <- fmt.Errorf("failed to create form file for concurrent upload %d: %v", id, err)
						return
					}

					content := fmt.Sprintf("content for file %d", id)
					_, err = part.Write([]byte(content))
					if err != nil {
						errors <- fmt.Errorf("failed to write content for concurrent upload %d: %v", id, err)
						return
					}
					writer.Close()

					req := httptest.NewRequest(http.MethodPost, "/upload", body)
					req.Header.Set("Content-Type", writer.FormDataContentType())
					resp := httptest.NewRecorder()

					router.ServeHTTP(resp, req)

					if resp.Code == http.StatusOK {
						results <- true
					} else {
						errors <- fmt.Errorf("concurrent file upload %d failed with status code %d", id, resp.Code)
					}
				}(i)
			}

			// Wait for all uploads to complete
			successCount := 0
			for i := 0; i < numUploads; i++ {
				select {
				case <-results:
					successCount++
				case err := <-errors:
					t.Errorf("Concurrent file upload failed: %v", err)
				case <-time.After(10 * time.Second):
					t.Error("Timeout waiting for concurrent file uploads to complete")
					return
				}
			}

			assert.Equal(t, numUploads, successCount, "All concurrent file uploads should succeed. Expected %d successful uploads, but got %d", numUploads, successCount)

			// Cleanup uploaded files
			for i := 0; i < numUploads; i++ {
				os.Remove(fmt.Sprintf("./uploads/concurrent-test-%d.txt", i))
			}
		})

		t.Run("Redis Concurrent Operations", func(t *testing.T) {
			t.Parallel() // Safe to run in parallel
			redis.InitRedis("localhost:6379", "", "")

			const numOperations = 20
			results := make(chan bool, numOperations)
			errors := make(chan error, numOperations)

			// Launch concurrent Redis operations
			for i := 0; i < numOperations; i++ {
				go func(id int) {
					key := fmt.Sprintf("load:test:key:%d", id)
					value := fmt.Sprintf("value_%d", id)

					// Set value
					err := redis.Rdb.Set(context.Background(), key, value, 0).Err()
					if err != nil {
						errors <- fmt.Errorf("Redis set operation failed for key '%s': %v", key, err)
						return
					}

					// Get value
					result, err := redis.Rdb.Get(context.Background(), key).Result()
					if err != nil {
						errors <- fmt.Errorf("Redis get operation failed for key '%s': %v", key, err)
						return
					}

					if result == value {
						results <- true
					} else {
						errors <- fmt.Errorf("Redis data integrity check failed for key '%s': expected value '%s', but got '%s'", key, value, result)
					}

					// Cleanup
					redis.Rdb.Del(context.Background(), key)
				}(i)
			}

			// Wait for all operations to complete
			successCount := 0
			for i := 0; i < numOperations; i++ {
				select {
				case <-results:
					successCount++
				case err := <-errors:
					t.Errorf("Concurrent Redis operation failed: %v", err)
				case <-time.After(5 * time.Second):
					t.Error("Timeout waiting for concurrent Redis operations to complete")
					return
				}
			}

			assert.Equal(t, numOperations, successCount, "All concurrent Redis operations should succeed. Expected %d successful operations, but got %d", numOperations, successCount)
		})
	})
}
