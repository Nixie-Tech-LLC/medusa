package packets

// RESPONSES FOR /api/tv/screens/*

import "time"

// Response mirrors model.Content but flattens time.
type ContentResponse struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	URL       string `json:"url"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	CreatedAt string `json:"created_at"`
}

// screenResponse mirrors model.Screen but flattens times to RFC3339
type ScreenResponse struct {
	ID                int     `json:"id"`
	DeviceID          *string `json:"device_id"`
	ClientInformation *string `json:"client_information"`
	ClientWidth       *int    `json:"client_width"`
	ClientHeight      *int    `json:"client_height"`
	Name              string  `json:"name"`
	Location          *string `json:"location"`
	Paired            bool    `json:"paired"`
	CreatedAt         string  `json:"created_at"`
	UpdatedAt         string  `json:"updated_at"`
}

type PlaylistItemResponse struct {
	ID        int       `json:"id"`
	ContentID int       `json:"content_id"`
	Position  int       `json:"position"`
	Duration  int       `json:"duration"`
	CreatedAt time.Time `json:"created_at"`
}

type PlaylistResponse struct {
	ID          int                    `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	CreatedBy   int                    `json:"created_by"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Items       []PlaylistItemResponse `json:"items"`
}

// Simple response for TV clients - just URLs and durations
type TVPlaylistResponse struct {
	PlaylistName string          `json:"playlist_name"`
	ContentList  []TVContentItem `json:"content_list"`
}

type TVContentItem struct {
	URL      string `json:"url"`
	Duration int    `json:"duration"`
	Type     string `json:"type"`
}
