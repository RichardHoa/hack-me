package api

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"strings"
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

func (handler *UserHandler) GetUserActivity(w http.ResponseWriter, r *http.Request) {
	result, err := utils.GetValuesFromCookie(r, []string{constants.JWTUserID})
	if err != nil {
		handler.Logger.Printf("ERROR: GetUserActivity > JWT token checking: %v", err)
		utils.WriteJSON(w, http.StatusUnauthorized, utils.NewMessage(constants.UnauthorizedMessage, constants.MSG_LACKING_MANDATORY_FIELDS, ""))
		return
	}
	userID := result[0]

	activityData, err := handler.UserStore.GetUserActivity(userID)
	if err != nil {
		switch utils.ClassifyError(err) {
		case constants.ResourceNotFound:
			utils.WriteJSON(w, http.StatusNotFound, utils.NewMessage(err.Error(), constants.MSG_INVALID_REQUEST_DATA, "user"))
			return
		default:
			handler.Logger.Printf("ERROR: GetUserActivity > GetUserActivity: %v", err)
			utils.WriteJSON(w, http.StatusInternalServerError, utils.NewMessage(constants.StatusInternalErrorMessage, "", ""))
			return
		}
	}

	utils.WriteJSON(w, http.StatusOK, utils.Message{"data": activityData})
}

func (handler *UserHandler) RegisterNewUser(w http.ResponseWriter, r *http.Request) {

	var User store.User

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	err := decoder.Decode(&User)
	if err != nil {
		utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage(constants.StatusInvalidBodyMessage, constants.MSG_MALFORMED_REQUEST_DATA, "request"))
		return
	}

	if User.ImageLink != "" {
		u, err := url.Parse(User.ImageLink)
		if err != nil || u.Scheme == "" || u.Host == "" {
			utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage("imageLink must be a valid absolute URL", constants.MSG_INVALID_REQUEST_DATA, "imageLink"))
			return
		}

		host := strings.ToLower(u.Hostname())
		if !strings.HasSuffix(host, "googleusercontent.com") && host != "avatars.githubusercontent.com" {
			utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage("External CDN links are not allowed", constants.MSG_INVALID_REQUEST_DATA, "imageLink"))
			return
		}
	}

	if strings.TrimSpace(User.Username) == "" || strings.TrimSpace(User.Email) == "" {
		utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage("Two fields must exist", constants.MSG_LACKING_MANDATORY_FIELDS, "userName and email"))
		return
	}

	if User.Password.PlainText != "" {
		checkResult := utils.CheckPasswordValid(User.Password.PlainText)
		if checkResult.Error == nil && checkResult.ErrorMessage != "" {
			utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage(checkResult.ErrorMessage, constants.MSG_INVALID_REQUEST_DATA, "password"))
			return
		}

		if checkResult.Error != nil && checkResult.ErrorMessage == "" {
			utils.WriteJSON(w, http.StatusInternalServerError, utils.NewMessage(checkResult.ErrorMessage, "", ""))
			return
		}
	}

	//TODO: We forget to check for email validity here

	_, err = handler.UserStore.CreateUser(&User)
	if err != nil {
		switch utils.ClassifyError(err) {
		case constants.PQInvalidByteSequence:
			utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage("input contains null character", constants.MSG_INVALID_REQUEST_DATA, "body"))
		case constants.PQUniqueViolation:
			utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage("User already exist, please try again with a different account", constants.MSG_INVALID_REQUEST_DATA, "userName or email"))
			return
		case constants.PQCheckViolation:
			utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage("One of the three fields must exist", constants.MSG_LACKING_MANDATORY_FIELDS, "googleID and githubID and password"))
			return

		default:
			handler.Logger.Printf("ERROR: RegisterNewUser > CreateUser: %v", err)
			utils.WriteJSON(w, http.StatusInternalServerError, utils.NewMessage(err.Error(), "", ""))
			return
		}
	}

	utils.WriteJSON(w, http.StatusCreated, utils.NewMessage("Register new user successfully", "", ""))

}

func (handler *UserHandler) ChangeUsername(w http.ResponseWriter, r *http.Request) {
	result, err := utils.GetValuesFromCookie(r, []string{constants.JWTUserID})
	if err != nil {
		handler.Logger.Printf("ERROR: ChangeUsername > JWT token checking: %v", err)
		utils.WriteJSON(w, http.StatusUnauthorized, utils.NewMessage(constants.UnauthorizedMessage, constants.MSG_LACKING_MANDATORY_FIELDS, ""))
		return
	}
	userID := result[0]

	var req store.ChangeUsernameRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	err = decoder.Decode(&req)
	if err != nil {
		utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage(constants.StatusInvalidBodyMessage, constants.MSG_MALFORMED_REQUEST_DATA, "request"))
		return
	}

	trimmedUsername := strings.TrimSpace(req.NewUsername)
	if trimmedUsername == "" {
		utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage("newUsername is required", constants.MSG_LACKING_MANDATORY_FIELDS, "newUsername"))
		return
	}

	req.UserID = userID
	req.NewUsername = trimmedUsername

	err = handler.UserStore.ChangeUsername(req)
	if err != nil {
		switch utils.ClassifyError(err) {
		case constants.PQUniqueViolation:
			utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage("username already exists", constants.MSG_INVALID_REQUEST_DATA, "newUsername"))
			return
		case constants.ResourceNotFound:
			utils.WriteJSON(w, http.StatusNotFound, utils.NewMessage(err.Error(), constants.MSG_INVALID_REQUEST_DATA, "user"))
			return
		default:
			handler.Logger.Printf("ERROR: ChangeUsername > userStore.ChangeUsername: %v", err)
			utils.WriteJSON(w, http.StatusInternalServerError, utils.NewMessage(constants.StatusInternalErrorMessage, "", ""))
			return
		}
	}

	utils.WriteJSON(w, http.StatusOK, utils.NewMessage("Username changed successfully", "", ""))
}

func (handler *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	result, err := utils.GetValuesFromCookie(r, []string{constants.JWTUserID})
	if err != nil {
		handler.Logger.Printf("ERROR: DeleteUser > JWT token checking: %v", err)
		utils.WriteJSON(w, http.StatusUnauthorized, utils.NewMessage(constants.UnauthorizedMessage, constants.MSG_LACKING_MANDATORY_FIELDS, ""))
		return
	}
	userID := result[0]

	//NOTE: We need to check for password

	err = handler.UserStore.DeleteUser(userID)
	if err != nil {
		switch utils.ClassifyError(err) {
		case constants.ResourceNotFound:
			utils.WriteJSON(w, http.StatusNotFound, utils.NewMessage(err.Error(), constants.MSG_INVALID_REQUEST_DATA, "user"))
			return
		default:
			handler.Logger.Printf("ERROR: DeleteUser > userStore.DeleteUser: %v", err)
			utils.WriteJSON(w, http.StatusInternalServerError, utils.NewMessage(constants.StatusInternalErrorMessage, "", ""))
			return
		}
	}

	utils.SendEmptyTokens(w)
	utils.WriteJSON(w, http.StatusOK, utils.NewMessage("User deleted successfully", "", ""))
}

func (handler *UserHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	result, err := utils.GetValuesFromCookie(r, []string{constants.JWTUserID})
	if err != nil {
		handler.Logger.Printf("ERROR: ChangePassword > JWT token checking: %v", err)
		utils.WriteJSON(w, http.StatusUnauthorized, utils.NewMessage(constants.UnauthorizedMessage, constants.MSG_LACKING_MANDATORY_FIELDS, ""))
		return
	}
	userID := result[0]

	var req store.ChangePasswordRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	err = decoder.Decode(&req)
	if err != nil {
		utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage(constants.StatusInvalidBodyMessage, constants.MSG_MALFORMED_REQUEST_DATA, "request"))
		return
	}
	if req.OldPassword == req.NewPassword {
		utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage("old password cannot be the same as new password", constants.MSG_INVALID_REQUEST_DATA, "oldPassword, newPassword"))
		return
	}

	if req.NewPassword == "" {
		utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage("newPassword is required", constants.MSG_LACKING_MANDATORY_FIELDS, "newPassword"))
		return
	}

	checkResult := utils.CheckPasswordValid(req.NewPassword)
	if checkResult.Error == nil && checkResult.ErrorMessage != "" {
		utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage(checkResult.ErrorMessage, constants.MSG_INVALID_REQUEST_DATA, "newPassword"))
		return
	}
	if checkResult.Error != nil {
		handler.Logger.Printf("ERROR: ChangePassword > CheckPasswordValid: %v", checkResult.Error)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.NewMessage(constants.StatusInternalErrorMessage, "", ""))
		return
	}

	req.UserID = userID
	err = handler.UserStore.ChangePassword(req)
	if err != nil {
		handler.Logger.Printf("ERROR: ChangePassword > userStore.ChangePassword: %v", err)
		switch utils.ClassifyError(err) {
		case constants.InvalidData:
			utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage(err.Error(), constants.MSG_INVALID_REQUEST_DATA, "password"))
			return
		case constants.ResourceNotFound:
			// unusual case, there is something odd here
			utils.WriteJSON(w, http.StatusInternalServerError, utils.NewMessage("System error, please contact admin to change your password", "", ""))
			return
		default:
			utils.WriteJSON(w, http.StatusInternalServerError, utils.NewMessage(constants.StatusInternalErrorMessage, "", ""))
			return
		}
	}

	utils.WriteJSON(w, http.StatusOK, utils.NewMessage("Password changed successfully", "", ""))
}

func (handler *UserHandler) LoginUser(w http.ResponseWriter, r *http.Request) {

	startTime := time.Now()
	const targetDuration = 400 * time.Millisecond

	defer func() {
		elapsed := time.Since(startTime)
		if elapsed < targetDuration {
			remaining := targetDuration - elapsed
			if remaining > 0 {
				time.Sleep(remaining)
			}
		}
	}()

	var user store.User

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	err := decoder.Decode(&user)
	if err != nil {
		utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage(constants.StatusInvalidBodyMessage, constants.MSG_MALFORMED_REQUEST_DATA, "request"))
		return
	}

	if strings.TrimSpace(user.Email) == "" {
		utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage("email must exist", constants.MSG_LACKING_MANDATORY_FIELDS, "email"))
		return
	}

	if user.GoogleID != "" && user.GithubID != "" {
		utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage("googleID and githubID cannot coexist", constants.MSG_CONFLICTING_FIELDS, "googleID and githubID"))
		return
	}

	if user.Password.PlainText == "" && user.GoogleID == "" && user.GithubID == "" {
		utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage("One of the three fields must exist", constants.MSG_LACKING_MANDATORY_FIELDS, "googleID or githubID or password"))
		return
	}

	accessToken, refreshToken, csrfToken, err := handler.UserStore.LoginAndIssueTokens(&user)
	if err != nil {
		handler.Logger.Printf("ERROR: LoginUser > LoginAndIssueTokens: %v", err)
		switch utils.ClassifyError(err) {
		case constants.PQInvalidByteSequence:
			utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage("input contains null character", constants.MSG_INVALID_REQUEST_DATA, "email and password"))
		case constants.InvalidData:
			utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage(err.Error(), constants.MSG_INVALID_REQUEST_DATA, "email and password"))
		case constants.ResourceNotFound:
			utils.WriteJSON(w, http.StatusNotFound, utils.NewMessage(err.Error(), "your account is not found", ""))
		default:
			utils.WriteJSON(w, http.StatusInternalServerError, utils.NewMessage(constants.StatusInternalErrorMessage, "", ""))
		}
		return
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

	result, err := utils.GetValuesFromCookie(r, []string{constants.JWTUserID, constants.JWTRefreshTokenID})
	if err != nil {
		utils.SendEmptyTokens(w)

		handler.Logger.Printf("ERROR: Logout user > JWT token checking: %v", err)
		utils.WriteJSON(w, http.StatusUnauthorized, utils.NewMessage(constants.UnauthorizedMessage, constants.MSG_LACKING_MANDATORY_FIELDS, ""))
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

	result, err := utils.GetValuesFromCookieWithoutAccessToken(r, []string{constants.JWTUserID, constants.JWTRefreshTokenID})

	if err != nil {
		handler.Logger.Printf("ERROR: Refresh-token-rotation > JWT token checking: %v", err)
		utils.WriteJSON(w, http.StatusUnauthorized, utils.NewMessage(constants.UnauthorizedMessage, constants.MSG_LACKING_MANDATORY_FIELDS, ""))
		return
	}

	userID := result[0]
	refreshTokenID := result[1]
	userName, err := handler.UserStore.GetUserName(userID)
	if err != nil {
		handler.Logger.Printf("ERROR: Refresh-token-rotation > get user name : %v", err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.NewMessage(constants.StatusInternalErrorMessage, "", ""))
		return
	}

	DBRefreshToken, err := handler.TokenStore.GetRefreshToken(userID)
	if err != nil {
		handler.Logger.Printf("ERROR: Refresh-token-rotation > get refresh token : %v", err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.NewMessage(constants.StatusInternalErrorMessage, "", ""))
		return
	}

	if DBRefreshToken.ID != refreshTokenID {
		handler.Logger.Printf("ERROR: Refresh-token-rotation > tokenID from browser |%s|, tokenID from database |%s|", refreshTokenID, DBRefreshToken.ID)
		utils.WriteJSON(w, http.StatusForbidden, utils.NewMessage(constants.ForbiddenMessage, "", ""))
		return
	}

	refreshTokenTimeUntilExpiry := DBRefreshToken.CreatedAt.Add(constants.RefreshTokenTime).Unix()

	accessToken, refreshToken, err := utils.CreateTokens(userID, userName, refreshTokenTimeUntilExpiry)

	result, err = utils.ExtractClaimsFromJWT(refreshToken, []string{constants.JWTRefreshTokenID})
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
