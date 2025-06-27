package constants

import (
	"fmt"
	"time"
)

const (
	StatusInternalErrorMessage = "Internal error"
	RefreshTokenSecret         = "JWTSECRET, hahahahaha!"
	AccessTokenSecret          = "AccesstokenSecert, haha"
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
