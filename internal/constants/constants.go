package constants

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
)

func LoadEnv() error {
	err := godotenv.Load(".env")
	if err != nil {
		err = godotenv.Load("../../.env")
		if err != nil {
			return err
		}
	}

	RefreshTokenSecret = os.Getenv("REFRESHTOKENSECRET")
	AccessTokenSecret = os.Getenv("ACCESSTOKENSECRET")
	DevMode := os.Getenv("DEV_MODE")

	if DevMode == "local" {
		IsDevMode = true
	} else {
		IsDevMode = false
	}

	if RefreshTokenSecret == "" || AccessTokenSecret == "" {
		return errors.New("Missing token secrets in .env")
	}
	return nil
}

var (
	RefreshTokenSecret string
	AccessTokenSecret  string
	IsDevMode          bool
)

const (
	StatusInternalErrorMessage = "Internal error"
	AccessTokenTime            = 15 * time.Minute
	RefreshTokenTime           = 7 * (24 * time.Hour)
)

const (
	PQUniqueViolation           = iota // 0
	PQForeignKeyViolation              // 1
	PQInvalidEnum                      // 2
	PQInvalidTextRepresentation        // 3 (e.g. casting string to int/date fails)
	PQNotNullViolation                 // 4
	PQCheckViolation                   // 5
	PQNumericValueOutOfRange           // 6
	PQInvalidUUIDFormat                // 7
	PQDatatypeMismatch                 // 8
	PQSyntaxError                      // 9
	ResourceNotFound                   // 10
	InvalidData                        // 11

)

var PQErrorMessages = map[string]int{
	"23505":                             PQUniqueViolation,           // unique_violation
	"23503":                             PQForeignKeyViolation,       // foreign_key_violation
	"22P02":                             PQInvalidTextRepresentation, // invalid_text_representation (e.g. ENUM, int fail)
	"23502":                             PQNotNullViolation,          // not_null_violation
	"23514":                             PQCheckViolation,            // check_violation
	"22003":                             PQNumericValueOutOfRange,    // numeric_value_out_of_range
	"22P05":                             PQInvalidEnum,               // untranslatable character or invalid enum
	"22P04":                             PQInvalidUUIDFormat,         // bad UUID text representation
	"42804":                             PQDatatypeMismatch,          // datatype_mismatch
	"42601":                             PQSyntaxError,               // syntax_error
	fmt.Sprintf("%v", ResourceNotFound): ResourceNotFound,
	fmt.Sprintf("%v", InvalidData):      InvalidData,
}
