package db

import (
	"database/sql"
	"errors"
	"github.com/rs/zerolog/log"

	_ "github.com/lib/pq"

	"github.com/Nixie-Tech-LLC/medusa/internal/model"
)

func AssignContentToScreen(screenID, contentID int) error {
	// upsert into screen_contents
	_, err := DB.Exec(`
		INSERT INTO screen_contents (screen_id, content_id, assigned_at)
		VALUES ($1, $2, now())
		ON CONFLICT (screen_id)
		DO UPDATE SET content_id = EXCLUDED.content_id,
		assigned_at = EXCLUDED.assigned_at;
		`, screenID, contentID)
	log.Error().Msg("failed to assign content to screen")
	return err
}

func GetContentForScreen(screenID int) (*model.Content, error) {
	var c model.Content
	err := DB.Get(&c, `
		SELECT c.id, c.name, c.type, c.url, c.created_at
		FROM content c
		JOIN screen_contents sc ON sc.content_id = c.id
		WHERE sc.screen_id = $1
		`, screenID)
	if errors.Is(err, sql.ErrNoRows) {
		log.Error().Err(err).Msg("Failed to get content for screen")
		return nil, sql.ErrNoRows
	}
	return &c, err
}

func CreateContent(
	name, typ, url string,
	defaultDuration int,
	createdBy int,
) (model.Content, error) {
	var c model.Content
	query := `
	INSERT INTO content
	(name, type, url, default_duration, created_by, created_at, updated_at)
	VALUES
	($1,   $2,   $3,  $4,              $5,         now(),     now())
	RETURNING
	id, name, type, url, default_duration, created_by, created_at,  updated_at;`

	if err := DB.Get(&c, query,
		name,
		typ,
		url,
		defaultDuration,
		createdBy,
	); err != nil {
		log.Error().Err(err).Msg("Failed to create content for screen")
		return model.Content{}, err
	}
	return c, nil
}

func GetContentByID(id int) (model.Content, error) {
	var c model.Content
	query := `
	SELECT
	id, name, type, url, default_duration, created_by, created_at, updated_at
	FROM content
	WHERE id = $1;`

	err := DB.Get(&c, query, id)
	if errors.Is(err, sql.ErrNoRows) {

		log.Error().Err(err).Msg("Failed to get content for screen by ID")
		return model.Content{}, sql.ErrNoRows
	}
	return c, err
}

func ListContent() ([]model.Content, error) {
	var all []model.Content
	query := `
	SELECT
	id,
	name,
	type,
	url,
	default_duration,
	created_by,
	created_at,
	updated_at
	FROM content
	ORDER BY id;
	`
	if err := DB.Select(&all, query); err != nil {
		log.Error().Err(err).Msg("Failed to list content for screen")
		return nil, err
	}
	return all, nil
}

func UpdateContent(
	id int,
	name, url *string,
	defaultDuration *int,
) error {
	_, err := DB.Exec(`
		UPDATE content
		SET
		name             = COALESCE($2, name),
		url              = COALESCE($3, url),
		default_duration = COALESCE($4, default_duration),
		updated_at       = now()
		WHERE id = $1;`,
		id, name, url, defaultDuration,
	)
	log.Error().Err(err).Msg("Failed to update content")
	return err
}

func DeleteContent(id int) error {
	_, err := DB.Exec(`DELETE FROM content WHERE id = $1;`, id)
	log.Error().Err(err).Msg("Failed to delete content")
	return err
}
