// exposes a Store interface that is passed to API calls w/ param requirements 
package db 

import (
	"github.com/Nixie-Tech-LLC/medusa/internal/model"
    "github.com/jmoiron/sqlx"
)

type Store interface {
	// user functions
	CreateUser(email, hashedPassword string, name *string) (int, error) 
	GetUserByEmail(email string) (*model.User, error)
	GetUserByID(id int) (*model.User, error)
	UpdateUserProfile(id int, email string, name *string) error

	// screen functions
	GetScreenByID(id int)
}

type pgStore struct { 
	db *sqlx.DB
}

// compile-time check that pgStore implements Store
// required so linter doesn't complain
var _ Store = (*pgStore)(nil)

func NewStore() Store {
	return &pgStore{db: DB}
}

