package utils

import (
	"errors"
	"github.com/RichardHoa/hack-me/internal/constants"
	"github.com/jackc/pgx/v5/pgconn"
)

// CustomAppError represents an application-specific error with a unique code
// and a descriptive message.
type CustomAppError struct {
	Code int
	Msg  string
}

// Error implements the standard error interface, returning the error message.
func (e *CustomAppError) Error() string {
	return e.Msg
}

// NewCustomAppError creates and returns a new instance of CustomAppError.
func NewCustomAppError(code int, msg string) *CustomAppError {
	return &CustomAppError{Code: code, Msg: msg}
}

/*
ClassifyError inspects a given error and returns a corresponding integer code.
It prioritizes classifying PostgreSQL-specific errors by looking up their
unique violation codes. If the error is not a PostgreSQL error, it then
checks if it's a CustomAppError and returns its code. It returns 0 for
nil errors or if the error type is unclassified
*/
func ClassifyError(err error) int {
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

	// Custom application-level error
	var customErr *CustomAppError
	if errors.As(err, &customErr) {
		return customErr.Code
	}

	return 0
}
