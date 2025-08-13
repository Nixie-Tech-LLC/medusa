// internal/db/schedules.go
package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/Nixie-Tech-LLC/medusa/internal/model"
	"github.com/rs/zerolog/log"
)

func CreateSchedule(name string, createdBy int) (model.Schedule, error) {
	var s model.Schedule
	const q = `
	INSERT INTO schedules (name, created_by, created_at, updated_at)
	VALUES ($1, $2, now(), now())
	RETURNING id, name, created_by, created_at, updated_at;`
	if err := DB.Get(&s, q, name, createdBy); err != nil {
		log.Error().Err(err).Msg("CreateSchedule failed")
		return model.Schedule{}, err
	}
	return s, nil
}

func DeleteSchedule(scheduleID int) error {
	_, err := DB.Exec(`DELETE FROM schedules WHERE id = $1;`, scheduleID)
	if err != nil {
		log.Error().Err(err).Int("schedule_id", scheduleID).Msg("DeleteSchedule failed")
	}
	return err
}

func ListSchedules(ownerID int) ([]model.Schedule, error) {
	var out []model.Schedule
	const q = `
	SELECT id, name, created_by, created_at, updated_at
	  FROM schedules
	 WHERE created_by = $1
	 ORDER BY id;`
	if err := DB.Select(&out, q, ownerID); err != nil {
		log.Error().Err(err).Msg("ListSchedules failed")
		return nil, err
	}
	return out, nil
}

func GetSchedule(scheduleID int) (model.Schedule, error) {
	var s model.Schedule
	err := DB.Get(&s, `SELECT id, name, created_by, created_at, updated_at FROM schedules WHERE id = $1;`, scheduleID)
	if err != nil {
		log.Error().Err(err).Int("schedule_id", scheduleID).Msg("GetSchedule failed")
	}
	return s, err
}

func AssignScheduleToScreen(scheduleID, screenID int) error {
	_, err := DB.Exec(`
	INSERT INTO schedule_screens (schedule_id, screen_id)
	VALUES ($1,$2)
	ON CONFLICT DO NOTHING;`, scheduleID, screenID)
	if err != nil {
		log.Error().Err(err).Int("schedule_id", scheduleID).Int("screen_id", screenID).Msg("AssignScheduleToScreen failed")
	}
	return err
}

func UnassignScheduleFromScreen(scheduleID, screenID int) error {
	_, err := DB.Exec(`DELETE FROM schedule_screens WHERE schedule_id = $1 AND screen_id = $2;`, scheduleID, screenID)
	if err != nil {
		log.Error().Err(err).Int("schedule_id", scheduleID).Int("screen_id", screenID).Msg("UnassignScheduleFromScreen failed")
	}
	return err
}

type overlapResult struct {
	Msg *string `db:"schedule_has_overlap"`
}

func CreateScheduleWindow(
	scheduleID, playlistID int,
	start, end time.Time,
	recurrence string,
	recurUntil *time.Time,
	priority int,
) (model.ScheduleWindow, error) {
	var ov overlapResult
	var recurUntilVal sql.NullTime
	if recurUntil != nil {
		recurUntilVal = sql.NullTime{Time: *recurUntil, Valid: true}
	}
	// NOTE: alias column to stable key name across drivers
	if err := DB.Get(&ov, `
		SELECT schedule_has_overlap($1,$2,tstzrange($3,$4,'[)')::tstzrange,$5,$6) AS schedule_has_overlap;
	`, scheduleID, playlistID, start, end, recurrence, recurUntilVal); err != nil {
		log.Error().Err(err).Msg("overlap check failed")
		return model.ScheduleWindow{}, err
	}
	if ov.Msg != nil {
		return model.ScheduleWindow{}, fmt.Errorf("time window overlaps: %s", *ov.Msg)
	}

	var w model.ScheduleWindow
	err := DB.Get(&w, `
	INSERT INTO schedule_windows
	  (schedule_id, playlist_id, time_window, recurrence, recur_until, priority, enabled, created_at, updated_at)
	VALUES
	  ($1,$2,tstzrange($3,$4,'[)'),$5,$6,$7,true,now(),now())
	RETURNING
	  id, schedule_id, playlist_id,
	  lower(time_window) AS start_ts,
	  upper(time_window) AS end_ts,
	  recurrence, recur_until, priority, enabled, created_at, updated_at;
	`, scheduleID, playlistID, start, end, recurrence, recurUntilVal, priority)
	if err != nil {
		log.Error().Err(err).Msg("CreateScheduleWindow failed")
		return model.ScheduleWindow{}, err
	}
	return w, nil
}

func DeleteScheduleWindowAll(windowID int) error {
	_, err := DB.Exec(`DELETE FROM schedule_windows WHERE id = $1;`, windowID)
	if err != nil {
		log.Error().Err(err).Int("window_id", windowID).Msg("DeleteScheduleWindowAll failed")
	}
	return err
}

func DeleteScheduleWindowOneOccurrence(windowID int, occurStart time.Time) error {
	_, err := DB.Exec(`
		INSERT INTO schedule_window_exceptions (window_id, occur_start)
		VALUES ($1,$2)
		ON CONFLICT DO NOTHING;`, windowID, occurStart)
	if err != nil {
		log.Error().Err(err).Int("window_id", windowID).Time("occur_start", occurStart).Msg("Delete one occurrence failed")
	}
	return err
}

func ListScheduleOccurrences(scheduleID int, from, to time.Time) ([]model.ScheduleOccurrence, error) {
	type row struct {
		WindowID   int       `db:"window_id"`
		Start      time.Time `db:"occur_start"`
		End        time.Time `db:"occur_end"`
		PlaylistID int       `db:"playlist_id"`
		Priority   int       `db:"priority"`
		Recurrence string    `db:"recurrence"`
	}
	var rows []row
	const q = `
	  WITH w AS (
	    SELECT id AS window_id, playlist_id, time_window, recurrence, recur_until, priority
	      FROM schedule_windows
	     WHERE schedule_id = $1 AND enabled = true
	  )
	  SELECT w.window_id, o.occur_start, o.occur_end, w.playlist_id, w.priority, w.recurrence
	    FROM w
	    CROSS JOIN LATERAL schedule_window_occurrences(w.window_id, $2, $3) AS o
	    ORDER BY o.occur_start, w.priority DESC, w.window_id;
	`
	if err := DB.Select(&rows, q, scheduleID, from, to); err != nil {
		log.Error().Err(err).Int("schedule_id", scheduleID).Msg("ListScheduleOccurrences failed")
		return nil, err
	}
	out := make([]model.ScheduleOccurrence, 0, len(rows))
	for _, r := range rows {
		out = append(out, model.ScheduleOccurrence{
			WindowID:  r.WindowID,
			Start:     r.Start,
			End:       r.End,
			Playlist:  r.PlaylistID,
			Priority:  r.Priority,
			Recurring: r.Recurrence != "none",
		})
	}
	return out, nil
}

func GetScheduleByWindowID(windowID int) (model.Schedule, error) {
	var s model.Schedule
	const q = `
		SELECT sc.id, sc.name, sc.created_by, sc.created_at, sc.updated_at
		  FROM schedule_windows w
		  JOIN schedules sc ON sc.id = w.schedule_id
		 WHERE w.id = $1;
	`
	if err := DB.Get(&s, q, windowID); err != nil {
		log.Error().Err(err).Int("window_id", windowID).Msg("GetScheduleByWindowID failed")
		return model.Schedule{}, err
	}
	return s, nil
}

func ResolvePlaylistForScreenAt(screenID int, at time.Time) (int, error) {
	const q = `
	  WITH w AS (
	    SELECT w.id AS window_id, w.playlist_id, w.priority
	      FROM schedule_windows w
	      JOIN schedule_screens ss ON ss.schedule_id = w.schedule_id
	     WHERE ss.screen_id = $1
	       AND w.enabled = TRUE
	  ),
	  o AS (
	    SELECT w.window_id, w.playlist_id, w.priority,
	           o.occur_start, o.occur_end
	      FROM w
	      CROSS JOIN LATERAL schedule_window_occurrences(
	                     w.window_id,
	                     $2::timestamptz - interval '1 day',
	                     $2::timestamptz + interval '1 day'
	                   ) AS o
	  ),
	  active AS (
	    SELECT *
	      FROM o
	     WHERE o.occur_start <= $2
	       AND $2 <  o.occur_end
	  )
	  SELECT playlist_id
	    FROM active
	   ORDER BY priority DESC, occur_start DESC
	   LIMIT 1;
	`
	var pid int
	err := DB.Get(&pid, q, screenID, at.UTC())
	if err != nil {
		log.Debug().Err(err).Int("screen_id", screenID).Time("at", at.UTC()).
			Msg("ResolvePlaylistForScreenAt: no active schedule window")
		return 0, err
	}
	return pid, nil
}
