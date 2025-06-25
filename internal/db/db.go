package db

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

	"github.com/Nixie-Tech-LLC/medusa/internal/model"
)

var (
	DB *sqlx.DB
)

// opens a PostgreSQL connection and assigns it to DB.
func Init(databaseURL string) error {
	var err error
	DB, err = sqlx.Connect("postgres", databaseURL)
	if err != nil {
		return err
	}
	return nil
}

// finds all “*.up.sql” files in migrationsPath (sorted by name)
// and executes their SQL contents in order. It ignores “*.down.sql” files.
// returns that error immediately upon execution failure
func RunMigrations(migrationsPath string) error {
	// find all files matching “*.up.sql”
	pattern := filepath.Join(migrationsPath, "*.up.sql")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("failed to glob migrations: %w", err)
	}
	if len(files) == 0 {
		// nothing to do
		return nil
	}

	// sort file names so that they run in deterministic order
	sort.Strings(files)

	// for each file, read its contents and execute as a single SQL statement
	for _, file := range files {
		sqlBytes, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("could not read migration %q: %w", file, err)
		}
		sqlStmt := string(sqlBytes)
		if sqlStmt == "" {
			continue
		}
		if _, err := DB.Exec(sqlStmt); err != nil {
			return fmt.Errorf("error executing migration %q: %w", file, err)
		}
	}
	return nil
}

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
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("no such user")
	}
	return nil
}

func GetScreenByID(id int) (model.Screen, error) {
	var screen model.Screen
	err := DB.Get(&screen, `
		SELECT id, device_id, name, location, paired, created_at, updated_at
		FROM screens
		WHERE id = $1
		`, id)
	return screen, err
}

func GetScreenByDeviceID(deviceID *string) (model.Screen, error) {
	var screen model.Screen
	err := DB.Get(&screen, `
		SELECT id, device_id, name, location, paired, created_at, updated_at
		FROM screens
		WHERE device_id = $1
		`, deviceID)
	return screen, err
}

func IsScreenPairedByDeviceID(deviceID *string) (bool, error) {
	var isPaired bool
	err := DB.Get(isPaired, `
		SELECT paired
		FROM screens
		WHERE device_id = $1
		`, deviceID)
	return isPaired, err
}

func ListScreens() ([]model.Screen, error) {
	var screens []model.Screen
	err := DB.Select(&screens, `
		SELECT id, device_id, name, location, paired, created_at, updated_at
		FROM screens
		ORDER BY id
		`)
	return screens, err
}

func CreateScreen(name string, location *string, createdBy int) (model.Screen, error) {
	var s model.Screen
	q := `
	INSERT INTO screens (device_id, name, location, paired, created_by, created_at, updated_at)
	VALUES (gen_random_uuid()::text, $1, $2, false, $3, now(), now())
	RETURNING id, device_id, name, location, paired, created_by, created_at, updated_at;`
	if err := DB.Get(&s, q, name, location, createdBy); err != nil {
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
	return err
}

func PairScreen(id int) error {
	_, err := DB.Exec(`
		UPDATE screens
		SET paired = TRUE,
		updated_at = now()
		WHERE id = $1
		`, id)
	return err
}

func AssignDeviceIDToScreen(screenID int, deviceID *string) error {
	_, err := DB.Exec(`
		UPDATE screens
		SET device_id = COALESCE($2, device_id),
		updated_at = now()
		WHERE id = $1
		`, screenID, deviceID)
	return err
}

func DeleteScreen(id int) error {
	_, err := DB.Exec(`DELETE FROM screens WHERE id = $1`, id)
	return err
}

func AssignScreenToUser(screenID, userID int) error {
	_, err := DB.Exec(`
		INSERT INTO screen_assignments (screen_id, user_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
		`, screenID, userID)
	return err
}

func AssignContentToScreen(screenID, contentID int) error {
	// upsert into screen_contents
	_, err := DB.Exec(`
		INSERT INTO screen_contents (screen_id, content_id, assigned_at)
		VALUES ($1, $2, now())
		ON CONFLICT (screen_id)
		DO UPDATE SET content_id = EXCLUDED.content_id,
		assigned_at = EXCLUDED.assigned_at;
		`, screenID, contentID)
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
	($1,   $2,   $3,  $5,              $6,         now(),     now())
	RETURNING
	id, name, type, url, default_duration, created_at, created_by, updated_at;`

	if err := DB.Get(&c, query,
		name,             // $1 → name
		typ,              // $2 → type
		url,              // $3 → url
		defaultDuration,  // $5 → default_duration
		createdBy,        // $6 → created_by
		); err != nil {
		return model.Content{}, err
	}
	return c, nil
}

func GetContentByID(id int) (model.Content, error) {
	var c model.Content
	query := `
	SELECT
	id, name, type, url, default_duration, created_at, updated_at
	FROM content
	WHERE id = $1;`

	err := DB.Get(&c, query, id)
	if errors.Is(err, sql.ErrNoRows) {
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
		default_duration = COALESCE($5, default_duration),
		updated_at       = now()
		WHERE id = $1;`,
		id, name, url, defaultDuration,
		)
	return err
}

func DeleteContent(id int) error {
	_, err := DB.Exec(`DELETE FROM content WHERE id = $1;`, id)
	return err
}

// @ PLAYLIST
func CreatePlaylist(name, description string, createdBy int) (model.Playlist, error) {
	var p model.Playlist
	q := `
	INSERT INTO playlists
	(name, description, created_by, created_at, updated_at)
	VALUES
	($1,   $2,          $3,         now(),      now())
	RETURNING id, name, description, created_by, created_at, updated_at;`
	if err := DB.Get(&p, q, name, description, createdBy); err != nil {
		return model.Playlist{}, err
	}
	return p, nil
}

func GetPlaylistByID(id int) (model.Playlist, error) {
	p, err := func() (model.Playlist, error) {
		var p model.Playlist
		q := `
		SELECT id, name, description, created_at, updated_at
		FROM playlists
		WHERE id = $1;`
		if err := DB.Get(&p, q, id); err != nil {
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
	query := `
	SELECT id, name, description, created_at, updated_at
	FROM playlists
	ORDER BY id;`

	err := DB.Select(&out, query)
	return out, err
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
	return err
}

func DeletePlaylist(id int) error {
	_, err := DB.Exec(`DELETE FROM playlists WHERE id = $1;`, id)
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
	return err
}

func RemovePlaylistItem(itemID int) error {
	_, err := DB.Exec(`DELETE FROM playlist_items WHERE id = $1;`, itemID)
	return err
}

func ListPlaylistItems(playlistID int) ([]model.PlaylistItem, error) {
	var list []model.PlaylistItem
	query := `
	SELECT
	id, playlist_id, content_id, position, duration, created_at
	FROM playlist_items
	WHERE playlist_id = $1
	ORDER BY position;`

	err := DB.Select(&list, query, playlistID)
	return list, err
}

func AssignPlaylistToScreen(screenID, playlistID int) error {
	_, err := DB.Exec(`
		INSERT INTO screen_playlists
		(screen_id, playlist_id, active, assigned_at)
		VALUES
		($1,        $2,          true,    now())
		ON CONFLICT (screen_id)
		DO UPDATE SET
		playlist_id = EXCLUDED.playlist_id,
		active      = true,
		assigned_at = now();`,
		screenID, playlistID,
		)
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
			return model.Playlist{}, sql.ErrNoRows
		}
		return model.Playlist{}, err
	}
	return GetPlaylistByID(pid)
}
