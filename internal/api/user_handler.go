package api

import (
	"encoding/json"
	"github.com/RichardHoa/hack-me/internal/constants"
	"github.com/RichardHoa/hack-me/internal/store"
	"github.com/RichardHoa/hack-me/internal/utils"
	"log"
	"net/http"
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
		utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage(constants.StatusInvalidJSONMessage, constants.MSG_MALFORMED_REQUEST_DATA, "request"))
		return
	}

	if User.Username == "" || User.Email == "" {
		utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage("Two fields must exist", constants.MSG_LACKING_MANDATORY_FIELDS, "userName and email"))
		return
	}

	if User.Password.PlainText == "" && User.GoogleID == "" && User.GithubID == "" {
		utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage("One of the three fields must exist", constants.MSG_LACKING_MANDATORY_FIELDS, "googleID and githubID and password"))
		return
	}

	if User.Password.PlainText != "" {
		checkResult := utils.CheckPasswordValid(User.Password.PlainText)
		if checkResult.Error == nil && checkResult.ErrorMessage != "" {
			utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage("Invalid password", constants.MSG_INVALID_REQUEST_DATA, checkResult.ErrorMessage))
			return
		}

		if checkResult.Error != nil && checkResult.ErrorMessage == "" {
			utils.WriteJSON(w, http.StatusInternalServerError, utils.NewMessage(constants.StatusInternalErrorMessage, "", ""))
			return
		}
	}

	_, err = handler.UserStore.CreateUser(&User)
	if err != nil {
		switch utils.ClassifyError(err) {
		case constants.PQUniqueViolation:
			utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage("User already exist", constants.MSG_INVALID_REQUEST_DATA, "userName or email"))
			return
		default:
			handler.Logger.Printf("ERROR: RegisterNewUser > CreateUser: %v", err)
			utils.WriteJSON(w, http.StatusInternalServerError, utils.NewMessage(constants.StatusInternalErrorMessage, "", ""))
			return

		}
	}

	utils.WriteJSON(w, http.StatusCreated, utils.NewMessage("Register new user successfully", "", ""))

}

func (handler *UserHandler) LoginUser(w http.ResponseWriter, r *http.Request) {

	var user store.User

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	err := decoder.Decode(&user)
	if err != nil {
		utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage(constants.StatusInvalidJSONMessage, constants.MSG_MALFORMED_REQUEST_DATA, "request"))
		return
	}

	if user.GoogleID != "" && user.GithubID != "" {
		utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage("googleID and githubID cannot coexist", constants.MSG_CONFLICTING_FIELDS, "googleID and githubID"))
		return
	}

	if user.Password.PlainText == "" && user.GoogleID == "" && user.GithubID == "" {
		utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage("One of the three fields must exist", constants.MSG_LACKING_MANDATORY_FIELDS, "googleID and githubID and password"))
		return
	}

	accessToken, refreshToken, csrfToken, err := handler.UserStore.LoginAndIssueTokens(&user)
	if err != nil {
		handler.Logger.Printf("ERROR: LoginUser > LoginAndIssueTokens: %v", err)
		switch utils.ClassifyError(err) {
		case constants.InvalidData:
			utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage("Either password is incorrect or user email does not exist", constants.MSG_INVALID_REQUEST_DATA, "email and password"))
			return
		default:
			utils.WriteJSON(w, http.StatusInternalServerError, utils.NewMessage(constants.StatusInternalErrorMessage, "", ""))
			return

		}
	}

	err = handler.TokenStore.AddRefreshToken(refreshToken, user.ID)
	if err != nil {
		switch utils.ClassifyError(err) {
		default:
			handler.Logger.Printf("ERROR: LoginUser > AddRefreshToken: %v", err)
			utils.WriteJSON(w, http.StatusInternalServerError, utils.NewMessage(constants.StatusInternalErrorMessage, "", ""))
			return

		}
	}

	utils.SendTokens(w, accessToken, refreshToken, csrfToken)

	utils.WriteJSON(w, http.StatusOK, utils.NewMessage("Successful authentication", "", ""))
}
