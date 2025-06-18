package packets

// Request for creating new content; optional ScreenID to immediately show.
type CreateContentRequest struct {
	Name     string `json:"name"  binding:"required"`
	Type     string `json:"type"  binding:"required"`
	URL      string `json:"url"   binding:"required,url"`
	ScreenID *int   `json:"screen_id"`
}
