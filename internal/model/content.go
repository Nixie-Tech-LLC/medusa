package model

import "time"

type Content struct {
	ID        int       `db:"id"           json:"id"`
	Name      string    `db:"name"         json:"name"`
	Type      string    `db:"type"         json:"type"`
	URL       string    `db:"url"          json:"url"`
	Width     int       `db:"resolution_width"        json:"width"`
	Height    int       `db:"resolution_height"       json:"height"`
	CreatedAt time.Time `db:"created_at"   json:"created_at"`
	CreatedBy int       `db:"created_by"   json:"created_by"`
	UpdatedAt time.Time `db:"updated_at"   json:"updated_at"`
}
