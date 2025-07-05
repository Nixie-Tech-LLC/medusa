package packets

// CreateContentRequest Request for creating new content; optional ScreenID to immediately show.
type CreateContentRequest struct {
	Name     string `json:"name"  binding:"required"`
	Type     string `json:"type"  binding:"required"`
	URL      string `json:"url"   binding:"required,url"`
	ScreenID *int   `json:"screen_id"`
}

// CreateScreenRequest REQUESTS FOR /api/tv/screens/*
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
	ScreenID    int    `json:"screen_id" binding:"required"`
}
