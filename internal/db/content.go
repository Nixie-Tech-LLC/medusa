package db

import (
	"database/sql"
	"errors"
	"strings"
	"fmt"

	"github.com/rs/zerolog/log"
	_ "github.com/lib/pq"

	"github.com/Nixie-Tech-LLC/medusa/internal/model"
)

func normalizeContentType(in string) (string, error) {
	s := strings.ToLower(strings.TrimSpace(in))

	switch {
	case s == "integration":
		return "integration", nil
	case s == "html" || s == "text/html":
		return "html", nil
	case s == "image" || strings.HasPrefix(s, "image/"):
		return "image", nil
	case s == "video" || strings.HasPrefix(s, "video/"):
		return "video", nil
	default:
		return "", fmt.Errorf("unsupported content type %q", in)
	}
}
// CreateContent inserts content. width/height of 0 => NULL (to satisfy CHECK > 0 if not null).
func CreateContent(
	name, typ, url string, resolutionWidth, resolutionHeight,
	createdBy int,
) (model.Content, error) {
	var c model.Content

	// NEW: normalize type to pass CHECK (image|video|html|integration)
	normType, err := normalizeContentType(typ)
	if err != nil {
		log.Error().Err(err).Str("type", typ).Msg("invalid content type")
		return model.Content{}, err
	}

	var wptr, hptr *int
	if resolutionWidth > 0 {
		wptr = &resolutionWidth
	}
	if resolutionHeight > 0 {
		hptr = &resolutionHeight
	}

	const query = `
	INSERT INTO content
	(name, type, url, resolution_width, resolution_height, created_by, created_at, updated_at)
	VALUES
	($1,   $2,   $3,  $4,               $5,                $6,        now(),     now())
	RETURNING
	id, name, type, url, resolution_width, resolution_height, created_by, created_at, updated_at;`

	if err := DB.Get(&c, query,
		name,
		normType, // <- use normalized
		url,
		wptr,
		hptr,
		createdBy,
	); err != nil {
		log.Error().Err(err).Str("name", name).Msg("Failed to create content")
		return model.Content{}, err
	}
	return c, nil
}

func GetContentByID(id int) (model.Content, error) {
	var c model.Content
	const query = `
	SELECT
	  id, name, type, url, resolution_width, resolution_height, created_by, created_at, updated_at
	FROM content
	WHERE id = $1;`
	err := DB.Get(&c, query, id)
	if errors.Is(err, sql.ErrNoRows) {
		log.Error().Err(err).Int("id", id).Msg("Content not found by ID")
		return model.Content{}, sql.ErrNoRows
	}
	return c, err
}

func ListContent() ([]model.Content, error) {
	var all []model.Content
	const query = `
	SELECT
	  id, name, type, url, resolution_width, resolution_height, created_by, created_at, updated_at
	FROM content
	ORDER BY id;`
	if err := DB.Select(&all, query); err != nil {
		log.Error().Err(err).Msg("Failed to list content")
		return nil, err
	}
	return all, nil
}

// UpdateContent: width/height of 0 => leave unchanged (NULL COALESCE).
func UpdateContent(
	id int,
	name, url *string,
	resolutionWidth int,
	resolutionHeight int,
) error {
	var wptr, hptr *int
	if resolutionWidth > 0 {
		wptr = &resolutionWidth
	}
	if resolutionHeight > 0 {
		hptr = &resolutionHeight
	}

	_, err := DB.Exec(`
		UPDATE content
		   SET name              = COALESCE($2, name),
		       url               = COALESCE($3, url),
		       resolution_width  = COALESCE($4, resolution_width),
		       resolution_height = COALESCE($5, resolution_height),
		       updated_at        = now()
		 WHERE id = $1;`,
		id, name, url, wptr, hptr,
	)
	if err != nil {
		log.Error().Err(err).Int("id", id).Msg("Failed to update content")
	}
	return err
}

func DeleteContent(id int) error {
	_, err := DB.Exec(`DELETE FROM content WHERE id = $1;`, id)
	if err != nil {
		log.Error().Err(err).Int("id", id).Msg("Failed to delete content")
	}
	return err
}

