-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS refresh_token (
    id TEXT PRIMARY KEY, 
    user_id UUID NOT NULL UNIQUE REFERENCES "user"(id) ON DELETE CASCADE, 
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

COMMENT ON COLUMN refresh_token.id IS '(confidentiality, n/a), (integrity, low), (availability, high), internal';
COMMENT ON COLUMN refresh_token.user_id IS '(confidentiality, n/a), (integrity, low), (availability, high), internal';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS refresh_token;
-- +goose StatementEnd
