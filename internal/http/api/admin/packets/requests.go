package packets

// CreateContentRequest Request for creating new content; optional ScreenID to immediately show.
type CreateContentRequest struct {
	Name            string `json:"name"  binding:"required"`
	Type            string `json:"type"  binding:"required"`
	URL             string `json:"url"   binding:"required,url"`
	DefaultDuration int    `json:"default_duration" binding:"required"`
	ScreenID        *int   `json:"screen_id"`
}

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

type UpdateContentRequest struct {
	Name            *string `json:"name"`
	Type            *string `json:"type"`
	URL             *string `json:"url"`
	DefaultDuration *int    `json:"default_duration"`
}

type CreatePlaylistRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

type UpdatePlaylistRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
}

type AddPlaylistItemRequest struct {
	ContentID int  `json:"content_id" binding:"required"`
	Position  int  `json:"position"`
	Duration  *int `json:"duration"` // seconds; nil = use content.default_duration
}

type UpdatePlaylistItemRequest struct {
	Position *int `json:"position"`
	Duration *int `json:"duration"`
}

type AssignPlaylistToScreenRequest struct {
	PlaylistID int `json:"playlist_id" binding:"required"`
}
