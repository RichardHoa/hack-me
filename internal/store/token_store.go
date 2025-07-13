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

type RefreshToken struct {
	ID        string
	CreatedAt time.Time
}

type TokenStore interface {
	AddRefreshToken(refreshToken string, userID string) error
	DeleteRefreshToken(userID string) error
	GetRefreshToken(userID string) (RefreshToken, error)
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

func (tokenStore *DBTokenStore) GetRefreshToken(userID string) (RefreshToken, error) {
	ID := ""
	var CreatedAt time.Time

	query := `
		SELECT id, created_at
		FROM refresh_token
		WHERE user_id = $1;
	`

	err := tokenStore.DB.QueryRow(query, userID).Scan(&ID, &CreatedAt)

	if err != nil {
		return RefreshToken{}, nil
	}

	return RefreshToken{ID: ID, CreatedAt: CreatedAt}, nil
}

func (tokenStore *DBTokenStore) DeleteRefreshToken(userID string) error {
	query := `
		DELETE FROM refresh_token
		WHERE user_id = $1;
	`

	_, err := tokenStore.DB.Exec(query, userID)
	if err != nil {
		return err
	}

	return nil
}
