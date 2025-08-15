package model

import "time"

// Screen represents a display device in the system.
type Screen struct {
	ID                int       `db:"id"           json:"id"`
	DeviceID          *string   `db:"device_id"    json:"device_id"`
	ClientInformation *string   `db:"client_information" json:"client_information"`
	ClientWidth       *int      `db:"client_width"  json:"client_width"`
	ClientHeight      *int      `db:"client_height"  json:"client_height"`
	Name              string    `db:"name"         json:"name"`
	Location          *string   `db:"location"     json:"location"`
	StorageSize       *int64    `db:"storage_size"  json:"storage_size"`
	IPAddress         *string   `db:"ip_address"           json:"ip_address"`
	Paired            bool      `db:"paired"       json:"paired"`
	CreatedAt         time.Time `db:"created_at"   json:"created_at"`
	CreatedBy         int       `db:"created_by"   json:"created_by"`
	UpdatedAt         time.Time `db:"updated_at"   json:"updated_at"`
}

type ScreenGroup struct {
	ID          int       `db:"id"`
	Name        string    `db:"name"`
	Description *string   `db:"description"`
	CreatedBy   int       `db:"created_by"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}
