-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS "user" (
    id UUID PRIMARY KEY,
    username TEXT NOT NULL UNIQUE,
    email TEXT NOT NULL UNIQUE,
    image_link TEXT,
    password TEXT,
    google_id TEXT UNIQUE,
    github_id TEXT UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT valid_user_entry CHECK (
      password IS NOT NULL OR
      google_id IS NOT NULL OR
      github_id IS NOT NULL
    )
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS "user";
-- +goose StatementEnd

