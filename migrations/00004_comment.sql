-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS comment (
    id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    
    -- Recursive parent comment
    parent_id INT REFERENCES comment(id) ON DELETE CASCADE,

    -- Associated to either a challenge or a challenge_response
    challenge_id INT REFERENCES challenge(id) ON DELETE CASCADE,
    challenge_response_id INT REFERENCES challenge_response(id) ON DELETE CASCADE,

    -- The actual comment content
    user_id UUID NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
-- Comments on challenge_id = 1
INSERT INTO comment (challenge_id, user_id, content)
VALUES 
(1, 'd45aef9f-3a34-45f1-a55c-7c1d668aa8d0', 'Excited to try this challenge!'),
(1, '2291cd29-982c-45a4-91e1-a243020b7ce2', 'This one looks tough but fun.');

-- Comments on challenge_response_id = 2
INSERT INTO comment (challenge_response_id, user_id, content)
VALUES 
(2, 'd45aef9f-3a34-45f1-a55c-7c1d668aa8d0', 'Great solution! Very clear.'),
(2, '18f89717-b49e-4fa2-832c-a506e4de4cd9', 'I think this could be improved with edge case handling.');

-- Recursive comment thread on challenge_id = 2

-- Level 1
INSERT INTO comment (challenge_id, user_id, content)
VALUES 
(2, '2291cd29-982c-45a4-91e1-a243020b7ce2', 'This challenge reminds me of a similar one last week.')
RETURNING id;

-- Let's say the returned id is 101 (Level 1)
-- Level 2
INSERT INTO comment (challenge_id, parent_id, user_id, content)
VALUES 
(2, 1, '18f89717-b49e-4fa2-832c-a506e4de4cd9', 'Yes, that one was tricky too!')
RETURNING id;

-- Let's say the returned id is 102 (Level 2)
-- Level 3
INSERT INTO comment (challenge_id, parent_id, user_id, content)
VALUES 
(2, 2, 'd45aef9f-3a34-45f1-a55c-7c1d668aa8d0', 'Glad I’m not the only one who struggled!');
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS comment;
-- +goose StatementEnd
