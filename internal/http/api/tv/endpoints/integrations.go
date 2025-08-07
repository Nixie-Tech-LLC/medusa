package endpoints

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Nixie-Tech-LLC/medusa/internal/http/api"
	"github.com/Nixie-Tech-LLC/medusa/internal/model"
)

// IntegrationsModule mounts public /integrations endpoints under /api/tv.
// We bind raw gin handlers here since they return HTML (not JSON).
func IntegrationsModule() api.Module {
	return api.ModuleFunc(func(c *api.Controller) {
		c.Group.GET("/integrations/:name", serveIntegration)
	})
}

func serveIntegration(ctx *gin.Context) {
	switch ctx.Param("name") {
	case "athan":
		serveAthan(ctx)
	default:
		ctx.String(http.StatusNotFound, "integration not found")
	}
}

func serveAthan(ctx *gin.Context) {
	now := time.Now()
	dateStr := now.Format("JANUARY 2, 2006")

	// TODO: make location dynamic (query params or per-screen setting).
	lat, lon := 41.8781, -87.6298 // Chicago
	resp, err := http.Get(
		fmt.Sprintf(
			"https://api.aladhan.com/v1/timings?latitude=%f&longitude=%f&method=2",
			lat, lon,
		),
	)
	if err != nil || resp.StatusCode != http.StatusOK {
		if resp != nil {
			defer resp.Body.Close()
		}
		ctx.String(http.StatusInternalServerError, "failed to get prayer times")
		return
	}
	defer resp.Body.Close()

	var aladhan struct {
		Data struct {
			Timings map[string]string `json:"timings"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&aladhan); err != nil {
		ctx.String(http.StatusInternalServerError, "failed to parse prayer times")
		return
	}

	city := "CHICAGO"

	order := []string{"Fajr", "Dhuhr", "Asr", "Maghrib", "Isha"}
	prayers := make([]model.Prayer, len(order))
	for i, nm := range order {
		t24 := aladhan.Data.Timings[nm]
		time12, period := to12Hour(t24)
		prayers[i] = model.Prayer{
			Name:   strings.ToUpper(nm),
			Time:   time12,
			Period: period,
			Iqama:  "00:00", // placeholder (customize if needed)
		}
	}

	data := model.AthanPageData{
		City:    city,
		Date:    dateStr,
		Prayers: prayers,
	}

	ctx.HTML(http.StatusOK, "athan.html", data)
}

// to12Hour converts "HH:MM" → "hh:MM", "AM/PM"
func to12Hour(t24 string) (string, string) {
	parts := strings.Split(t24, ":")
	if len(parts) < 2 {
		return t24, "" // best-effort fallback
	}
	h, _ := strconv.Atoi(parts[0])
	m := parts[1]
	period := "AM"
	switch {
	case h == 0:
		h = 12
		period = "AM"
	case h == 12:
		period = "PM"
	case h > 12:
		h -= 12
		period = "PM"
	}
	return fmt.Sprintf("%02d:%s", h, m), period
}

