-- Drop scheduling helpers first (dependencies)
DROP FUNCTION IF EXISTS schedule_has_overlap(BIGINT,BIGINT,TSTZRANGE,recurrence_kind,TIMESTAMPTZ);
DROP FUNCTION IF EXISTS schedule_window_occurrences(BIGINT,TIMESTAMPTZ,TIMESTAMPTZ);
DROP TABLE IF EXISTS schedule_window_exceptions;
DROP TABLE IF EXISTS schedule_windows;
DROP TABLE IF EXISTS schedule_screens;
DROP TABLE IF EXISTS schedules;
DROP TYPE  IF EXISTS recurrence_kind;

-- Legacy/table set (safe to keep IF EXISTS even if not present)
DROP TABLE IF EXISTS screen_playlists;
DROP TABLE IF EXISTS playlist_items;
DROP TABLE IF EXISTS playlists;
DROP TABLE IF EXISTS content;
DROP TABLE IF EXISTS screen_assignments;
DROP TABLE IF EXISTS screens;
DROP TABLE IF EXISTS users;

-- If you had this historically, keep it safe to drop:
DROP TABLE IF EXISTS screen_contents;

-- (Optional) You usually don't drop extensions in down for a baseline, but safe:
DROP EXTENSION IF EXISTS citext;
DROP EXTENSION IF EXISTS btree_gist;

