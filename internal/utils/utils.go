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
	"unicode/utf8"

	"github.com/RichardHoa/hack-me/internal/constants"
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
