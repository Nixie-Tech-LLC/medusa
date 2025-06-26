package packets

// Request for creating new content; optional ScreenID to immediately show.
type CreateContentRequest struct {
	Name     string `json:"name"  binding:"required"`
	Type     string `json:"type"  binding:"required"`
	URL      string `json:"url"   binding:"required,url"`
	DefaultDuration int `json:"default_duration" binding:"required"`
	ScreenID *int   `json:"screen_id"`
}

type UpdateContentRequest struct {
    Name            *string         `json:"name"`
    Type            *string         `json:"type"`
    URL             *string         `json:"url"`
    DefaultDuration *int            `json:"default_duration"`
}

