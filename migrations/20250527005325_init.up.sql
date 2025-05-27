CREATE TABLE screens (
  id SERIAL PRIMARY KEY,
  name TEXT NOT NULL,
  location TEXT
);

CREATE TABLE schedules (
  id SERIAL PRIMARY KEY,
  screen_id INT REFERENCES screens(id),
  start_time TIMESTAMP NOT NULL,
  end_time TIMESTAMP NOT NULL,
  content_url TEXT NOT NULL
);

