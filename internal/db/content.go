package db

import (
	"database/sql"
	"errors"
	"github.com/rs/zerolog/log"

	_ "github.com/lib/pq"

	"github.com/Nixie-Tech-LLC/medusa/internal/model"
)

func CreateContent(
	name, typ, url string, resolutionWidth, resolutionHeight,
	createdBy int,
) (model.Content, error) {
	var c model.Content
	query := `
	INSERT INTO content
	(name, type, url, resolution_width, resolution_height, created_by, created_at, updated_at)
	VALUES
	($1,   $2,   $3,  $4, $5, $6,         now(),     now())
	RETURNING
	id, name, type, url, resolution_width, resolution_height, created_by, created_at, updated_at;`

	if err := DB.Get(&c, query,
		name,
		typ,
		url,
		resolutionWidth,
		resolutionHeight,
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
	id, name, type, url, resolution_width, resolution_height, created_by, created_at, updated_at
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
	resolution_width,
	resolution_height,
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
	resolution_width int,
	resolution_height int,
) error {
	_, err := DB.Exec(`
		UPDATE content
		SET
		name       = COALESCE($2, name),
		url        = COALESCE($3, url),
		resolution_width = COALESCE($4, resolution_width),
		resolution_height = COALESCE($5, resolution_height),
		updated_at = now()
		WHERE id = $1;`,
		id, name, url, resolution_width, resolution_height,
	)
	log.Error().Err(err).Msg("Failed to update content")
	return err
}

func DeleteContent(id int) error {
	_, err := DB.Exec(`DELETE FROM content WHERE id = $1;`, id)
	log.Error().Err(err).Msg("Failed to delete content")
	return err
}
