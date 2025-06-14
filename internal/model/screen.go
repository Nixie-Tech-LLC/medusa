package model

import "time"

// 'Screen' represents a display device in the system.
type Screen struct {
    ID          int        `db:"id"           json:"id"`
    Name        string     `db:"name"         json:"name"`
    Location    *string    `db:"location"     json:"location,omitempty"`
    Paired      bool       `db:"paired"       json:"paired"`
    PairingCode string     `db:"pairing_code" json:"pairing_code,omitempty"`
    CreatedAt   time.Time  `db:"created_at"   json:"created_at"`
    UpdatedAt   time.Time  `db:"updated_at"   json:"updated_at"`
}

