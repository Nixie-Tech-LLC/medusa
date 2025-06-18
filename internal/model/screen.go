package model

import "time"

// 'Screen' represents a display device in the system.
type Screen struct {
	ID        int       `db:"id"           json:"id"`
	DeviceID  *string   `db:"device_id"    json:"device_id"`
	Name      string    `db:"name"         json:"name"`
	Location  *string   `db:"location"     json:"location"`
	Paired    bool      `db:"paired"       json:"paired"`
	CreatedAt time.Time `db:"created_at"   json:"created_at"`
	UpdatedAt time.Time `db:"updated_at"   json:"updated_at"`
}
