package utils

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/RichardHoa/hack-me/internal/constants"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func SendTokens(w http.ResponseWriter, accessToken, refreshToken, csrfToken string) {

	http.SetCookie(w, &http.Cookie{
		Name:     "accessToken",
		Path:     "/",
		Value:    accessToken,
		Expires:  time.Now().Add(constants.AccessTokenTime),
		MaxAge:   int(constants.AccessTokenTime.Seconds()),
		HttpOnly: false,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "csrfToken",
		Path:     "/",
		Value:    csrfToken,
		Expires:  time.Now().Add(constants.AccessTokenTime),
		MaxAge:   int(constants.AccessTokenTime.Seconds()),
		HttpOnly: false,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "refreshToken",
		Path:     "/",
		Value:    refreshToken,
		Expires:  time.Now().Add(constants.RefreshTokenTime),
		MaxAge:   int(constants.RefreshTokenTime.Seconds()),
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	})

}

func ExtractClaimsFromJWT(tokenStr string, keys []string) ([]string, error) {
	token, _, err := new(jwt.Parser).ParseUnverified(tokenStr, jwt.MapClaims{})
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}

	result := make([]string, len(keys))
	for i, key := range keys {
		val, exists := claims[key]
		if !exists {
			return nil, fmt.Errorf("missing claim: %s", key)
		}
		strVal, ok := val.(string)
		if !ok {
			return nil, fmt.Errorf("claim %s is not a string", key)
		}
		result[i] = strVal
	}

	return result, nil
}

func CheckCSRFToken(csrfToken string, sessionID string) (bool, error) {
	// we use refreshTokenID as sessionID
	parts := strings.Split(csrfToken, ".")
	if len(parts) != 2 {
		return false, errors.New("invalid CSRF token format")
	}

	hmacFromRequest := parts[0]
	randomValue := parts[1]

	message := fmt.Sprintf(
		"%d!%s!%d!%s",
		len(sessionID),
		sessionID,
		len(randomValue),
		randomValue,
	)

	// Generate expected HMAC
	h := hmac.New(sha256.New, []byte(constants.CSRFTokenSecret))
	h.Write([]byte(message))
	expectedHmac := h.Sum(nil)

	hmacRequestBytes, err := hex.DecodeString(hmacFromRequest)
	if err != nil {
		return false, fmt.Errorf("invalid HMAC hex from request: %w", err)
	}

	// Safe compare
	if !hmac.Equal(hmacRequestBytes, expectedHmac) {
		fmt.Printf("CSRF Error: Invalid HMAC for sessionID: %s\n", sessionID)
		return false, nil
	}

	return true, nil
}

func CreateCSRFToken(sessionID string) (string, error) {
	randomBytes := make([]byte, 64)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	randomValueHex := hex.EncodeToString(randomBytes)

	message := fmt.Sprintf(
		"%d!%s!%d!%s",
		len(sessionID),
		sessionID,
		len(randomValueHex),
		randomValueHex,
	)

	// Generate HMAC
	h := hmac.New(sha256.New, []byte(constants.CSRFTokenSecret))
	h.Write([]byte(message))
	hmacSum := h.Sum(nil)
	hmacHex := hex.EncodeToString(hmacSum)

	csrfToken := fmt.Sprintf("%s.%s", hmacHex, randomValueHex)
	return csrfToken, nil
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
		"refreshID": uuid.New().String(),
		"userID":    userID,
		"exp":       time.Now().Add(constants.RefreshTokenTime).Unix(),
		"iat":       time.Now().Unix(),
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
	accessCookie, err := r.Cookie("accessToken")
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
	refreshCookie, err := r.Cookie("refreshToken")
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

	userID, ok1 := claims["userID"].(string)
	refreshTokenID, ok2 := claims["refreshID"].(string)
	if !ok1 || !ok2 {
		return "", "", err
	}

	return userID, refreshTokenID, nil
}
