package admin_test

import (
	"encoding/json"
	"github.com/rs/zerolog/log"
	"testing"
	"time"

	"github.com/Nixie-Tech-LLC/medusa/internal/http/middleware"
)

func TestMQTTFunctionality(t *testing.T) {
	// This is a demonstration test showing how to use the MQTT functionality
	// Note: This test requires an MQTT broker to be running

	// Initialize MQTT (this would normally be done in main.go)
	err := middleware.InitMQTT()
	if err != nil {
		t.Skipf("MQTT broker not available, skipping test: %v", err)
	}
	defer middleware.CleanupMQTT()

	// Example: Send a content update message to a specific device
	updateMessage := map[string]interface{}{
		"type":         "content_update",
		"content_id":   123,
		"content_name": "Test Video",
		"content_type": "video",
		"content_url":  "https://example.com/test-video.mp4",
		"timestamp":    time.Now().Unix(),
	}

	messageBytes, err := json.Marshal(updateMessage)
	if err != nil {
		t.Fatalf("Failed to marshal message: %v", err)
	}

	// Note: This will fail if no device is connected, but that's expected
	// In a real scenario, devices would be connected first
	err = middleware.SendMessageToScreen("test-device-123", messageBytes)
	if err != nil {
		// This is expected if no device is connected
		t.Logf("Expected error (no device connected): %v", err)
	}

	// Example: Broadcast a message to all connected devices
	broadcastMessage := map[string]interface{}{
		"type":      "broadcast",
		"message":   "Hello all connected TVs!",
		"timestamp": time.Now().Unix(),
	}

	broadcastBytes, err := json.Marshal(broadcastMessage)
	if err != nil {
		t.Fatalf("Failed to marshal broadcast message: %v", err)
	}

	err = middleware.SendMessageToAllScreens(broadcastBytes)
	if err != nil {
		// This is expected if no devices are connected
		t.Logf("Expected error (no devices connected): %v", err)
	}

	// Get list of connected devices
	connectedDevices := middleware.GetConnectedTVs()
	t.Logf("Connected devices: %v", connectedDevices)
}

func TestMQTTConfiguration(t *testing.T) {
	// Test MQTT broker URL configuration

	// Set a custom broker URL
	middleware.SetBrokerURL("tcp://mqtt.example.com:1883")

	// Note: In a real implementation, you might want to verify the URL was set
	// This is just a demonstration of the API

	t.Log("MQTT configuration test completed")
}
