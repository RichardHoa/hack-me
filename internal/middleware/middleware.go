package middleware

import (
	"log"
	"net/http"

	"github.com/RichardHoa/hack-me/internal/constants"
	"github.com/RichardHoa/hack-me/internal/utils"
)

type MiddleWare struct {
	Logger *log.Logger
}

func NewMiddleWare(logger *log.Logger) MiddleWare {
	return MiddleWare{Logger: logger}
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
		_, refreshTokenID, err := utils.ValidateTokensFromCookies(r)
		if err != nil {
			middleware.Logger.Printf("Middleware > RequireCSRFToken: failed to validate refreshToken: %v", err)
			utils.WriteJSON(w, http.StatusUnauthorized, utils.NewMessage(
				constants.UnauthorizedMessage,
				constants.MSG_LACKING_MANDATORY_FIELDS,
				"",
			))
			return
		}

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
