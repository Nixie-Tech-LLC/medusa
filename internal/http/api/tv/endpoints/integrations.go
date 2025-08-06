package endpoints
import ( 
	"time"
	"fmt"
	"net/http"
	"strconv"
	"encoding/json"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/Nixie-Tech-LLC/medusa/internal/model"
)

func RegisterIntegrationRoutes(r gin.IRoutes) {
	r.GET("/integrations/:name", serveIntegration)
}

func serveIntegration(ctx *gin.Context) {
	name := ctx.Param("name")
	switch name {
	case "athan":
		serveAthan(ctx)
	default:
		ctx.String(http.StatusNotFound, "integration not found")
	}
}

func serveAthan(ctx *gin.Context) {
	now := time.Now()
	dateStr := now.Format("JANUARY 2, 2006")

	lat, lon := 41.8781, -87.6298 
	resp, err := http.Get(
		fmt.Sprintf("https://api.aladhan.com/v1/timings?latitude=%f&longitude=%f&method=2",
			lat, lon,
			),
		)
	if err != nil || resp.StatusCode != http.StatusOK {
		ctx.String(http.StatusInternalServerError, "failed to get prayer times")
		return
	}
	var aladan struct {
		Data struct {
			Timings map[string]string `json:"timings"`
		} `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&aladan)

	city := "CHICAGO"

	order := []string{"Fajr", "Dhuhr", "Asr", "Maghrib", "Isha"}
	prayers := make([]model.Prayer, len(order))
	for i, nm := range order {
		t24 := aladan.Data.Timings[nm]
		// convert "17:30" → ("05:30","PM")
		parts := strings.Split(t24, ":")
		h, _ := strconv.Atoi(parts[0])
		m := parts[1]
		period := "AM"
		if h >= 12 {
			period = "PM"
			if h > 12 {
				h -= 12
			}
		}
		time12 := fmt.Sprintf("%02d:%s", h, m)
		prayers[i] = model.Prayer{
			Name:   strings.ToUpper(nm),
			Time:   time12,
			Period: period,
			Iqama:  "00:00", // or compute if you need
		}
	}

	data := model.AthanPageData{
		City:    city,
		Date:    dateStr,
		Prayers: prayers,
	}

	ctx.HTML(http.StatusOK, "athan.html", data)
}

