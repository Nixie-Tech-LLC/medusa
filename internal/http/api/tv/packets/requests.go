package packets

// REQUESTS FOR /api/tv/pair
type TVRequest struct {
	DeviceID string `json:"device_id" binding:"required"`
}
