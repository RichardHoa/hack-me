package middleware

import (
	"log"
	"net/http"
	"time"

	"github.com/RichardHoa/hack-me/internal/constants"
	"github.com/RichardHoa/hack-me/internal/utils"
)

type MiddleWare struct {
	Logger *log.Logger
}

func NewMiddleWare(logger *log.Logger) MiddleWare {
	return MiddleWare{Logger: logger}
}

func (middleware *MiddleWare) CorsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Vary", "Origin")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (middleware *MiddleWare) NoCacheMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, proxy-revalidate, max-age=0")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")

		next.ServeHTTP(w, r)
	})
}

func (middleware *MiddleWare) WaitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(300 * time.Millisecond)

		// Continue with the next handler
		next.ServeHTTP(w, r)
	})
}

func (middleware *MiddleWare) RequireCSRFToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// 1: Get CSRF token from header
		csrfToken := r.Header.Get("X-CSRF-Token")
		if csrfToken == "" {
			utils.WriteJSON(w, http.StatusUnauthorized, utils.NewMessage(
				constants.UnauthorizedMessage,
				constants.MSG_LACKING_MANDATORY_FIELDS,
				"",
			))
			return
		}

		// Step 2: Validate refresh token from cookies
		result, err := utils.GetValuesFromCookie(r, []string{constants.JWTRefreshTokenID})
		if err != nil {
			middleware.Logger.Printf("Middleware > RequireCSRFToken: failed to validate refreshToken: %v", err)
			utils.WriteJSON(w, http.StatusUnauthorized, utils.NewMessage(
				constants.UnauthorizedMessage,
				constants.MSG_LACKING_MANDATORY_FIELDS,
				"",
			))
			return
		}

		refreshTokenID := result[0]

		// Step 3: Validate CSRF token against refreshTokenID
		isValid, err := utils.CheckCSRFToken(csrfToken, refreshTokenID)
		if err != nil {
			middleware.Logger.Printf("Middleware > RequireCSRFToken: error checking CSRF token: %v", err)
			utils.WriteJSON(w, http.StatusInternalServerError, utils.NewMessage(
				constants.StatusInternalErrorMessage,
				"",
				"",
			))
			return
		}

		if !isValid {
			utils.WriteJSON(w, http.StatusUnauthorized, utils.NewMessage(
				constants.UnauthorizedMessage,
				constants.MSG_LACKING_MANDATORY_FIELDS,
				"",
			))
			return
		}

		// Step 4: Proceed to the next handler
		next.ServeHTTP(w, r)
	})
}
