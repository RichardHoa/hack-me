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

var (
	RefreshTokenSecret string
	AccessTokenSecret  string
	CSRFTokenSecret    string
	AppPort            int
	IsDevMode          bool
)

const (
	TokenRefreshID = "refreshID"
	TokenUserName  = "userName"
	TokenUserID    = "userID"
)

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

const (
	MSG_INVALID_REQUEST_DATA     = "INVALID_REQUEST_DATA"
	MSG_MALFORMED_REQUEST_DATA   = "MALFORMED_REQUEST_DATA"
	MSG_CONFLICTING_FIELDS       = "CONFLICTING_FIELDS"
	MSG_LACKING_MANDATORY_FIELDS = "LACKING_MANDATORY_FIELDS"
)

const (
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
	ResourceNotFound
	InvalidData
	InternalError
	LackingPermission
)

var PQErrorMessages = map[string]int{
	"23505": PQUniqueViolation,           // unique_violation
	"23503": PQForeignKeyViolation,       // foreign_key_violation
	"22P02": PQInvalidTextRepresentation, // invalid_text_representation (e.g. ENUM, int fail)
	"23502": PQNotNullViolation,          // not_null_violation
	"23514": PQCheckViolation,            // check_violation
	"22003": PQNumericValueOutOfRange,    // numeric_value_out_of_range
	"22P05": PQInvalidEnum,               // untranslatable character or invalid enum
	"22P04": PQInvalidUUIDFormat,         // bad UUID text representation
	"42804": PQDatatypeMismatch,          // datatype_mismatch
	"42601": PQSyntaxError,               // syntax_error
	"22021": PQInvalidByteSequence,       //input contains null bytes
}
