package api

import (
	"encoding/json"
	"log"
	"net/http"

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
			utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage(checkResult.ErrorMessage, constants.MSG_INVALID_REQUEST_DATA, checkResult.ErrorMessage))
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

	err = handler.TokenStore.DeleteRefreshToken(user.ID)
	if err != nil {
		switch utils.ClassifyError(err) {
		default:
			handler.Logger.Printf("ERROR: LoginUser > DeleteRefreshToken: %v", err)
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

	err = utils.SendTokens(w, accessToken, refreshToken, csrfToken)
	if err != nil {
		handler.Logger.Printf("ERROR: LoginUser > Send tokens: %v", err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.NewMessage(constants.StatusInternalErrorMessage, "", ""))
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.NewMessage("Successful authentication", "", ""))
}

func (handler *UserHandler) LogoutUser(w http.ResponseWriter, r *http.Request) {

	result, err := utils.ValidateTokensFromCookies(r, []string{constants.TokenUserID, constants.TokenRefreshID})
	if err != nil {
		handler.Logger.Printf("ERROR: Logout user > JWT token checking: %v", err)
		utils.WriteJSON(w, http.StatusUnauthorized, utils.NewMessage(constants.UnauthorizedMessage, constants.MSG_LACKING_MANDATORY_FIELDS, "cookies"))
		return
	}
	userID := result[0]
	refreshTokenID := result[1]

	DBRefreshToken, err := handler.TokenStore.GetRefreshToken(userID)
	if err != nil {
		handler.Logger.Printf("ERROR: Logout user > get refresh token : %v", err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.NewMessage(constants.StatusInternalErrorMessage, "", ""))
		return
	}

	// maybe the previous user session logout
	// so the two IDs may not match
	if DBRefreshToken.ID != refreshTokenID {
		handler.Logger.Printf("ABNORMAL: Logout user >  tokenID from browser %s, tokenID from database %s", refreshTokenID, DBRefreshToken.ID)
	} else {
		// only empty the db when two IDs are the same
		err = handler.TokenStore.DeleteRefreshToken(userID)
		if err != nil {
			handler.Logger.Printf("ERROR: Logout user > delete refresh token : %v", err)
			utils.WriteJSON(w, http.StatusInternalServerError, utils.NewMessage(constants.StatusInternalErrorMessage, "", ""))
			return
		}
	}

	utils.SendEmptyTokens(w)
	utils.WriteJSON(w, http.StatusOK, utils.NewMessage("log out successful", "", ""))
}

func (handler *UserHandler) RefreshTokenRotation(w http.ResponseWriter, r *http.Request) {

	result, err := utils.ValidateTokensFromCookiesWithoutAccessToken(r, []string{constants.TokenUserID, constants.TokenRefreshID, constants.TokenUserName})

	if err != nil {
		handler.Logger.Printf("ERROR: Refresh-token-rotation > JWT token checking: %v", err)
		utils.WriteJSON(w, http.StatusUnauthorized, utils.NewMessage(constants.UnauthorizedMessage, constants.MSG_LACKING_MANDATORY_FIELDS, "cookies"))
		return
	}

	userID := result[0]
	refreshTokenID := result[1]
	userName := result[2]

	DBRefreshToken, err := handler.TokenStore.GetRefreshToken(userID)
	if err != nil {
		handler.Logger.Printf("ERROR: Refresh-token-rotation > get refresh token : %v", err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.NewMessage(constants.StatusInternalErrorMessage, "", ""))
		return
	}

	if DBRefreshToken.ID != refreshTokenID {
		err = handler.TokenStore.DeleteRefreshToken(userID)
		if err != nil {
			handler.Logger.Printf("ERROR: Refresh-token-rotation > delete refresh token : %v", err)
			utils.WriteJSON(w, http.StatusInternalServerError, utils.NewMessage(constants.StatusInternalErrorMessage, "", ""))
			return
		}

		utils.SendEmptyTokens(w)
		handler.Logger.Printf("ERROR: Refresh-token-rotation >  tokenID from browser %s, tokenID from database %s", refreshTokenID, DBRefreshToken.ID)
		utils.WriteJSON(w, http.StatusForbidden, utils.NewMessage(constants.ForbiddenMessage, "", ""))
		return
	}

	refreshTokenTimeUntilExpiry := DBRefreshToken.CreatedAt.Add(constants.RefreshTokenTime).Unix()

	accessToken, refreshToken, err := utils.CreateTokens(userID, userName, refreshTokenTimeUntilExpiry)

	result, err = utils.ExtractClaimsFromJWT(refreshToken, []string{"refreshID"})
	if err != nil {
		handler.Logger.Printf("ERROR: Refresh-token-rotation >  extract claim from refresh token %v", err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.NewMessage(constants.StatusInternalErrorMessage, "", ""))
		return
	}
	refreshtokenID := result[0]

	csrfToken, err := utils.CreateCSRFToken(refreshtokenID)
	if err != nil {
		handler.Logger.Printf("ERROR: Refresh-token-rotation >  create CSRF Token %v", err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.NewMessage(constants.StatusInternalErrorMessage, "", ""))
		return

	}

	err = handler.TokenStore.AddRefreshToken(refreshToken, userID)
	if err != nil {
		switch utils.ClassifyError(err) {
		default:
			handler.Logger.Printf("ERROR: Refresh-token-rotation > AddRefreshToken: %v", err)
			utils.WriteJSON(w, http.StatusInternalServerError, utils.NewMessage(constants.StatusInternalErrorMessage, "", ""))
			return

		}
	}

	err = utils.SendTokens(w, accessToken, refreshToken, csrfToken)
	if err != nil {
		handler.Logger.Printf("ERROR: RefreshTokenRotation > Send tokens: %v", err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.NewMessage(constants.StatusInternalErrorMessage, "", ""))
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.NewMessage("Getting new access token successful", "", ""))
}
