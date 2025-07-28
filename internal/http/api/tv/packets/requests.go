package packets

// REQUESTS FOR /api/tv/pair
type TVRequest struct {
	PairingCode string `json:"code" binding:"required"`
	DeviceID    string `json:"device_id" binding:"required"`
}
