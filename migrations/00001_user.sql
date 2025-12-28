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

COMMENT ON COLUMN "user".id IS '(confidentiality, n/a), (integrity, n/a), (availability, high), internal';
COMMENT ON COLUMN "user".username IS '(confidentiality, n/a), (integrity, low), (availability, high), public';
COMMENT ON COLUMN "user".email IS '(confidentiality, moderate), (integrity, high), (availability, low), internal';
COMMENT ON COLUMN "user".image_link IS '(confidentiality, n/a), (integrity, low), (availability, low), public';
COMMENT ON COLUMN "user".password IS '(confidentiality, high), (integrity, high), (availability, high), restricted';
COMMENT ON COLUMN "user".google_id IS '(confidentiality, low), (integrity, high), (availability, high), restricted';
COMMENT ON COLUMN "user".github_id IS '(confidentiality, low), (integrity, high), (availability, high), restricted';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS "user";
-- +goose StatementEnd

