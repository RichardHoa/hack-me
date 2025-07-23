package utils

import (
	"bufio"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"
)

/*
PasswordCheckResult encapsulates the result of a password validation check.
*/
type PasswordCheckResult struct {
	Error        error
	ErrorMessage string
}

/*
CheckPasswordValid checks a password against local format rules and the 'Have I
Been Pwned' public breach database.
*/
func CheckPasswordValid(password string) PasswordCheckResult {
	if utf8.RuneCountInString(password) < 8 {
		return PasswordCheckResult{nil, "Password length must be over 8 character"}
	}

	byteLen := len([]byte(password))
	if byteLen > 256 {
		return PasswordCheckResult{nil, "Password length is too long"}
	}

	// SHA-1 hash
	hash := sha1.Sum([]byte(password))
	sha1Hex := strings.ToUpper(hex.EncodeToString(hash[:]))
	prefix := sha1Hex[:5]
	suffix := sha1Hex[5:]

	url := fmt.Sprintf("https://api.pwnedpasswords.com/range/%s", prefix)

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get(url)
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

	// Search suffix in list
	scanner := bufio.NewScanner(strings.NewReader(string(body)))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, ":")
		if len(parts) != 2 {
			continue
		}
		// check with suffix
		if strings.TrimSpace(parts[0]) == suffix {
			return PasswordCheckResult{nil, fmt.Sprintf("Your password has been found in breach %v times, please change to a more secure password", parts[1])}
		}
	}

	if err := scanner.Err(); err != nil {
		return PasswordCheckResult{err, ""}
	}

	return PasswordCheckResult{nil, ""}
}
