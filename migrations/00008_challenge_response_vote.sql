-- +goose Up
-- +goose StatementBegin
CREATE TABLE challenge_response_votes (
    user_id UUID NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    challenge_response_id INT REFERENCES challenge_response(id) ON DELETE CASCADE,
    vote_type SMALLINT NOT NULL CHECK (vote_type IN (1, -1)),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, challenge_response_id)
);
-- +goose StatementEnd

COMMENT ON COLUMN challenge_response_votes.user_id IS '(confidentiality, low), (integrity, high), (availability, high), internal';
COMMENT ON COLUMN challenge_response_votes.challenge_response_id IS '(confidentiality, n/a), (integrity, high), (availability, high), internal';
COMMENT ON COLUMN challenge_response_votes.vote_type IS '(confidentiality, n/a), (integrity, high), (availability, high), public';

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS challenge_response_votes;
-- +goose StatementEnd
