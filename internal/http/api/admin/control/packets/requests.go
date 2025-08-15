package packets

import (
	"encoding/json"
	"time"
)

// CreateContentRequest Request for creating new content; optional ScreenID to immediately show.
type CreateContentRequest struct {
	Name     string `json:"name"  binding:"required"`
	Type     string `json:"type"  binding:"required"`
	URL      string `json:"url"   binding:"required,url"`
	ScreenID *int   `json:"screen_id"`
}

type CreateScreenRequest struct {
	Name     string  `json:"name" binding:"required"`
	Location *string `json:"location"`
}

type UpdateScreenRequest struct {
	Name     *string `json:"name"`
	Location *string `json:"location"`
}

type AssignScreenRequest struct {
	UserID int `json:"user_id" binding:"required"`
}

type AssignContentToScreenRequest struct {
	ContentID int `json:"content_id" binding:"required"`
}

type PairScreenRequest struct {
	PairingCode string `json:"code" binding:"required"`
	ScreenID    int    `json:"screen_id" binding:"required"`
}

type UpdateContentRequest struct {
	Name   *string `json:"name"`
	Type   *string `json:"type"`
	URL    *string `json:"url"`
	Width  int     `json:"width"`
	Height int     `json:"height"`
}

type CreatePlaylistRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

type UpdatePlaylistRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
}

type AddPlaylistItemRequest struct {
	ContentID int `json:"content_id" binding:"required"`
	Position  int `json:"position"`
	Duration  int `json:"duration" binding:"required"` // seconds; required for playlist items
}

type UpdatePlaylistItemRequest struct {
	Position *int `json:"position"`
	Duration *int `json:"duration"`
}

type AssignPlaylistToScreenRequest struct {
	PlaylistID int `json:"playlist_id" binding:"required"`
}

type AddIntegrationRequest struct {
	IntegrationName string          `json:"integration_name" binding:"required"`
	Duration        *int            `json:"duration"`
	Position        *int            `json:"position"`
	Config          json.RawMessage `json:"config"`
}

type ReorderItemsRequest struct {
	ItemIDs []int `json:"item_ids" binding:"required"`
}

type CreateScheduleRequest struct {
	Name string `json:"name" binding:"required"`
}

type AssignScheduleRequest struct {
	ScreenID int `json:"screen_id" binding:"required"`
}

type CreateWindowRequest struct {
	PlaylistID int        `json:"playlist_id" binding:"required"`
	Start      time.Time  `json:"start" binding:"required"` // RFC3339
	End        time.Time  `json:"end" binding:"required"`
	Recurrence string     `json:"recurrence" binding:"required,oneof=none daily weekly monthly"`
	RecurUntil *time.Time `json:"recur_until,omitempty"` // required if recurrence != none
	Priority   int        `json:"priority,omitempty"`
}

type DeleteWindowRequest struct {
	Scope      string     `json:"scope" binding:"required,oneof=one all"`
	OccurStart *time.Time `json:"occur_start,omitempty"` // required when scope=one
}

type ListOccurrencesQuery struct {
	From time.Time `form:"from" binding:"required"`
	To   time.Time `form:"to" binding:"required"`
}

type CreateScreenGroupRequest struct {
	Name        string  `json:"name" binding:"required"`
	Description *string `json:"description"`
}

type RenameScreenGroupRequest struct {
	Name        *string `json:"name"`        // optional
	Description *string `json:"description"` // optional
}

type ModifyGroupMembershipRequest struct {
	ScreenID int `json:"screen_id" binding:"required"`
}
