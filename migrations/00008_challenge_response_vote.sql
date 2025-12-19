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

COMMENT ON COLUMN challenge_response_votes.user_id IS '(confidentiality, low), (integrity, high), (availability, high), internal';
COMMENT ON COLUMN challenge_response_votes.challenge_response_id IS '(confidentiality, n/a), (integrity, high), (availability, high), internal';
COMMENT ON COLUMN challenge_response_votes.vote_type IS '(confidentiality, n/a), (integrity, high), (availability, high), public';

CREATE OR REPLACE FUNCTION sync_challenge_response_votes()
RETURNS TRIGGER AS $$
BEGIN
    -- 1. Handle NEW votes (INSERT)
    IF (TG_OP = 'INSERT') THEN
        IF NEW.vote_type = 1 THEN
            UPDATE challenge_response SET up_vote = up_vote + 1 WHERE id = NEW.challenge_response_id;
        ELSE
            UPDATE challenge_response SET down_vote = down_vote + 1 WHERE id = NEW.challenge_response_id;
        END IF;

    -- 2. Handle CHANGED votes (UPDATE)
    ELSIF (TG_OP = 'UPDATE') THEN
        IF OLD.vote_type = NEW.vote_type THEN
            RETURN NEW; -- No change in vote type, do nothing
        END IF;

        IF NEW.vote_type = 1 THEN
            -- Changed from -1 to 1
            UPDATE challenge_response 
            SET up_vote = up_vote + 1, down_vote = down_vote - 1 
            WHERE id = NEW.challenge_response_id;
        ELSE
            -- Changed from 1 to -1
            UPDATE challenge_response 
            SET up_vote = up_vote - 1, down_vote = down_vote + 1 
            WHERE id = NEW.challenge_response_id;
        END IF;

    -- 3. Handle REMOVED votes (DELETE)
    ELSIF (TG_OP = 'DELETE') THEN
        IF OLD.vote_type = 1 THEN
            UPDATE challenge_response SET up_vote = up_vote - 1 WHERE id = OLD.challenge_response_id;
        ELSE
            UPDATE challenge_response SET down_vote = down_vote - 1 WHERE id = OLD.challenge_response_id;
        END IF;
    END IF;

    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_sync_votes
AFTER INSERT OR UPDATE OR DELETE ON challenge_response_votes
FOR EACH ROW
EXECUTE FUNCTION sync_challenge_response_votes();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS challenge_response_votes;
-- +goose StatementEnd
