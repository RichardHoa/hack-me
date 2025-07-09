package api

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/RichardHoa/hack-me/internal/constants"
	"github.com/RichardHoa/hack-me/internal/store"
	"github.com/RichardHoa/hack-me/internal/utils"
)

type ChallengeResponseHandler struct {
	ChallengeResponseStore store.ChallengeResponseStore
	Logger                 *log.Logger
}

func NewChallengeResponseHandler(store store.ChallengeResponseStore, logger *log.Logger) *ChallengeResponseHandler {
	return &ChallengeResponseHandler{
		ChallengeResponseStore: store,
		Logger:                 logger,
	}
}

func (handler *ChallengeResponseHandler) PostChallengeResponse(w http.ResponseWriter, r *http.Request) {
	userID, _, err := utils.ValidateTokensFromCookies(r)
	if err != nil {
		handler.Logger.Printf("ERROR: PostChallengeResponse > JWT token checking: %v", err)
		utils.WriteJSON(w, http.StatusUnauthorized, utils.NewMessage(constants.UnauthorizedMessage, constants.MSG_LACKING_MANDATORY_FIELDS, "cookies"))
		return
	}

	var req store.PostChallengeResponseRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	err = decoder.Decode(&req)
	if err != nil {
		handler.Logger.Printf("ERROR: PostChallengeResponse > json encoding: %v", err)
		utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage(constants.StatusInvalidJSONMessage, constants.MSG_MALFORMED_REQUEST_DATA, "request"))
		return
	}

	req.UserID = userID

	err = utils.ValidateJSONFieldsNotEmpty(w, req)
	if err != nil {
		return
	}

	err = handler.ChallengeResponseStore.PostResponse(req)
	if err != nil {
		switch utils.ClassifyError(err) {
		case constants.PQUniqueViolation:
			utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage("You already make a response to this challenge", constants.MSG_INVALID_REQUEST_DATA, "challengeID"))
			return
		default:
			handler.Logger.Printf("ERROR: PostChallengeResponse > store PostResponse: %v", err)
			utils.WriteJSON(w, http.StatusInternalServerError, utils.NewMessage(constants.StatusInternalErrorMessage, "", ""))
			return
		}

	}

	utils.WriteJSON(w, http.StatusCreated, utils.NewMessage("New challenge response created successfully", "", ""))
}

func (handler *ChallengeResponseHandler) ModifyChallengeResponse(w http.ResponseWriter, r *http.Request) {
	userID, _, err := utils.ValidateTokensFromCookies(r)
	if err != nil {
		handler.Logger.Printf("ERROR: ModifyChallengeResponse > JWT token checking: %v", err)
		utils.WriteJSON(w, http.StatusUnauthorized, utils.NewMessage(constants.UnauthorizedMessage, constants.MSG_LACKING_MANDATORY_FIELDS, "cookies"))
		return
	}

	var req store.PutChallengeResponseRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	err = decoder.Decode(&req)
	if err != nil {
		handler.Logger.Printf("ERROR: ModifyChallengeResponse > json encoding: %v", err)
		utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage(constants.StatusInvalidJSONMessage, constants.MSG_MALFORMED_REQUEST_DATA, "request"))
		return
	}

	req.UserID = userID

	if req.ChallengeResponseID == "" {
		utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage("challengeResponseID must exist", constants.MSG_LACKING_MANDATORY_FIELDS, "challengeResponseID"))
	}

	if req.Name == "" || req.Content == "" {
		utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage("name or content field must exist", constants.MSG_LACKING_MANDATORY_FIELDS, "name, content"))
	}

	err = handler.ChallengeResponseStore.ModifyResponse(req)
	if err != nil {
		switch utils.ClassifyError(err) {
		case constants.InvalidData:
			utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage(err.Error(), constants.MSG_INVALID_REQUEST_DATA, "unknown"))
			return
		case constants.LackingPermission:
			utils.WriteJSON(w, http.StatusForbidden, utils.NewMessage(err.Error(), constants.MSG_INVALID_REQUEST_DATA, ""))
			return
		default:
			handler.Logger.Printf("ERROR: ModifyChallengeResponse > store PostResponse: %v", err)
			utils.WriteJSON(w, http.StatusInternalServerError, utils.NewMessage(constants.StatusInternalErrorMessage, "", ""))
			return
		}

	}

	utils.WriteJSON(w, http.StatusOK, utils.NewMessage("Modify challenge successfully", "", ""))
}

func (handler *ChallengeResponseHandler) DeleteChallengeResponse(w http.ResponseWriter, r *http.Request) {
	userID, _, err := utils.ValidateTokensFromCookies(r)
	if err != nil {
		handler.Logger.Printf("ERROR: DeleteChallengeResponse > JWT token checking: %v", err)
		utils.WriteJSON(w, http.StatusUnauthorized, utils.NewMessage(constants.UnauthorizedMessage, constants.MSG_LACKING_MANDATORY_FIELDS, "cookies"))
		return
	}

	var req store.DeleteChallengeResponseRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	err = decoder.Decode(&req)
	if err != nil {
		handler.Logger.Printf("ERROR: DeleteChallengeResponse > json encoding: %v", err)
		utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage(constants.StatusInvalidJSONMessage, constants.MSG_MALFORMED_REQUEST_DATA, "request"))
		return
	}

	req.UserID = userID

	err = utils.ValidateJSONFieldsNotEmpty(w, req)
	if err != nil {
		return
	}

	err = handler.ChallengeResponseStore.DeleteResponse(req)
	if err != nil {
		switch utils.ClassifyError(err) {
		case constants.InvalidData:
			utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage(err.Error(), constants.MSG_INVALID_REQUEST_DATA, "unknown"))
			return
		case constants.LackingPermission:
			utils.WriteJSON(w, http.StatusForbidden, utils.NewMessage(err.Error(), constants.MSG_INVALID_REQUEST_DATA, "challengeID"))
			return
		default:
			handler.Logger.Printf("ERROR: DeleteChallengeResponse > store DeleteResponse: %v", err)
			utils.WriteJSON(w, http.StatusInternalServerError, utils.NewMessage(constants.StatusInternalErrorMessage, "", ""))
			return
		}

	}

	utils.WriteJSON(w, http.StatusOK, utils.NewMessage("Challenge response deleted successfully", "", ""))
}

func (handler *ChallengeResponseHandler) GetChallengeResponse(w http.ResponseWriter, r *http.Request) {

	var req store.GetChallengeResponseRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	err := decoder.Decode(&req)
	if err != nil {
		handler.Logger.Printf("ERROR: GetChallengeResponse > json encoding: %v", err)
		utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage(constants.StatusInvalidJSONMessage, constants.MSG_MALFORMED_REQUEST_DATA, "request"))
		return
	}

	responses, err := handler.ChallengeResponseStore.GetResponses(req)
	if err != nil {
		switch utils.ClassifyError(err) {
		default:
			handler.Logger.Printf("ERROR: GetChallengeResponse > store GetResponses: %v", err)
			utils.WriteJSON(w, http.StatusInternalServerError, utils.NewMessage(constants.StatusInternalErrorMessage, "", ""))
			return
		}
	}

	utils.WriteJSON(w, http.StatusOK, utils.Message{
		"data": responses,
	})
	return

}
