CREATE TABLE IF NOT EXISTS screens (
  id SERIAL PRIMARY KEY,
  name TEXT NOT NULL,
  location TEXT,
  paired BOOLEAN NOT NULL DEFAULT false,
  pairing_code TEXT,
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

CREATE TABLE IF NOT EXISTS users (
  id SERIAL PRIMARY KEY,
  email TEXT NOT NULL UNIQUE,
  hashed_password TEXT NOT NULL,
  name TEXT,
  created_at TIMESTAMP NOT NULL DEFAULT now(),
  updated_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS screen_assignments (
  screen_id INT REFERENCES screens(id),
  user_id INT REFERENCES users(id),
  PRIMARY KEY (screen_id, user_id)
);

