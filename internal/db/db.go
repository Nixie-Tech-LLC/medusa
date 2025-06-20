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
		SELECT id, name, location, paired, pairing_code, created_at, updated_at
		FROM screens
		WHERE id = $1
	`, id)
	return screen, err
}

func ListScreens() ([]model.Screen, error) {
	var screens []model.Screen
	err := DB.Select(&screens, `
		SELECT id, name, location, paired, pairing_code, created_at, updated_at
		FROM screens
		ORDER BY id
	`)
	return screens, err
}

func CreateScreen(name string, location *string) (model.Screen, error) {
	var screen model.Screen
	err := DB.Get(&screen, `
		INSERT INTO screens (name, location)
		VALUES ($1, $2)
		RETURNING id, name, location, paired, pairing_code, created_at, updated_at
	`, name, location)
	return screen, err
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

