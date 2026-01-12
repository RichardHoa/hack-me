package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

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
	result, err := utils.GetValuesFromCookie(r, []string{constants.JWTUserID})
	if err != nil {
		handler.Logger.Printf("ERROR: PostChallengeResponse > JWT token checking: %v", err)
		utils.WriteJSON(w, http.StatusUnauthorized, utils.NewMessage(constants.UnauthorizedMessage, constants.MSG_LACKING_MANDATORY_FIELDS, ""))
		return
	}
	userID := result[0]

	var req store.PostChallengeResponseRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	err = decoder.Decode(&req)
	if err != nil {
		handler.Logger.Printf("ERROR: PostChallengeResponse > jsonDecoding: %v", err)
		utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage(constants.StatusInvalidBodyMessage, constants.MSG_MALFORMED_REQUEST_DATA, "request"))
		return
	}

	req.UserID = userID

	err = utils.ValidateJSONFieldsNotEmpty(w, req)
	if err != nil {
		return
	}

	//BUG: We did not check if the response author is the author of the challenge

	challengeResponseID, err := handler.ChallengeResponseStore.PostResponse(req)
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

	data := map[string]any{
		"challengeResponseID": challengeResponseID,
	}

	utils.WriteJSON(w, http.StatusCreated, utils.Message{
		"message": "add challenge response successfully",
		"data":    data,
	})
}

func (handler *ChallengeResponseHandler) ModifyChallengeResponse(w http.ResponseWriter, r *http.Request) {
	result, err := utils.GetValuesFromCookie(r, []string{constants.JWTUserID})
	if err != nil {
		handler.Logger.Printf("ERROR: ModifyChallengeResponse > JWT token checking: %v", err)
		utils.WriteJSON(w, http.StatusUnauthorized, utils.NewMessage(constants.UnauthorizedMessage, constants.MSG_LACKING_MANDATORY_FIELDS, ""))
		return
	}
	userID := result[0]

	var req store.PutChallengeResponseRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	err = decoder.Decode(&req)
	if err != nil {
		handler.Logger.Printf("ERROR: ModifyChallengeResponse > jsonDecoding: %v", err)
		utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage(constants.StatusInvalidBodyMessage, constants.MSG_MALFORMED_REQUEST_DATA, "request"))
		return
	}

	req.UserID = userID

	if req.ChallengeResponseID == "" {
		utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage("challengeResponseID must exist", constants.MSG_LACKING_MANDATORY_FIELDS, "challengeResponseID"))
		return
	}

	if req.Name == "" || req.Content == "" {
		utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage("name or content field must exist", constants.MSG_LACKING_MANDATORY_FIELDS, "name, content"))
		return
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
	result, err := utils.GetValuesFromCookie(r, []string{constants.JWTUserID})
	if err != nil {
		handler.Logger.Printf("ERROR: DeleteChallengeResponse > JWT token checking: %v", err)
		utils.WriteJSON(w, http.StatusUnauthorized, utils.NewMessage(constants.UnauthorizedMessage, constants.MSG_LACKING_MANDATORY_FIELDS, ""))
		return
	}

	userID := result[0]

	var req store.DeleteChallengeResponseRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	err = decoder.Decode(&req)
	if err != nil {
		handler.Logger.Printf("ERROR: DeleteChallengeResponse > jsonDecoding: %v", err)
		utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage(constants.StatusInvalidBodyMessage, constants.MSG_MALFORMED_REQUEST_DATA, "request"))
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

	query := r.URL.Query()

	challengeID := query.Get("challengeID")
	challengeResponseID := query.Get("challengeResponseID")

	trimmedChallengeID := strings.TrimSpace(challengeID)
	trimmedChallengeResponseID := strings.TrimSpace(challengeResponseID)

	if trimmedChallengeID != "" && trimmedChallengeResponseID != "" {
		utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage("challengeID and challengeResponseID cannot be present at the same time", constants.MSG_MALFORMED_REQUEST_DATA, "query parameter"))
		return
	}
	req := store.GetChallengeResponseRequest{
		ChallengeID:         trimmedChallengeID,
		ChallengeResponseID: trimmedChallengeResponseID,
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

}
