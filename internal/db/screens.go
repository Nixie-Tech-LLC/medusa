package db

import (
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	_ "github.com/lib/pq"

	"github.com/Nixie-Tech-LLC/medusa/internal/model"
)

func GetScreenByID(id int) (model.Screen, error) {
	var screen model.Screen
	err := DB.Get(&screen, `
		SELECT id, device_id, client_information, client_width, client_height, name, location, paired, created_by, created_at, updated_at
		FROM screens
		WHERE id = $1
		`, id)
	if err != nil {
		log.Error().Err(err).Int("id", id).Msg("failed to get screen by id")
	}
	return screen, err
}

func GetScreenByDeviceID(deviceID *string) (model.Screen, error) {
	var screen model.Screen
	err := DB.Get(&screen, `
		SELECT id, device_id, client_information, client_width, client_height, name, location, paired, created_by, created_at, updated_at
		FROM screens
		WHERE device_id = $1
		`, deviceID)
	if err != nil {
		log.Error().Err(err).Msg("failed to get screen by device id")
	}
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
		log.Info().Str("device_id", deref(deviceID)).Msg("No rows found when checking if device is paired")
		return false, nil
	}
	return isPaired, err
}

func ListScreens() ([]model.Screen, error) {
	var screens []model.Screen
	err := DB.Select(&screens, `
		SELECT id, device_id, client_information, client_width, client_height, name, location, paired, created_by, created_at, updated_at
		FROM screens
		ORDER BY id
		`)
	if err != nil {
		log.Error().Err(err).Msg("failed to list screens")
	}
	return screens, err
}

// CreateScreen now generates a UUID for device_id to satisfy NOT NULL + UNIQUE.
func CreateScreen(name string, location *string, createdBy int) (model.Screen, error) {
    var s model.Screen
    deviceID := uuid.NewString()

    q := `
    INSERT INTO screens (device_id, name, location, paired, created_by, created_at, updated_at)
    VALUES ($1, $2, $3, false, $4, now(), now())
    RETURNING id, device_id, client_information, client_width, client_height,
              name, location, paired, created_by, created_at, updated_at;
    `
    if err := DB.Get(&s, q, deviceID, name, location, createdBy); err != nil {
        log.Error().Err(err).Str("device_id", deviceID).Msg("failed to create screen")
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
	if err != nil {
		log.Error().Err(err).Int("id", id).Msg("failed to update screen")
	}
	return err
}

func PairScreen(id int) error {
	_, err := DB.Exec(`
		UPDATE screens
		   SET paired = TRUE,
		       updated_at = now()
		 WHERE id = $1
	`, id)
	if err != nil {
		log.Error().Err(err).Int("id", id).Msg("failed to pair screen")
	}
	return err
}

func AssignDeviceIDToScreen(screenID int, deviceID *string) error {
	_, err := DB.Exec(`
		UPDATE screens
		   SET device_id = COALESCE($2, device_id),
		       updated_at = now()
		 WHERE id = $1
	`, screenID, deviceID)
	if err != nil {
		log.Error().Err(err).Int("screen_id", screenID).Str("device_id", deref(deviceID)).Msg("failed to assign device ID to screen")
	}
	return err
}

func UpdateClientInformation(screenID int, clientInformation *string) error {
	_, err := DB.Exec(`
		UPDATE screens
		SET client_information = $2,
		updated_at = now()
		WHERE id = $1
		`, screenID, clientInformation)
	if err != nil {
		log.Error().Msg("failed to update client information")
	}
	return err
}

func UpdateClientDimensions(screenID int, width, height int) error {
	_, err := DB.Exec(`
		UPDATE screens
		SET client_width = $2,
		client_height = $3,
		updated_at = now()
		WHERE id = $1
		`, screenID, width, height)
	if err != nil {
		log.Error().Msg("failed to update client dimensions")
	}
	return err
}

func DeleteScreen(id int) error {
	_, err := DB.Exec(`DELETE FROM screens WHERE id = $1`, id)
	if err != nil {
		log.Error().Err(err).Int("id", id).Msg("failed to delete screen")
	}
	return err
}

func AssignScreenToUser(screenID, userID int) error {
	_, err := DB.Exec(`
		INSERT INTO screen_assignments (screen_id, user_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`, screenID, userID)
	if err != nil {
		log.Error().Err(err).Int("screen_id", screenID).Int("user_id", userID).Msg("failed to assign screen to user")
	}
	return err
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

