package packets

// REQUESTS FOR /api/tv/pair
type PairingRequest struct {
	DeviceID string `json:"device_id" binding:"required"`
}

// REQUESTS FOR /api/tv/screens/*
type CreateScreenRequest struct {
	Name     string  `json:"name" binding:"required"`
	Location *string `json:"location"`
}

type UpdateScreenRequest struct {
	Name     *string `json:"name"`
	Location *string `json:"location"`
}

type AssignScreenRequest struct {
	UserID int `json:"user_id" binding:"required"`
}

type AssignContentToScreenRequest struct {
	ContentID int `json:"content_id" binding:"required"`
}

type PairScreenRequest struct {
	PairingCode string `json:"code" binding:"required"`
	UserID      int    `json:"user_id" binding:"required"`
	ScreenID    int    `json:"screen_id" binding:"required"`
}
