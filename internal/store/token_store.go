package store

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type DBTokenStore struct {
	DB *sql.DB
}

func NewTokenStore(db *sql.DB) DBTokenStore {
	return DBTokenStore{
		DB: db,
	}
}

type TokenStore interface {
	AddRefreshToken(refreshToken string, userID string) error
}

func (tokenStore *DBTokenStore) AddRefreshToken(refreshToken string, userID string) error {
	query := `
		INSERT INTO refresh_token (id, user_id, token, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id) DO UPDATE SET token = EXCLUDED.token, updated_at = now();
	`
	_, err := tokenStore.DB.Exec(query, uuid.New(), userID, refreshToken, time.Now(), time.Now())
	if err != nil {
		return err
	}

	return nil
}
