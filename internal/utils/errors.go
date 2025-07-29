package utils

import (
	"errors"
	"github.com/RichardHoa/hack-me/internal/constants"
	"github.com/jackc/pgx/v5/pgconn"
)

type CustomAppError struct {
	Code int    // application-specific error
	Msg  string // message to send to user
}

// CustomAppError implements the standard error interface
func (e *CustomAppError) Error() string {
	return e.Msg
}

func NewCustomAppError(code int, msg string) *CustomAppError {
	return &CustomAppError{Code: code, Msg: msg}
}

/*
ClassifyError inspects a given error and returns a corresponding integer code.
It returns 0 for nil errors or if the error type is unclassified
*/
func ClassifyError(err error) int {
	if err == nil {
		return 0
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if msg, ok := constants.PQErrorMessages[pgErr.Code]; ok {
			return msg
		}
	}

	var customErr *CustomAppError
	if errors.As(err, &customErr) {
		return customErr.Code
	}

	return 0
}
