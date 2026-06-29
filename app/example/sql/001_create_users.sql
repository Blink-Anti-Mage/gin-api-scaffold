CREATE TABLE IF NOT EXISTS users (
	id text PRIMARY KEY,
	name text NOT NULL,
	email text NOT NULL,
	password_hash text NOT NULL,
	created_at timestamptz NOT NULL DEFAULT now()
);

ALTER TABLE users
	ADD COLUMN IF NOT EXISTS password_hash text;

CREATE UNIQUE INDEX IF NOT EXISTS users_email_lower_unique ON users (lower(email));
CREATE INDEX IF NOT EXISTS users_created_at_id_idx ON users (created_at, id);
