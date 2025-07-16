package utils

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/RichardHoa/hack-me/internal/constants"
	"github.com/golang-jwt/jwt/v5"
)

func generateSecureHexString(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func SendEmptyTokens(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "accessToken",
		Path:     "/",
		Value:    "",
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
		HttpOnly: false,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "csrfToken",
		Path:     "/",
		Value:    "",
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
		HttpOnly: false,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "refreshToken",
		Path:     "/",
		Value:    "",
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	})
}

func SendTokens(w http.ResponseWriter, accessToken, refreshToken, csrfToken string) error {

	http.SetCookie(w, &http.Cookie{
		Name:     "accessToken",
		Path:     "/",
		Value:    accessToken,
		MaxAge:   int(constants.AccessTokenTime.Seconds()),
		HttpOnly: false,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "csrfToken",
		Path:     "/",
		Value:    csrfToken,
		MaxAge:   int(constants.AccessTokenTime.Seconds()),
		HttpOnly: false,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	})

	result, err := ExtractClaimsFromJWT(refreshToken, []string{"iat", "exp"})
	if err != nil {
		return err
	}

	iatStr := result[0]
	expStr := result[1]

	iatUnix, err := strconv.ParseInt(iatStr, 10, 64)
	if err != nil {
		return err
	}

	expUnix, err := strconv.ParseInt(expStr, 10, 64)
	if err != nil {
		return err
	}

	iatTime := time.Unix(iatUnix, 0)
	expTime := time.Unix(expUnix, 0)

	refreshTokenDuration := expTime.Sub(iatTime)
	// fmt.Printf("time left: %v\n", refreshTokenDuration.Seconds())

	http.SetCookie(w, &http.Cookie{
		Name:     "refreshToken",
		Path:     "/",
		Value:    refreshToken,
		MaxAge:   int(refreshTokenDuration.Seconds()),
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	})

	return nil

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

		var strVal string

		switch v := val.(type) {
		case string:
			strVal = v
		case float64:
			strVal = strconv.FormatInt(int64(v), 10)
		case int64:
			strVal = strconv.FormatInt(v, 10)
		case int:
			strVal = strconv.Itoa(v)
		default:
			strVal = fmt.Sprintf("%v", val)
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
	randomValueHex, err := generateSecureHexString(64)
	if err != nil {
		return "", err
	}

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

func CreateTokens(userID, userName string, refreshTokenTime int64) (accessToken string, refreshToken string, err error) {

	if refreshTokenTime == 0 {
		refreshTokenTime = time.Now().Add(constants.RefreshTokenTime).Unix()
	}

	secureID, err := generateSecureHexString(16)
	if err != nil {
		return "", "", err
	}

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
		constants.TokenRefreshID: secureID,
		constants.TokenUserName:  userName,
		constants.TokenUserID:    userID,
		"exp":                    refreshTokenTime,
		"iat":                    time.Now().Unix(),
	}
	refreshTokenObj := jwt.NewWithClaims(jwt.SigningMethodHS512, refreshClaims)
	refreshToken, err = refreshTokenObj.SignedString([]byte(constants.RefreshTokenSecret))
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil

}

func ValidateTokensFromCookiesWithoutAccessToken(r *http.Request, claimsList []string) (result []string, err error) {

	// Refresh token: validate and extract claims
	refreshCookie, err := r.Cookie("refreshToken")
	if err != nil || refreshCookie.Value == "" {
		return []string{}, err
	}

	refreshToken, err := jwt.Parse(refreshCookie.Value, func(token *jwt.Token) (interface{}, error) {
		return []byte(constants.RefreshTokenSecret), nil
	},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS512.Alg()}),
		jwt.WithIssuedAt(),
	)
	if err != nil || !refreshToken.Valid {
		return []string{}, err
	}

	result, err = ExtractClaimsFromJWT(refreshCookie.Value, claimsList)
	if err != nil {
		return []string{}, err
	}

	if len(result) == 0 {
		return []string{}, errors.New("Empty result list")
	}

	return result, nil

}

func ValidateTokensFromCookies(r *http.Request, claimsList []string) (result []string, err error) {
	// Access token: validate structure
	accessCookie, err := r.Cookie("accessToken")
	if err != nil || accessCookie.Value == "" {
		return []string{}, err
	}

	accessToken, err := jwt.Parse(accessCookie.Value, func(token *jwt.Token) (interface{}, error) {
		return []byte(constants.AccessTokenSecret), nil
	},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS512.Alg()}),
		jwt.WithIssuedAt(),
	)
	if err != nil || !accessToken.Valid {
		return []string{}, err
	}

	// Refresh token: validate and extract claims
	refreshCookie, err := r.Cookie("refreshToken")
	if err != nil || refreshCookie.Value == "" {
		return []string{}, err
	}

	refreshToken, err := jwt.Parse(refreshCookie.Value, func(token *jwt.Token) (interface{}, error) {
		return []byte(constants.RefreshTokenSecret), nil
	},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS512.Alg()}),
		jwt.WithIssuedAt(),
	)
	if err != nil || !refreshToken.Valid {
		return []string{}, err
	}

	result, err = ExtractClaimsFromJWT(refreshCookie.Value, claimsList)
	if err != nil {
		return []string{}, err
	}

	if len(result) == 0 {
		return []string{}, errors.New("Empty result list")
	}

	return result, nil

}
