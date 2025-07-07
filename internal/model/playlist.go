package model

import "time"

type Playlist struct {
	ID          int       `db:"id"           json:"id"`
	Name        string    `db:"name"         json:"name"`
	Description *string   `db:"description"  json:"description,omitempty"`
	CreatedAt   time.Time `db:"created_at"   json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"   json:"updated_at"`
	CreatedBy   int       `db:"created_by"   json:"created_by"`
	Items       []PlaylistItem `json:"items,omitempty"`
}

type PlaylistItem struct {
	ID          int       `db:"id"           json:"id"`
	PlaylistID  int       `db:"playlist_id"  json:"playlist_id"`
	ContentID   int       `db:"content_id"   json:"content_id"`
	Position    int       `db:"position"     json:"position"`
	Duration    *int      `db:"duration"     json:"duration,omitempty"`
	CreatedAt   time.Time `db:"created_at"   json:"created_at"`
	CreatedBy   int       `db:"created_by"   json:"created_by"`
	Content     *Content  `db:"-"            json:"content,omitempty"`
}

type ScreenPlaylist struct {
	ID          int       `db:"id"           json:"id"`
	ScreenID    int       `db:"screen_id"    json:"screen_id"`
	PlaylistID  int       `db:"playlist_id"  json:"playlist_id"`
	Active      bool      `db:"active"       json:"active"`
	AssignedAt  time.Time `db:"assigned_at"  json:"assigned_at"`
	Playlist    *Playlist `db:"-"            json:"playlist,omitempty"`
}
