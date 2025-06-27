-- +goose Up
-- +goose StatementBegin

-- Create ENUM type for category
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'challenge_category') THEN
        CREATE TYPE challenge_category AS ENUM (
            'web hacking',
            'embedded hacking',
            'reverse engineering',
            'crypto challenge',
            'forensics'
        );
    END IF;
END$$;

-- Create challenge table
CREATE TABLE IF NOT EXISTS challenge (
    id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name TEXT NOT NULL,
    content TEXT NOT NULL,
    category challenge_category NOT NULL,
    popular_score INT NOT NULL DEFAULT 0,
    user_id UUID NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);


-- Insert test data
INSERT INTO challenge (name, content, category, popular_score, user_id)
VALUES
    ('XSS Lab', 'Find and exploit a reflected XSS vulnerability.', 'web hacking', 15, 'd45aef9f-3a34-45f1-a55c-7c1d668aa8d0'),
    ('UART Dumping', 'Extract firmware via UART from a real IoT device.', 'embedded hacking', 23, '2291cd29-982c-45a4-91e1-a243020b7ce2'),
    ('ELF Binary Crackme', 'Reverse this binary to retrieve the flag.', 'reverse engineering', 42, '18f89717-b49e-4fa2-832c-a506e4de4cd9');
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS challenge;
DROP TYPE IF EXISTS challenge_category;
-- +goose StatementEnd

