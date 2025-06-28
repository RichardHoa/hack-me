package utils

import (
	"net/http"
	"time"

	"github.com/RichardHoa/hack-me/internal/constants"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func SendTokens(w http.ResponseWriter, accessToken string, refreshToken string) {

	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Path:     "/",
		Value:    accessToken,
		Expires:  time.Now().Add(constants.AccessTokenTime),
		MaxAge:   int(constants.AccessTokenTime.Seconds()),
		HttpOnly: false,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Path:     "/",
		Value:    refreshToken,
		Expires:  time.Now().Add(constants.RefreshTokenTime),
		MaxAge:   int(constants.RefreshTokenTime.Seconds()),
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	})

}

func CreateTokens(userID string) (accessToken string, refreshToken string, err error) {

	// 1. Create Access Token
	accessClaims := jwt.MapClaims{
		"exp": time.Now().Add(constants.AccessTokenTime).Unix(),
		"iat": time.Now().Unix(),
	}
	accessTokenObj := jwt.NewWithClaims(jwt.SigningMethodHS512, accessClaims)
	accessToken, err = accessTokenObj.SignedString([]byte(constants.AccessTokenSecret))
	if err != nil {
		return "", "", err
	}

	// 2. Create Refresh Token
	refreshClaims := jwt.MapClaims{
		"refresh_id": uuid.New().String(),
		"user_id":    userID,
		"exp":        time.Now().Add(constants.RefreshTokenTime).Unix(),
		"iat":        time.Now().Unix(),
	}
	refreshTokenObj := jwt.NewWithClaims(jwt.SigningMethodHS512, refreshClaims)
	refreshToken, err = refreshTokenObj.SignedString([]byte(constants.RefreshTokenSecret))
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil

}

func ValidateTokensFromCookies(r *http.Request) (userID string, refreshTokenID string, err error) {
	// Access token: validate structure
	accessCookie, err := r.Cookie("access_token")
	if err != nil || accessCookie.Value == "" {
		return "", "", err
	}

	accessToken, err := jwt.Parse(accessCookie.Value, func(token *jwt.Token) (interface{}, error) {
		return []byte(constants.AccessTokenSecret), nil
	},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS512.Alg()}),
		jwt.WithIssuedAt(),
	)
	if err != nil || !accessToken.Valid {
		return "", "", err
	}

	// Refresh token: validate and extract claims
	refreshCookie, err := r.Cookie("refresh_token")
	if err != nil || refreshCookie.Value == "" {
		return "", "", err
	}

	refreshToken, err := jwt.Parse(refreshCookie.Value, func(token *jwt.Token) (interface{}, error) {
		return []byte(constants.RefreshTokenSecret), nil
	},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS512.Alg()}),
		jwt.WithIssuedAt(),
	)
	if err != nil || !refreshToken.Valid {
		return "", "", err
	}

	claims, ok := refreshToken.Claims.(jwt.MapClaims)
	if !ok {
		return "", "", err
	}

	userID, ok1 := claims["user_id"].(string)
	refreshTokenID, ok2 := claims["refresh_id"].(string)
	if !ok1 || !ok2 {
		return "", "", err
	}

	return userID, refreshTokenID, nil
}
