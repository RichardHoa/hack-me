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

-- User 1111 bookmarks challenge 1 and challenge_response 2
INSERT INTO user_bookmark (user_id, challenge_id)
VALUES ('d45aef9f-3a34-45f1-a55c-7c1d668aa8d0', 1);

INSERT INTO user_bookmark (user_id, challenge_response_id)
VALUES ('d45aef9f-3a34-45f1-a55c-7c1d668aa8d0', 2);

-- User 2222 bookmarks challenge 2
INSERT INTO user_bookmark (user_id, challenge_id)
VALUES ('2291cd29-982c-45a4-91e1-a243020b7ce2', 2);

-- User 3333 bookmarks two responses
INSERT INTO user_bookmark (user_id, challenge_response_id)
VALUES 
('18f89717-b49e-4fa2-832c-a506e4de4cd9', 3),
('18f89717-b49e-4fa2-832c-a506e4de4cd9', 4);
-- +goose StatementEnd


-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS user_bookmark;
-- +goose StatementEnd
