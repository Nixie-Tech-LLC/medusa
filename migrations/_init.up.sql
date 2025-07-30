CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    hashed_password TEXT NOT NULL,
    name TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS screens (
    id SERIAL PRIMARY KEY,
    device_id TEXT UNIQUE,
    name TEXT NOT NULL,
    location TEXT,
    paired BOOLEAN NOT NULL DEFAULT false,
    created_by INT NOT NULL REFERENCES users(id),
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS schedules (
    id SERIAL PRIMARY KEY,
    screen_id INT REFERENCES screens(id),
    start_time TIMESTAMP NOT NULL,
    end_time TIMESTAMP NOT NULL,
    content_url TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS screen_assignments (
    screen_id INT REFERENCES screens(id),
    user_id INT REFERENCES users(id),
    PRIMARY KEY (screen_id, user_id)
);

CREATE TABLE IF NOT EXISTS content (
    id          SERIAL PRIMARY KEY,
    name        TEXT NOT NULL,
    type        TEXT NOT NULL,
    url         TEXT NOT NULL,
    default_duration INT NOT NULL,
    created_by  INT NOT NULL REFERENCES users(id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS screen_contents (
    screen_id INT REFERENCES screens(id) ON DELETE CASCADE,
    content_id INT REFERENCES content(id) ON DELETE CASCADE,
    assigned_at TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (screen_id)
);

-- Playlists
CREATE TABLE IF NOT EXISTS playlists (
    id          SERIAL PRIMARY KEY,
    name        TEXT NOT NULL,
    description TEXT,
    created_by  INT NOT NULL REFERENCES users(id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Items in a playlist
CREATE TABLE IF NOT EXISTS playlist_items (
    id          SERIAL PRIMARY KEY,
    playlist_id INT REFERENCES playlists(id) ON DELETE CASCADE,
    content_id  INT REFERENCES content(id),
    position    INT NOT NULL,            -- ordering
    duration    INT NOT NULL DEFAULT 5,                     -- override (seconds); NULL = use content.default_duration
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uniq_order_per_playlist UNIQUE(playlist_id, position)
);

-- Assign a playlist to a screen
CREATE TABLE IF NOT EXISTS screen_playlists (
    id          SERIAL PRIMARY KEY,
    screen_id   INT REFERENCES screens(id) ON DELETE CASCADE,
    playlist_id INT REFERENCES playlists(id),
    active      BOOL NOT NULL DEFAULT true,
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
