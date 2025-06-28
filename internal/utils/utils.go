package utils

import (
	"bufio"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/RichardHoa/hack-me/internal/constants"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgconn"
)

type Message map[string]interface{}

func BeautifyJSON(v interface{}) string {
	bytes, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fmt.Printf("Error marshaling JSON: %v\n", err)
		return ""
	}
	return (string(bytes))
}

func WriteJSON(w http.ResponseWriter, statusCode int, data Message) error {
	jsonBytes, err := json.MarshalIndent(data, "", " ")
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	w.Write(jsonBytes)

	return nil
}

func ClassifyPgError(err error) int {
	if err == nil {
		return 0
	}

	// First, try PostgreSQL-specific error
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if msg, ok := constants.PQErrorMessages[pgErr.Code]; ok {
			return msg
		}
	}

	// Fallback: treat error string as exact code
	if msg, ok := constants.PQErrorMessages[err.Error()]; ok {
		return msg
	}

	return 0
}
func NullIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

type PasswordCheckResult struct {
	Error        error
	ErrorMessage string
}

// checkFormatOnly: if true, skip breach check
func CheckPasswordValid(password string) PasswordCheckResult {
	// 1. Type and length check
	if utf8.RuneCountInString(password) < 8 {
		return PasswordCheckResult{nil, "Password length must be over 8 character"}
	}

	byteLen := len([]byte(password))
	if byteLen > 256 {
		return PasswordCheckResult{nil, "Password length is too long"}
	}

	// 2. SHA-1 hash
	hash := sha1.Sum([]byte(password))
	sha1Hex := strings.ToUpper(hex.EncodeToString(hash[:]))
	prefix := sha1Hex[:5]
	suffix := sha1Hex[5:]

	// 3. Fetch from Pwned Passwords
	url := fmt.Sprintf("https://api.pwnedpasswords.com/range/%s", prefix)
	resp, err := http.Get(url)
	if err != nil {
		return PasswordCheckResult{err, ""}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return PasswordCheckResult{err, ""}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return PasswordCheckResult{err, ""}
	}

	// 4. Search suffix in list
	scanner := bufio.NewScanner(strings.NewReader(string(body)))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, ":")
		if len(parts) != 2 {
			continue
		}
		if strings.TrimSpace(parts[0]) == suffix {
			return PasswordCheckResult{nil, fmt.Sprintf("Your password has been found in breach %v times, please change to a more secure password", parts[1])}
		}
	}

	if err := scanner.Err(); err != nil {
		return PasswordCheckResult{err, ""}
	}

	return PasswordCheckResult{nil, ""}
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
	fmt.Printf("Access cookie: %s", accessCookie.String())
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
	fmt.Printf("Refresh cookie: %s", refreshCookie.String())
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
