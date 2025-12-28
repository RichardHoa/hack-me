-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS comment (
    id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    parent_id INT REFERENCES comment(id) ON DELETE CASCADE,
    challenge_id INT REFERENCES challenge(id) ON DELETE CASCADE,
    challenge_response_id INT REFERENCES challenge_response(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()

    -- Rule: Must belong to either a challenge or a response, but not both.
    CONSTRAINT comment_check CHECK (
        (challenge_id IS NOT NULL AND challenge_response_id IS NULL) OR
        (challenge_id IS NULL AND challenge_response_id IS NOT NULL)
    )
);

COMMENT ON COLUMN comment.id IS '(confidentiality, n/a), (integrity, low), (availability, high), internal';
COMMENT ON COLUMN comment.parent_id IS '(confidentiality, n/a), (integrity, low), (availability, high), internal';
COMMENT ON COLUMN comment.challenge_id IS '(confidentiality, n/a), (integrity, high), (availability, high), internal';
COMMENT ON COLUMN comment.challenge_response_id IS '(confidentiality, n/a), (integrity, high), (availability, high), internal';
COMMENT ON COLUMN comment.user_id IS '(confidentiality, low), (integrity, high), (availability, high), internal';
COMMENT ON COLUMN comment.content IS '(confidentiality, n/a), (integrity, high), (availability, high), public';

-- +goose StatementEnd


-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS comment;
-- +goose StatementEnd
