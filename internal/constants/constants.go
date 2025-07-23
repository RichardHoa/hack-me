/*
Package constants centralizes the management of application-wide constants and
environment-dependent configuration. It handles loading secrets and settings
from a .env file or the system environment, and provides a single source of
truth for values like token secrets, error codes, and standard messages.
*/
package constants

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

func LoadEnv() error {
	if _, err := os.Stat(".env"); err == nil {
		err := godotenv.Load(".env")
		if err != nil {
			fmt.Printf("Error loading .env file: %v\n", err)
		}
	} else if _, err := os.Stat("../../.env"); err == nil {
		err := godotenv.Load("../../.env")
		if err != nil {
			fmt.Printf("Error loading ../../.env file: %v\n", err)
		}
	} else {
		fmt.Printf("No .env file found - assuming environment variables\n")
	}

	RefreshTokenSecret = os.Getenv("REFRESH_TOKEN_SECRET")
	AccessTokenSecret = os.Getenv("ACCESS_TOKEN_SECRET")
	CSRFTokenSecret = os.Getenv("CSRF_TOKEN_SECRET")
	AppPort, _ = strconv.Atoi(os.Getenv("APP_PORT"))

	DevMode := os.Getenv("DEV_MODE")

	if DevMode == "LOCAL" {
		IsDevMode = true
	} else {
		IsDevMode = false
	}

	if RefreshTokenSecret == "" || AccessTokenSecret == "" || CSRFTokenSecret == "" {
		return errors.New("Missing token secrets in .env")
	}

	return nil
}

// Global variables holding configuration loaded from the environment.
var (
	RefreshTokenSecret string
	AccessTokenSecret  string
	CSRFTokenSecret    string
	AppPort            int
	IsDevMode          bool
)

// Defines the keys for standard claims within JSON Web Tokens.
const (
	TokenRefreshID = "refreshID"
	TokenUserName  = "userName"
	TokenUserID    = "userID"
)

// Defines general application constants
const (
	StatusInternalErrorMessage = "Internal server error"
	StatusInvalidJSONMessage   = "Invalid JSON format"
	UnauthorizedMessage        = "Unauthorized"
	ForbiddenMessage           = "Forbidden"
	AccessTokenTime            = 15 * time.Minute
	RefreshTokenTime           = 7 * (24 * time.Hour)
	CommentNestedLevel         = 5
	DefaultPageSize            = 10
)

// Defines standard error codes for API request validation failures.
const (
	MSG_INVALID_REQUEST_DATA     = "INVALID_REQUEST_DATA"
	MSG_MALFORMED_REQUEST_DATA   = "MALFORMED_REQUEST_DATA"
	MSG_CONFLICTING_FIELDS       = "CONFLICTING_FIELDS"
	MSG_LACKING_MANDATORY_FIELDS = "LACKING_MANDATORY_FIELDS"
)

// Defines enumerated integer codes for classifying various error types.
const (
	// Postgres Err
	PQUniqueViolation = iota
	PQForeignKeyViolation
	PQInvalidEnum
	PQInvalidTextRepresentation
	PQNotNullViolation
	PQCheckViolation
	PQNumericValueOutOfRange
	PQInvalidUUIDFormat
	PQDatatypeMismatch
	PQSyntaxError
	PQInvalidByteSequence
	// Application Err
	ResourceNotFound
	InvalidData
	InternalError
	LackingPermission
)

/*
PQErrorMessages maps standard PostgreSQL error codes (as strings) to the
application's internal integer-based error classification codes.
Source: https://www.postgresql.org/docs/current/errcodes-appendix.html
*/
var PQErrorMessages = map[string]int{
	"23505": PQUniqueViolation,
	"23503": PQForeignKeyViolation,
	"22P02": PQInvalidTextRepresentation,
	"23502": PQNotNullViolation,
	"23514": PQCheckViolation,
	"22003": PQNumericValueOutOfRange,
	"22P05": PQInvalidEnum,
	"22P04": PQInvalidUUIDFormat,
	"42804": PQDatatypeMismatch,
	"42601": PQSyntaxError,
	"22021": PQInvalidByteSequence, //input contains null bytes
}
