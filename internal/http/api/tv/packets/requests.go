package packets

// REQUESTS FOR /api/tv/pair
type PairingRequest struct {
	DeviceID string `json:"device_id" binding:"required"`
}
