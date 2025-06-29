package store

import (
	"database/sql"
	"errors"
	"github.com/golang-jwt/jwt/v5"
	"time"
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

	// Parse without verification
	token, _, err := new(jwt.Parser).ParseUnverified(refreshToken, jwt.MapClaims{})
	if err != nil {
		return err
	}

	var refreshTokenID string

	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		if id, exists := claims["refreshID"]; exists {
			if idStr, ok := id.(string); ok {
				refreshTokenID = idStr
			} else {
				return errors.New("refreshID is not a string")
			}
		} else {
			return errors.New("No refreshID field in refresh token")
		}
	} else {
		return errors.New("Invalid claims")
	}

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
