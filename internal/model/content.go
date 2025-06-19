package model

import "time"
import "encoding/json"

type Content struct {
    ID              int             `db:"id"           json:"id"`
    Name            string          `db:"name"         json:"name"`
    Type            string          `db:"type"         json:"type"`
    URL             string          `db:"url"          json:"url"`
    Metadata        json.RawMessage `db:"metadata"     json:"metadata"`
    DefaultDuration int             `db:"default_duration" json:"default_duration"` 
    CreatedAt       time.Time       `db:"created_at"   json:"created_at"`
    CreatedBy       int             `db:"created_by"   json:"created_by"`
    UpdatedAt       time.Time       `db:"updated_at"   json:"updated_at"`
}

