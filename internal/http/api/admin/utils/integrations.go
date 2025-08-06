package utils

import (
	"time"
	"fmt"
	"encoding/json"
	"net/http"
	"github.com/Nixie-Tech-LLC/medusa/internal/http/api"
)

func setupAthan(config json.RawMessage) (string, *api.Error) {
	// athan needs latitude, longitude, and optional date
	var cfg struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
		Date      string  `json:"date"` // YYYY-MM-DD
	}

	if err := json.Unmarshal(config, &cfg); err != nil {
		return "", &api.Error{Code: http.StatusBadRequest, Message: "invalid config for athan"}
	}

	if cfg.Date == "" {
		cfg.Date = time.Now().Format("2006-01-02")
	}

	return fmt.Sprintf(
		"/api/tv/integrations/athan?lat=%f&lon=%f&date=%s",
		cfg.Latitude, cfg.Longitude, cfg.Date,
		), nil
}

