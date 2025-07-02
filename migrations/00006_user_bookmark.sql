-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS user_bookmark (
    id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,

    user_id UUID NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    challenge_id INT REFERENCES challenge(id) ON DELETE CASCADE,
    challenge_response_id INT REFERENCES challenge_response(id) ON DELETE CASCADE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    -- Prevent duplicate bookmarks
    UNIQUE (user_id, challenge_id),
    UNIQUE (user_id, challenge_response_id)
);
-- +goose StatementEnd


-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS user_bookmark;
-- +goose StatementEnd
