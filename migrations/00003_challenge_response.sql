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

INSERT INTO challenge_response (challenge_id, user_id, name, content) 
VALUES 
    (1, 'd45aef9f-3a34-45f1-a55c-7c1d668aa8d0', 'User 1111 - Challenge 1', 'Response from user 1111 to challenge 1'),
    (2, 'd45aef9f-3a34-45f1-a55c-7c1d668aa8d0', 'User 1111 - Challenge 2', 'Response from user 1111 to challenge 2'),
    (3, 'd45aef9f-3a34-45f1-a55c-7c1d668aa8d0', 'User 1111 - Challenge 3', 'Response from user 1111 to challenge 3'),

    (1, '2291cd29-982c-45a4-91e1-a243020b7ce2', 'User 2222 - Challenge 1', 'Response from user 2222 to challenge 1'),
    (2, '2291cd29-982c-45a4-91e1-a243020b7ce2', 'User 2222 - Challenge 2', 'Response from user 2222 to challenge 2'),
    (3, '2291cd29-982c-45a4-91e1-a243020b7ce2', 'User 2222 - Challenge 3', 'Response from user 2222 to challenge 3'),

    (1, '18f89717-b49e-4fa2-832c-a506e4de4cd9', 'User 3333 - Challenge 1', 'Response from user 3333 to challenge 1'),
    (2, '18f89717-b49e-4fa2-832c-a506e4de4cd9', 'User 3333 - Challenge 2', 'Response from user 3333 to challenge 2'),
    (3, '18f89717-b49e-4fa2-832c-a506e4de4cd9', 'User 3333 - Challenge 3', 'Response from user 3333 to challenge 3');
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS challenge_response;
-- +goose StatementEnd

