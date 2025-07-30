package db 

import (
	"database/sql"
	"errors"
	"github.com/rs/zerolog/log"
	_ "github.com/lib/pq"

	"github.com/Nixie-Tech-LLC/medusa/internal/model"
)

// inserts new user into table, returns new user ID.
func CreateUser(email, hashedPassword string, name *string) (int, error) {
	query := `
	INSERT INTO users (email, hashed_password, name, created_at, updated_at)
	VALUES ($1, $2, $3, now(), now())
	RETURNING id;
	`
	var newID int
	err := DB.QueryRow(query, email, hashedPassword, name).Scan(&newID)
	if err != nil {
		log.Error().Msg("failed to create user")
		return 0, err
	}
	return newID, nil
}

// fetches user by email. returns nil, sql.ErrNoRows if not found.
func GetUserByEmail(email string) (*model.User, error) {
	var u model.User
	query := `
	SELECT id, email, hashed_password, name, created_at, updated_at
	FROM users
	WHERE email = $1;
	`
	err := DB.Get(&u, query, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		log.Error().Msg("failed to get user by email")
		return nil, err
	}
	return &u, nil
}

// fetches a user by ID. Returns nil, sql.ErrNoRows if not found.
func GetUserByID(id int) (*model.User, error) {
	var u model.User
	query := `
	SELECT id, email, hashed_password, name, created_at, updated_at
	FROM users
	WHERE id = $1;
	`
	err := DB.Get(&u, query, id)
	if err != nil {
		log.Error().Msg("failed to get user by id")
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}
	return &u, nil
}

// updates a user's email and name, and bumps updated_at.
// returns an error if no rows were affected (e.g. user ID doesn’t exist).
func UpdateUserProfile(id int, email string, name *string) error {
	query := `
	UPDATE users
	SET email = $2,
	name = $3,
	updated_at = now()
	WHERE id = $1;
	`
	res, err := DB.Exec(query, id, email, name)
	if err != nil {
		log.Error().Msg("failed to update user profile - exec")
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		log.Error().Msg("failed to update user profile - rows affected")
		{
			return err
		}
	}
	if rows == 0 {
		log.Error().Msg("failed to update user profile - no such user")
		return errors.New("no such user")
	}
	return nil
}

