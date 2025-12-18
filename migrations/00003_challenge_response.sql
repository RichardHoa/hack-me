-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS challenge_response (
    id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY, 
    challenge_id INT NOT NULL REFERENCES challenge(id) ON DELETE CASCADE, 
    user_id UUID NOT NULL REFERENCES "user"(id) ON DELETE CASCADE, 
    name TEXT NOT NULL, 
    content TEXT NOT NULL, 
    up_vote INT NOT NULL DEFAULT 0, 
    down_vote INT NOT NULL DEFAULT 0, 
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    -- Enforce uniqueness: a user can respond only once to a specific challenge
    UNIQUE (challenge_id, user_id)
);

COMMENT ON COLUMN challenge_response.id IS '(confidentiality, n/a), (integrity, low), (availability, high), internal';
COMMENT ON COLUMN challenge_response.challenge_id IS '(confidentiality, n/a), (integrity, high), (availability, high), internal';
COMMENT ON COLUMN challenge_response.user_id IS '(confidentiality, n/a), (integrity, high), (availability, high), internal';
COMMENT ON COLUMN challenge_response.name IS '(confidentiality, n/a), (integrity, moderate), (availability, high), public';
COMMENT ON COLUMN challenge_response.content IS '(confidentiality, n/a), (integrity, moderate), (availability, high), public';
COMMENT ON COLUMN challenge_response.up_vote IS '(confidentiality, n/a), (integrity, moderate), (availability, high), public';
COMMENT ON COLUMN challenge_response.down_vote IS '(confidentiality, n/a), (integrity, moderate), (availability, high), public';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS challenge_response;
-- +goose StatementEnd

