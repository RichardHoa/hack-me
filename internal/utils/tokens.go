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

/*
generateSecureHexString creates a cryptographically secure,
random hexadecimal string of a specified byte length.
*/
func generateSecureHexString(byteLength int) (string, error) {
	bytes := make([]byte, byteLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

/*
SendEmptyTokens sends cookies to the client with past expiration dates.
Effectively clean the tokens out of browser
*/
func SendEmptyTokens(w http.ResponseWriter) {

	secureAttribute := true
	sameSiteAttribute := http.SameSiteStrictMode

	if constants.IsDevMode {
		secureAttribute = false
		sameSiteAttribute = http.SameSiteLaxMode
	}

	// nosemgrep
	http.SetCookie(w, &http.Cookie{
		Name:     "accessToken",
		Path:     "/",
		Value:    "",
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
		HttpOnly: false,
		Secure:   secureAttribute,
		SameSite: sameSiteAttribute,
	})

	// nosemgrep
	http.SetCookie(w, &http.Cookie{
		Name:     "csrfToken",
		Path:     "/",
		Value:    "",
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
		HttpOnly: false,
		Secure:   secureAttribute,
		SameSite: sameSiteAttribute,
	})

	// nosemgrep
	http.SetCookie(w, &http.Cookie{
		Name:     "refreshToken",
		Path:     "/",
		Value:    "",
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Secure:   secureAttribute,
		SameSite: sameSiteAttribute,
	})
}

/*
SendTokens sets the access, refresh, and CSRF tokens as cookies in the HTTP
response, dynamically calculates refreshToken MaxAge from the token's expiration claim.
*/
func SendTokens(w http.ResponseWriter, accessToken, refreshToken, csrfToken string) error {

	secureAttribute := true
	sameSiteAttribute := http.SameSiteStrictMode

	if constants.IsDevMode {
		secureAttribute = false
		sameSiteAttribute = http.SameSiteLaxMode
	}

	// nosemgrep
	http.SetCookie(w, &http.Cookie{
		Name:     "accessToken",
		Path:     "/",
		Value:    accessToken,
		MaxAge:   int(constants.AccessTokenTime.Seconds()),
		HttpOnly: true,
		Secure:   secureAttribute,
		SameSite: sameSiteAttribute,
	})

	// nosemgrep
	http.SetCookie(w, &http.Cookie{
		Name:     "csrfToken",
		Path:     "/",
		Value:    csrfToken,
		MaxAge:   int(constants.AccessTokenTime.Seconds()),
		HttpOnly: true,
		Secure:   secureAttribute,
		SameSite: sameSiteAttribute,
	})

	//NOTE: HttpOnly of both csrfToken and accessToken are true because we are using Sveltekit, an SSR framework

	refreshTokenResult, err := ExtractClaimsFromJWT(refreshToken, []string{"iat", "exp"})
	if err != nil {
		return err
	}

	iatStr := refreshTokenResult[0]
	expStr := refreshTokenResult[1]

	iatUnix, err := strconv.ParseInt(iatStr, 10, 64)
	if err != nil {
		return err
	}

	expUnix, err := strconv.ParseInt(expStr, 10, 64)
	if err != nil {
		return err
	}

	issuedAtTime := time.Unix(iatUnix, 0)
	expiresTime := time.Unix(expUnix, 0)

	// refresh token rotation
	refreshTokenTime := expiresTime.Sub(issuedAtTime)

	// nosemgrep
	http.SetCookie(w, &http.Cookie{
		Name:     "refreshToken",
		Path:     "/",
		Value:    refreshToken,
		MaxAge:   int(refreshTokenTime.Seconds()),
		HttpOnly: true,
		Secure:   secureAttribute,
		SameSite: sameSiteAttribute,
	})

	return nil

}

/*
ExtractClaimsFromJWT parses a JWT string without verifying its signature and
extracts the values of specified claims. It returns a slice of string,
regardless of the token key type
*/
func ExtractClaimsFromJWT(tokenStr string, keys []string) ([]string, error) {
	token, _, err := new(jwt.Parser).ParseUnverified(tokenStr, jwt.MapClaims{})
	if err != nil {
		return nil, fmt.Errorf("failed to parse token |%s|: %w", tokenStr, err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid jwt claims structure")
	}

	result := make([]string, len(keys))
	for i, key := range keys {
		val, exists := claims[key]
		if !exists {
			return nil, fmt.Errorf("missing required claim: %s", key)
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

// CheckCSRFToken validates a CSRF token using the HMAC-based token pattern.
func CheckCSRFToken(csrfToken string, sessionID string) (bool, error) {
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

	h := hmac.New(sha256.New, []byte(constants.CSRFTokenSecret))
	h.Write([]byte(message))
	expectedHmac := h.Sum(nil)

	hmacRequestBytes, err := hex.DecodeString(hmacFromRequest)
	if err != nil {
		return false, fmt.Errorf("invalid HMAC hex from request: %w", err)
	}

	if !hmac.Equal(hmacRequestBytes, expectedHmac) {
		fmt.Printf("CSRF Error: Invalid HMAC for sessionID: %s\n", sessionID)
		return false, nil
	}

	return true, nil
}

// CreateCSRFToken generates a new CSRF token
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

	h := hmac.New(sha256.New, []byte(constants.CSRFTokenSecret))
	h.Write([]byte(message))
	hmacSum := h.Sum(nil)
	hmacHex := hex.EncodeToString(hmacSum)

	csrfToken := fmt.Sprintf("%s.%s", hmacHex, randomValueHex)
	return csrfToken, nil
}

/*
CreateTokens generates a new pair of signed JWTs for a user: a short-lived
access token and a long-lived refresh token
*/
func CreateTokens(userID, userName string, refreshTokenTime int64) (signedAccessToken string, signedRefreshToken string, err error) {

	if refreshTokenTime == 0 {
		refreshTokenTime = time.Now().Add(constants.RefreshTokenTime).Unix()
	}

	accessClaims := jwt.MapClaims{
		"exp": time.Now().Add(constants.AccessTokenTime).Unix(),
		"iat": time.Now().Unix(),
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS512, accessClaims)
	signedAccessToken, err = accessToken.SignedString([]byte(constants.AccessTokenSecret))
	if err != nil {
		return "", "", err
	}

	refreshID, err := generateSecureHexString(32)
	if err != nil {
		return "", "", err
	}

	refreshClaims := jwt.MapClaims{
		constants.JWTRefreshTokenID: refreshID,
		constants.JWTUserName:       userName,
		constants.JWTUserID:         userID,
		"exp":                       refreshTokenTime,
		"iat":                       time.Now().Unix(),
	}
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS512, refreshClaims)
	signedRefreshToken, err = refreshToken.SignedString([]byte(constants.RefreshTokenSecret))
	if err != nil {
		return "", "", err
	}

	return signedAccessToken, signedRefreshToken, nil

}

/*
GetValuesFromCookieWithoutAccessToken validates only the refresh token from the
request's cookies and extracts the specified claims. This function is intended
for use in contexts where the access token is expired or not required, such as
the token refresh endpoint.
*/
func GetValuesFromCookieWithoutAccessToken(r *http.Request, claimsList []string) (result []string, err error) {

	// Refresh token: validate and extract claims
	refreshCookie, err := r.Cookie("refreshToken")
	if err != nil || refreshCookie.Value == "" {
		return []string{}, fmt.Errorf("refreshToken is not present")
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

/*
GetValuesFromCookie validates both the access and refresh tokens from the
request's cookies. If both tokens are valid, it extracts and returns the
specified claims from the refresh token. This is used for standard
authenticated endpoints.
*/
func GetValuesFromCookie(r *http.Request, claimsList []string) (result []string, err error) {
	// Access token: validate structure
	accessCookie, err := r.Cookie("accessToken")
	if err != nil || accessCookie.Value == "" {
		return []string{}, fmt.Errorf("accessToken is not present")
	}

	accessToken, err := jwt.Parse(accessCookie.Value, func(token *jwt.Token) (any, error) {
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
		return []string{}, fmt.Errorf("refreshToken is not present")
	}

	refreshToken, err := jwt.Parse(refreshCookie.Value, func(token *jwt.Token) (any, error) {
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
