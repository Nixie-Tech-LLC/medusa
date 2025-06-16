package model

import "time"

type Content struct {
    ID        int       `db:"id"           json:"id"`
    Name      string    `db:"name"         json:"name"`
    Type      string    `db:"type"         json:"type"`
    URL       string    `db:"url"          json:"url"`
    CreatedAt time.Time `db:"created_at"   json:"created_at"`
}



