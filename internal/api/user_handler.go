package api

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/RichardHoa/hack-me/internal/constants"
	"github.com/RichardHoa/hack-me/internal/store"
	"github.com/RichardHoa/hack-me/internal/utils"
)

type UserHandler struct {
	UserStore  store.UserStore
	TokenStore store.TokenStore
	Logger     *log.Logger
}

func NewUserHandler(userStore store.UserStore, tokenStore store.TokenStore, logger *log.Logger) *UserHandler {
	return &UserHandler{
		UserStore:  userStore,
		TokenStore: tokenStore,
		Logger:     logger,
	}
}

func (handler *UserHandler) RegisterNewUser(w http.ResponseWriter, r *http.Request) {

	var User store.User

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	err := decoder.Decode(&User)
	if err != nil {
		utils.WriteJSON(w, http.StatusBadRequest, utils.Message{"message": "invalid json format"})
		return
	}

	if User.Password.PlainText == "" && User.GoogleID == "" && User.GithubID == "" {
		utils.WriteJSON(w, http.StatusBadRequest, utils.Message{"message": "one of the three field password, google_id, github_id must not be null"})
		return
	}

	if User.Password.PlainText != "" {
		checkResult := utils.CheckPasswordValid(User.Password.PlainText)
		if checkResult.Error == nil && checkResult.ErrorMessage != "" {
			utils.WriteJSON(w, http.StatusBadRequest, utils.Message{"message": checkResult.ErrorMessage})
			return
		}

		if checkResult.Error != nil && checkResult.ErrorMessage == "" {
			utils.WriteJSON(w, http.StatusInternalServerError, utils.Message{"message": constants.StatusInternalErrorMessage})
			return
		}
	}

	_, err = handler.UserStore.CreateUser(&User)
	if err != nil {

		switch utils.ClassifyPgError(err) {
		case constants.PQUniqueViolation:
			utils.WriteJSON(w, http.StatusBadRequest, utils.Message{"message": "user_name or email or github_id or google_id already exist"})
			return

		case constants.PQCheckViolation:
			utils.WriteJSON(w, http.StatusBadRequest, utils.Message{"message": "one of the three field password, google_id, github_id must not be null"})
			return

		case constants.PQNotNullViolation:
			utils.WriteJSON(w, http.StatusBadRequest, utils.Message{"message": "user_name, email must not be null"})
			return

		default:
			handler.Logger.Printf("ERROR: RegisterNewUser > CreateUser: %v", err)
			utils.WriteJSON(w, http.StatusInternalServerError, utils.Message{"message": constants.StatusInternalErrorMessage})
			return

		}
	}

	utils.WriteJSON(w, http.StatusCreated, utils.Message{"message": "Register new user successfully"})

}

func (handler *UserHandler) LoginUser(w http.ResponseWriter, r *http.Request) {

	var user store.User

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	err := decoder.Decode(&user)
	if err != nil {
		utils.WriteJSON(w, http.StatusBadRequest, utils.Message{"message": "invalid json format"})
		return
	}

	if user.GoogleID != "" && user.GithubID != "" {
		utils.WriteJSON(w, http.StatusBadRequest, utils.Message{"message": "google_id, github_id cannot have value at the same time"})
		return
	}

	if user.Password.PlainText == "" && user.GoogleID == "" && user.GithubID == "" {
		utils.WriteJSON(w, http.StatusBadRequest, utils.Message{"message": "one of the three field password, google_id, github_id must not be null"})
		return
	}

	accessToken, refreshToken, err := handler.UserStore.LoginUser(&user)
	if err != nil {
		switch utils.ClassifyPgError(err) {
		case constants.InvalidData:
			// data is lacking in somewhere
			utils.WriteJSON(w, http.StatusBadRequest, utils.Message{"message": "invalid data"})
			return
		default:
			// NOTE: Test more data handling
			handler.Logger.Printf("ERROR: RegisterNewUser > LoginUser: %v", err)
			utils.WriteJSON(w, http.StatusInternalServerError, utils.Message{"message": constants.StatusInternalErrorMessage})
			return

		}
	}

	err = handler.TokenStore.AddRefreshToken(refreshToken, user.ID)
	if err != nil {
		switch utils.ClassifyPgError(err) {
		default:
			handler.Logger.Printf("ERROR: RegisterNewUser > AddRefreshToken: %v", err)
			utils.WriteJSON(w, http.StatusInternalServerError, utils.Message{"message": constants.StatusInternalErrorMessage})
			return

		}
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    accessToken,
		Expires:  time.Now().Add(constants.AccessTokenTime),
		MaxAge:   int(constants.AccessTokenTime.Seconds()),
		HttpOnly: false,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		Expires:  time.Now().Add(constants.RefreshTokenTime),
		MaxAge:   int(constants.RefreshTokenTime.Seconds()),
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	})

	utils.WriteJSON(w, http.StatusOK, utils.Message{"message": "Log in successfully"})

}
