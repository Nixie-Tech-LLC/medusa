package model

import "time"

type Schedule struct {
	ID        int       `db:"id" json:"id"`
	Name      string    `db:"name" json:"name"`
	CreatedBy int       `db:"created_by" json:"created_by"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

type ScheduleWindow struct {
	ID          int        `db:"id" json:"id"`
	ScheduleID  int        `db:"schedule_id" json:"schedule_id"`
	PlaylistID  int        `db:"playlist_id" json:"playlist_id"`
	Start       time.Time  `db:"start_ts" json:"start"`
	End         time.Time  `db:"end_ts" json:"end"`
	Recurrence  string     `db:"recurrence" json:"recurrence"`
	RecurUntil  *time.Time `db:"recur_until" json:"recur_until"`
	Priority    int        `db:"priority" json:"priority"`
	Enabled     bool       `db:"enabled" json:"enabled"`
	CreatedAt   time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at" json:"updated_at"`
}

type ScheduleOccurrence struct {
	WindowID  int       `json:"window_id"`
	Start     time.Time `json:"start"`
	End       time.Time `json:"end"`
	Playlist  int       `json:"playlist_id"`
	Priority  int       `json:"priority"`
	Recurring bool      `json:"recurring"`
}

