package utils

import (
	"time"
	"fmt"
	"encoding/json"
	"net/http"
	"github.com/Nixie-Tech-LLC/medusa/internal/http/api"
)

func SetupAthan(config json.RawMessage) (string, *api.APIError) {
	// athan needs latitude, longitude, and optional date
	var cfg struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
		Date      string  `json:"date"` // YYYY-MM-DD
	}

	if err := json.Unmarshal(config, &cfg); err != nil {
		return "", &api.APIError{Code: http.StatusBadRequest, Message: "invalid config for athan"}
	}

	if cfg.Date == "" {
		cfg.Date = time.Now().Format("2006-01-02")
	}

	url := fmt.Sprintf(
		"/api/tv/integrations/athan?lat=%f&lon=%f&date=%s",
		cfg.Latitude, cfg.Longitude, cfg.Date,
		)

	return url, nil
}

// EnsureIntegrationURL returns a canonical URL for a given integration request.
// You already have SetupAthan(req.Config) used in addIntegration; we reuse that here.
func EnsureIntegrationURL(name string, cfg json.RawMessage) (string, *api.APIError) {
	switch name {
	case "athan":
		return SetupAthan(cfg) // existing function returns (url string, *api.APIError)
	default:
		return "", &api.APIError{Code: 400, Message: "unknown integration"}
	}
}
