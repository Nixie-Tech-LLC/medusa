package db

import (
	"database/sql"
	"errors"
	"github.com/rs/zerolog/log"
	"time"

	_ "github.com/lib/pq"

	"github.com/Nixie-Tech-LLC/medusa/internal/model"
)

// @ PLAYLIST
func CreatePlaylist(name, description string, createdBy int) (model.Playlist, error) {
	var p model.Playlist
	const q = `
    INSERT INTO playlists (name, description, created_by, created_at, updated_at)
    VALUES ($1, $2, $3, now(), now())
    RETURNING id, name, description, created_by, created_at, updated_at;
    `
	if err := DB.Get(&p, q, name, description, createdBy); err != nil {
		log.Error().Err(err).Msg("[db] CreatePlaylist: failed to insert playlist")
		return model.Playlist{}, err
	}
	// p.Items defaults to nil/empty
	return p, nil
}

func GetPlaylistByID(id int) (model.Playlist, error) {
	p, err := func() (model.Playlist, error) {
		var p model.Playlist
		q := `
		SELECT
		id,
		name,
		description,
		created_by,
		created_at,
		updated_at
		FROM playlists
		WHERE id = $1;`
		if err := DB.Get(&p, q, id); err != nil {

			log.Error().Err(err).Msg("Failed to get playlist by ID")
			return p, err
		}
		return p, nil
	}()
	if err != nil {
		return model.Playlist{}, err
	}

	items, err := ListPlaylistItems(id)
	if err != nil {
		return p, err
	}
	p.Items = items
	return p, nil
}

func ListPlaylists() ([]model.Playlist, error) {
	var out []model.Playlist
	const q = `SELECT id, name, description, created_by, created_at, updated_at FROM playlists ORDER BY id;`
	if err := DB.Select(&out, q); err != nil {
		log.Error().Err(err).Msg("[db] ListPlaylists: failed to select playlists")
		return nil, err
	}

	for i := range out {
		items, err := ListPlaylistItems(out[i].ID)
		if err != nil {
			log.Error().Err(err).Msgf("[db] ListPlaylists: failed to load items for playlist %d", out[i].ID)
			return nil, err
		}
		out[i].Items = items
	}
	return out, nil
}

func UpdatePlaylist(
	id int,
	name, description *string,
) error {
	_, err := DB.Exec(`
		UPDATE playlists
		SET
		name        = COALESCE($2, name),
		description = COALESCE($3, description),
		updated_at  = now()
		WHERE id = $1;`,
		id, name, description,
	)
	if err != nil {
		log.Error().Err(err).Msg("Failed to update playlist")
	}
	return err
}

func DeletePlaylist(id int) error {
	_, err := DB.Exec(`DELETE FROM playlists WHERE id = $1;`, id)
	log.Error().Err(err).Msg("Failed to delete playlist")
	return err
}

func AddItemToPlaylist(
	playlistID, contentID, position, duration int,
) (model.PlaylistItem, error) {
	var it model.PlaylistItem
	query := `
	INSERT INTO playlist_items
	(playlist_id, content_id, position, duration, created_at)
	VALUES
	($1,          $2,         $3,       $4,       now())
	RETURNING
	id, playlist_id, content_id, position, duration, created_at;`

	if err := DB.Get(&it, query,
		playlistID, contentID, position, duration,
	); err != nil {
		log.Error().Err(err).Msg("Failed to add item to playlist")
		return model.PlaylistItem{}, err
	}
	return it, nil
}

// UpdatePlaylistItem updates position/duration of an item.
func UpdatePlaylistItem(
	itemID int,
	position, duration *int,
) error {
	_, err := DB.Exec(`
		UPDATE playlist_items
		SET
		position = COALESCE($2, position),
		duration = COALESCE($3, duration)
		WHERE id = $1;`,
		itemID, position, duration,
	)
	if err != nil {
		log.Error().Err(err).Msg("Failed to update playlistItem")
	}
	return err
}

func RemovePlaylistItem(itemID int) error {
	_, err := DB.Exec(`DELETE FROM playlist_items WHERE id = $1;`, itemID)
	log.Error().Err(err).Msg("Failed to remove playlistItem")
	return err
}

func ListPlaylistItems(playlistID int) ([]model.PlaylistItem, error) {
	var list []model.PlaylistItem
	const query = `
    SELECT
      id, playlist_id, content_id, position, duration, created_at
    FROM playlist_items
    WHERE playlist_id = $1
    ORDER BY position;`

	err := DB.Select(&list, query, playlistID)
	if err != nil {
		log.Error().Err(err).Msg("Failed to list playlistItems")
	}
	return list, err
}

func ReorderPlaylistItems(playlistID int, itemIDs []int) error {
	tx, err := DB.Beginx()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			err := tx.Rollback()
			if err != nil {
				return
			}
		} else {
			err := tx.Commit()
			if err != nil {
				return
			}
		}
	}()

	count := len(itemIDs)
	if _, err = tx.Exec(`
        UPDATE playlist_items
           SET position = position + $1
         WHERE playlist_id = $2;
    `, count, playlistID); err != nil {
		return err
	}

	for idx, itemID := range itemIDs {
		newPos := idx + 1
		if _, err = tx.Exec(`
            UPDATE playlist_items
               SET position = $1
             WHERE id = $2
               AND playlist_id = $3;
        `, newPos, itemID, playlistID); err != nil {
			return err
		}
	}

	return nil
}

func AssignPlaylistToScreen(screenID, playlistID int) error {
	// First deactivate any existing playlist assignments for this screen
	_, err := DB.Exec(`
		UPDATE screen_playlists 
		SET active = false 
		WHERE screen_id = $1 AND active = true;`,
		screenID,
	)
	if err != nil {
		log.Error().Err(err).Msg("Failed to deactivate existing playlist assignments")
		return err
	}

	// Then insert the new assignment
	_, err = DB.Exec(`
		INSERT INTO screen_playlists
		(screen_id, playlist_id, active, assigned_at)
		VALUES
		($1,        $2,          true,    now());`,
		screenID, playlistID,
	)
	if err != nil {
		log.Error().Err(err).Msg("Failed to assign playlist to screen")
	}
	return err
}

func GetPlaylistForScreen(screenID int) (model.Playlist, error) {
	var pid int
	err := DB.Get(&pid, `
		SELECT playlist_id FROM screen_playlists
		WHERE screen_id = $1 AND active = true;`,
		screenID,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {

			log.Error().Err(err).Msg("Failed to get playlist for screen")
			return model.Playlist{}, sql.ErrNoRows
		}
		return model.Playlist{}, err
	}
	return GetPlaylistByID(pid)
}

// GetPlaylistContentForScreen returns playlist name and content URLs/durations for a screen
func GetPlaylistContentForScreen(screenID int) (string, []ContentItem, error) {
	var playlistName string
	if err := DB.Get(&playlistName, `
        SELECT p.name
          FROM screen_playlists sp
          JOIN playlists p ON sp.playlist_id = p.id
         WHERE sp.screen_id = $1
           AND sp.active = true;
    `, screenID); err != nil {
		log.Error().Err(err).Int("screen_id", screenID).Msg("Failed to get playlist name for screen")
		return "", nil, err
	}

	var items []ContentItem
	if err := DB.Select(&items, `
        SELECT
          c.url,
          pi.duration,
          COALESCE(NULLIF(c.type, ''), 'html') AS type
        FROM screen_playlists sp
        JOIN playlist_items   pi ON sp.playlist_id = pi.playlist_id
        JOIN content          c  ON pi.content_id    = c.id
       WHERE sp.screen_id = $1
         AND sp.active    = true
       ORDER BY pi.position;
    `, screenID); err != nil {
		log.Error().Err(err).Int("screen_id", screenID).Msg("Failed to get playlist content for screen")
		return playlistName, nil, err
	}

	return playlistName, items, nil
}

// GetScreensUsingPlaylist returns all screens that have the specified playlist assigned
func GetScreensUsingPlaylist(playlistID int) ([]model.Screen, error) {
	var screens []model.Screen
	err := DB.Select(&screens, `
		SELECT s.id, s.device_id, s.name, s.location, s.paired, s.created_by, s.created_at, s.updated_at
		  FROM screens s
		  JOIN screen_playlists sp ON s.id = sp.screen_id
		 WHERE sp.playlist_id = $1
		   AND sp.active = true;`,
		playlistID,
	)
	if err != nil {
		log.Error().Err(err).Int("playlist_id", playlistID).Msg("Failed to get screens using playlist")
		return nil, err
	}
	return screens, nil
}

// a direct get for content items in a playlist by its ID
func GetPlaylistContentByPlaylistID(playlistID int) (string, []ContentItem, error) {
	var playlistName string
	if err := DB.Get(&playlistName, `
        SELECT p.name
          FROM playlists p
         WHERE p.id = $1;
    `, playlistID); err != nil {
		log.Error().Err(err).Int("playlist_id", playlistID).Msg("GetPlaylistContentByPlaylistID: name query failed")
		return "", nil, err
	}

	var items []ContentItem
	if err := DB.Select(&items, `
        SELECT
          c.url,
          pi.duration,
          COALESCE(NULLIF(c.type, ''), 'html') AS type
        FROM playlist_items   pi
        JOIN content          c  ON pi.content_id = c.id
       WHERE pi.playlist_id = $1
       ORDER BY pi.position;
    `, playlistID); err != nil {
		log.Error().Err(err).Int("playlist_id", playlistID).Msg("GetPlaylistContentByPlaylistID: items query failed")
		return playlistName, nil, err
	}

	return playlistName, items, nil
}

// GetEffectivePlaylistForScreen tries:
//  1. active schedule window at 'now' -> ("schedule")
//  2. direct assignment via screen_playlists -> ("direct")
//
// Returns: (playlist, contentItems, source, error)
func GetEffectivePlaylistForScreen(screenID int, now time.Time) (model.Playlist, []ContentItem, string, error) {
	// SCHEDULE path
	if pid, err := ResolvePlaylistForScreenAt(screenID, now); err == nil && pid != 0 {
		pl, err := GetPlaylistByID(pid)
		if err != nil {
			return model.Playlist{}, nil, "", err
		}
		_, items, err := func() (string, []ContentItem, error) {
			return GetPlaylistContentByPlaylistID(pid)
		}()
		if err != nil {
			return model.Playlist{}, nil, "", err
		}
		return pl, items, "schedule", nil
	}

	// DIRECT path (fallback)
	pl, err := GetPlaylistForScreen(screenID)
	if err != nil {
		// normalize to sql.ErrNoRows so callers can 404
		return model.Playlist{}, nil, "", sql.ErrNoRows
	}
	name, items, err := GetPlaylistContentForScreen(screenID)
	if err != nil {
		return model.Playlist{}, nil, "", err
	}
	// keep Playlist.Name consistent with content name (in case of drift)
	pl.Name = name
	return pl, items, "direct", nil
}
