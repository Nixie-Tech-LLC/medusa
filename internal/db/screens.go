package db

import (
	"database/sql"
	"errors"
	"github.com/rs/zerolog/log"

	_ "github.com/lib/pq"

	"github.com/Nixie-Tech-LLC/medusa/internal/model"
)

func GetScreenByID(id int) (model.Screen, error) {
	var screen model.Screen
	err := DB.Get(&screen, `
		SELECT id, device_id, name, location, paired, created_by, created_at, updated_at
		FROM screens
		WHERE id = $1
		`, id)
	log.Error().Msg("failed to get screen by id")
	return screen, err
}

func GetScreenByDeviceID(deviceID *string) (model.Screen, error) {
	var screen model.Screen
	err := DB.Get(&screen, `
		SELECT id, device_id, name, location, paired, created_at, updated_at
		FROM screens
		WHERE device_id = $1
		`, deviceID)
	log.Error().Msg("failed to get screen by device id")
	return screen, err
}

func IsScreenPairedByDeviceID(deviceID *string) (bool, error) {
	var isPaired bool
	err := DB.Get(&isPaired, `
		SELECT paired
		FROM screens
		WHERE device_id = $1
		`, deviceID)
	if errors.Is(err, sql.ErrNoRows) {
		log.Error().Msg("failed to check if device is paired by device ID")
		return false, nil
	}
	return isPaired, err
}

func ListScreens() ([]model.Screen, error) {
	var screens []model.Screen
	err := DB.Select(&screens, `
		SELECT id, device_id, name, location, paired, created_by, created_at, updated_at
		FROM screens
		ORDER BY id
		`)
	log.Error().Msg("failed to list screens")
	return screens, err
}

func CreateScreen(name string, location *string, createdBy int) (model.Screen, error) {
	var s model.Screen
	q := `
	INSERT INTO screens (name, location, paired, created_by, created_at, updated_at)
	VALUES ($1, $2, false, $3, now(), now())
	RETURNING id, device_id, name, location, paired, created_by, created_at, updated_at;`
	if err := DB.Get(&s, q, name, location, createdBy); err != nil {
		log.Error().Msg("failed to create screen")
		return model.Screen{}, err
	}
	return s, nil
}

func UpdateScreen(id int, name, location *string) error {
	_, err := DB.Exec(`
		UPDATE screens
		SET name = COALESCE($2, name),
		location = COALESCE($3, location),
		updated_at = now()
		WHERE id = $1
		`, id, name, location)
	log.Error().Msg("failed to update screen")
	return err
}

func PairScreen(id int) error {
	_, err := DB.Exec(`
		UPDATE screens
		SET paired = TRUE,
		updated_at = now()
		WHERE id = $1
		`, id)
	log.Error().Msg("failed to pair screen")
	return err
}

func AssignDeviceIDToScreen(screenID int, deviceID *string) error {
	_, err := DB.Exec(`
		UPDATE screens
		SET device_id = COALESCE($2, device_id),
		updated_at = now()
		WHERE id = $1
		`, screenID, deviceID)
	log.Error().Msg("failed to assign device ID to screen")
	return err
}

func DeleteScreen(id int) error {
	_, err := DB.Exec(`DELETE FROM screens WHERE id = $1`, id)
	log.Error().Msg("failed to delete screen")
	return err
}

func AssignScreenToUser(screenID, userID int) error {
	_, err := DB.Exec(`
		INSERT INTO screen_assignments (screen_id, user_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
		`, screenID, userID)
	log.Error().Msg("failed to assign screen to user")
	return err
}

