-- +goose Up
-- +goose StatementBegin

-- Create ENUM type for category
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'challenge_category') THEN
        CREATE TYPE challenge_category AS ENUM (
            'web hacking',
            'embedded hacking',
            'reverse engineering',
            'crypto challenge',
            'forensics'
        );
    END IF;
END$$;

-- Create challenge table
CREATE TABLE IF NOT EXISTS challenge (
    id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    -- do not allow white space in name
    name TEXT NOT NULL UNIQUE CHECK(length(trim(name)) >= 3 AND trim(name) = name),
    content TEXT NOT NULL,
    category challenge_category NOT NULL,
    popular_score INT NOT NULL DEFAULT 0,
    user_id UUID NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- make sure name is case-insensitive unique
CREATE INDEX IF NOT EXISTS idx_challenge_name ON challenge(name);
CREATE INDEX IF NOT EXISTS idx_challenge_category ON challenge(category);
CREATE INDEX IF NOT EXISTS idx_challenge_popular_score ON challenge(popular_score);
CREATE INDEX IF NOT EXISTS idx_challenge_name_lower ON challenge(LOWER(name));
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS challenge;
DROP TYPE IF EXISTS challenge_category;
-- +goose StatementEnd

