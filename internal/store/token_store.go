package store

import (
	"database/sql"

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
		INSERT INTO refresh_token (id, user_id)
		VALUES ($1, $2)
		ON CONFLICT (user_id) DO UPDATE SET id = EXCLUDED.id, updated_at = now();
	`

	_, err = tokenStore.DB.Exec(query, refreshTokenID, userID)
	if err != nil {
		return err
	}

	return nil
}
