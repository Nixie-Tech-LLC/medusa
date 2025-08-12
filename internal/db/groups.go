package db

import (
	"database/sql"
	"fmt"

	"github.com/Nixie-Tech-LLC/medusa/internal/model"
	_ "github.com/lib/pq"

)

func userOwnsGroup(userID, groupID int) error {
	var exists int
	if err := DB.Get(&exists, `SELECT 1 FROM screen_groups WHERE id=$1 AND created_by=$2`, groupID, userID); err != nil {
		if err == sql.ErrNoRows {
			return sql.ErrNoRows
		}
		return err
	}
	return nil
}

func userOwnsScreen(userID, screenID int) error {
	var exists int
	if err := DB.Get(&exists, `SELECT 1 FROM screens WHERE id=$1 AND created_by=$2`, screenID, userID); err != nil {
		if err == sql.ErrNoRows {
			return sql.ErrNoRows
		}
		return err
	}
	return nil
}

func CreateScreenGroup(userID int, name, description *string) (model.ScreenGroup, error) {
	var g model.ScreenGroup
	if name == nil || *name == "" {
		return g, fmt.Errorf("group name is required")
	}
	err := DB.Get(&g, `
		INSERT INTO screen_groups (name, description, created_by)
		VALUES ($1, $2, $3)
		RETURNING id, name, description, created_by, created_at, updated_at
	`, *name, description, userID)
	return g, err
}

func RenameScreenGroup(userID, groupID int, newName, newDescription *string) (model.ScreenGroup, error) {
	var g model.ScreenGroup

	// Ensure the caller owns this group
	if err := userOwnsGroup(userID, groupID); err != nil {
		return g, err
	}

	err := DB.Get(&g, `
		UPDATE screen_groups
		   SET name        = COALESCE($1, name),
		       description = $2,
		       updated_at  = now()
		 WHERE id = $3 AND created_by = $4
		RETURNING id, name, description, created_by, created_at, updated_at
	`, newName, newDescription, groupID, userID)
	if err == sql.ErrNoRows {
		// Shouldn’t normally happen due to ownership check, but preserve semantics
		return g, sql.ErrNoRows
	}
	return g, err
}

func DeleteScreenGroup(userID, groupID int) error {
	res, err := DB.Exec(`DELETE FROM screen_groups WHERE id=$1 AND created_by=$2`, groupID, userID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func GetScreenGroupByID(groupID int) (model.ScreenGroup, error) {
	var g model.ScreenGroup
	err := DB.Get(&g, `
		SELECT id, name, description, created_by, created_at, updated_at
		  FROM screen_groups
		 WHERE id = $1
	`, groupID)
	return g, err
}

func ListScreenGroups(userID int) ([]model.ScreenGroup, error) {
	var groups []model.ScreenGroup
	err := DB.Select(&groups, `
		SELECT id, name, description, created_by, created_at, updated_at
		  FROM screen_groups
		 WHERE created_by = $1
		 ORDER BY name ASC, id ASC
	`, userID)
	return groups, err
}

func AddScreenToGroup(userID, groupID, screenID int) error {
	// Ownership checks
	if err := userOwnsGroup(userID, groupID); err != nil {
		return err
	}
	if err := userOwnsScreen(userID, screenID); err != nil {
		return err
	}

	// Insert membership (idempotent)
	_, err := DB.Exec(`
		INSERT INTO screen_group_members (group_id, screen_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`, groupID, screenID)
	return err
}

func RemoveScreenFromGroup(userID, groupID, screenID int) error {
	// Ownership checks
	if err := userOwnsGroup(userID, groupID); err != nil {
		return err
	}
	if err := userOwnsScreen(userID, screenID); err != nil {
		return err
	}

	res, err := DB.Exec(`
		DELETE FROM screen_group_members
		 WHERE group_id=$1 AND screen_id=$2
	`, groupID, screenID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		// Treat as not found to keep API semantics tight
		return sql.ErrNoRows
	}
	return nil
}

func ListScreensInGroup(userID, groupID int) ([]model.Screen, error) {
	// Ensure the caller owns the group
	if err := userOwnsGroup(userID, groupID); err != nil {
		return nil, err
	}

	var screens []model.Screen
	err := DB.Select(&screens, `
		SELECT s.id, s.device_id, s.client_information, s.client_width, s.client_height,
		       s.name, s.location, s.paired, s.created_by, s.created_at, s.updated_at
		  FROM screen_group_members m
		  JOIN screens s ON s.id = m.screen_id
		 WHERE m.group_id = $1
		   AND s.created_by = $2
		 ORDER BY s.name ASC, s.id ASC
	`, groupID, userID)
	return screens, err
}

func  ListGroupsForScreen(userID, screenID int) ([]model.ScreenGroup, error) {
	// Ensure the caller owns the screen
	if err := userOwnsScreen(userID, screenID); err != nil {
		return nil, err
	}

	var groups []model.ScreenGroup
	err := DB.Select(&groups, `
		SELECT g.id, g.name, g.description, g.created_by, g.created_at, g.updated_at
		  FROM screen_group_members m
		  JOIN screen_groups g ON g.id = m.group_id
		 WHERE m.screen_id = $1
		   AND g.created_by = $2
		 ORDER BY g.name ASC, g.id ASC
	`, screenID, userID)
	return groups, err
}

