package model

import "time"

type User struct {
    ID             int       `db:"id"`
    Email          string    `db:"email"`
    HashedPassword string    `db:"hashed_password"`
    Name           *string   `db:"name"`
    CreatedAt      time.Time `db:"created_at"`
    UpdatedAt      time.Time `db:"updated_at"`
}

