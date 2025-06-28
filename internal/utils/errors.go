package utils

import (
	"errors"

	"github.com/RichardHoa/hack-me/internal/constants"
	"github.com/jackc/pgconn"
)

type CustomAppError struct {
	Code int
	Msg  string
}

func (e *CustomAppError) Error() string {
	return e.Msg
}

func NewCustomAppError(code int, msg string) *CustomAppError {
	return &CustomAppError{Code: code, Msg: msg}
}

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
