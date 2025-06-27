-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS user_report (
    id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    
    user_id UUID NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    
    reason TEXT NOT NULL,
    
    challenge_response_id INT REFERENCES challenge_response(id) ON DELETE CASCADE,
    challenge_id INT REFERENCES challenge(id) ON DELETE CASCADE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    -- Prevent duplicate reports by same user
    UNIQUE (user_id, challenge_response_id),
    UNIQUE (user_id, challenge_id)
);

-- Report on a challenge
INSERT INTO user_report (user_id, reason, challenge_id)
VALUES 
('d45aef9f-3a34-45f1-a55c-7c1d668aa8d0', 'This challenge contains inappropriate content.', 1),
('2291cd29-982c-45a4-91e1-a243020b7ce2', 'Challenge seems plagiarized.', 2);

-- Report on a challenge_response
INSERT INTO user_report (user_id, reason, challenge_response_id)
VALUES 
('d45aef9f-3a34-45f1-a55c-7c1d668aa8d0', 'This response is offensive.', 2),
('18f89717-b49e-4fa2-832c-a506e4de4cd9', 'Spam content.', 3);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS user_report;
-- +goose StatementEnd
