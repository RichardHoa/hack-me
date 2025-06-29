package store

import (
	"database/sql"
	"time"

	"github.com/RichardHoa/hack-me/internal/utils"
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

	result, err := utils.ExtractClaimsFromJWT(refreshToken, []string{"refreshID"})
	if err != nil {
		return err
	}

	refreshTokenID := result[0]

	query := `
		INSERT INTO refresh_token (id, user_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id) DO UPDATE SET id = EXCLUDED.id, updated_at = now();
	`

	_, err = tokenStore.DB.Exec(query, refreshTokenID, userID, time.Now(), time.Now())
	if err != nil {
		return err
	}

	return nil
}
