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

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS challenge_response_votes;
-- +goose StatementEnd
