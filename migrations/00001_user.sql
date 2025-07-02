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
INSERT INTO "user" (id, username, email, image_link, password, google_id, github_id)
VALUES
    ('d45aef9f-3a34-45f1-a55c-7c1d668aa8d0', 'alice', 'alice@example.com', 'https://example.com/avatar1.png', 'hashed_password_1', 'google-111','github-111'),
    ('2291cd29-982c-45a4-91e1-a243020b7ce2', 'bob', 'bob@example.com', 'https://example.com/avatar2.png', 'hashed_password_2', 'google-222','github-222'),
    ('18f89717-b49e-4fa2-832c-a506e4de4cd9', 'charlie', 'charlie@example.com', 'https://example.com/avatar3.png', 'hashed_password_3', 'google-333','github-333');
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS "user";
-- +goose StatementEnd

