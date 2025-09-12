package api

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/RichardHoa/hack-me/internal/constants"
	"github.com/RichardHoa/hack-me/internal/store"
	"github.com/RichardHoa/hack-me/internal/utils"
)

type ChallengeResponseVoteHandler struct {
	Logger *log.Logger
	Store  store.VoteStore
}

func NewChallengeResponseVoteHandler(store store.VoteStore, logger *log.Logger) *ChallengeResponseVoteHandler {
	return &ChallengeResponseVoteHandler{Logger: logger, Store: store}
}

func (handler *ChallengeResponseVoteHandler) PostVote(w http.ResponseWriter, r *http.Request) {

	result, err := utils.GetValuesFromCookie(r, []string{constants.TokenUserID})
	if err != nil {
		handler.Logger.Printf("ERROR: PostChallengeResponseVote > JWT token checking: %v", err)
		utils.WriteJSON(w, http.StatusUnauthorized, utils.NewMessage(constants.UnauthorizedMessage, constants.MSG_LACKING_MANDATORY_FIELDS, ""))
		return
	}

	userID := result[0]

	var req store.PostVoteRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	err = decoder.Decode(&req)
	if err != nil {
		handler.Logger.Printf("ERROR: PostChallengeResponseVote > json encoding: %v", err)
		utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage(constants.StatusInvalidJSONMessage, constants.MSG_MALFORMED_REQUEST_DATA, "request"))
		return
	}

	req.UserID = userID

	err = utils.ValidateJSONFieldsNotEmpty(w, req)
	if err != nil {
		return
	}

	if req.VoteType != "upVote" && req.VoteType != "downVote" {

		utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage("voteType must be upVote OR downVote", constants.MSG_MALFORMED_REQUEST_DATA, "voteType"))
		return

	}

	err = handler.Store.PostVote(req)
	if err != nil {
		switch utils.ClassifyError(err) {
		case constants.InvalidData:
			utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage(err.Error(), "", ""))
			return
		case constants.PQForeignKeyViolation:
			utils.WriteJSON(w, http.StatusNotFound, utils.NewMessage("challengeResponseID not found", constants.MSG_INVALID_REQUEST_DATA, "challengeResponseID"))
			return
		case constants.PQInvalidTextRepresentation:
			utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage("invalid data type", constants.MSG_MALFORMED_REQUEST_DATA, "request body"))
			return
		default:
			handler.Logger.Printf("ERROR: PostChallengeResponseVote > PostVote error: %v", err)
			utils.WriteJSON(w, http.StatusInternalServerError, utils.NewMessage(constants.StatusInternalErrorMessage, "", ""))
			return
		}
	}

	utils.WriteJSON(w, http.StatusCreated, utils.NewMessage("new vote has been created", "", ""))
}

func (handler *ChallengeResponseVoteHandler) DeleteVote(w http.ResponseWriter, r *http.Request) {

	result, err := utils.GetValuesFromCookie(r, []string{constants.TokenUserID})
	if err != nil {
		handler.Logger.Printf("ERROR: DeleteChallengeResponseVote > JWT token checking: %v", err)
		utils.WriteJSON(w, http.StatusUnauthorized, utils.NewMessage(constants.UnauthorizedMessage, constants.MSG_LACKING_MANDATORY_FIELDS, ""))
		return
	}

	userID := result[0]

	var req store.DeleteVoteRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	err = decoder.Decode(&req)
	if err != nil {
		handler.Logger.Printf("ERROR: DeleteChallengeResponseVote > json encoding: %v", err)
		utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage(constants.StatusInvalidJSONMessage, constants.MSG_MALFORMED_REQUEST_DATA, "request"))
		return
	}

	req.UserID = userID

	err = utils.ValidateJSONFieldsNotEmpty(w, req)
	if err != nil {
		return
	}

	err = handler.Store.DeleteVote(req)
	if err != nil {
		handler.Logger.Printf("ERROR: DeleteChallengeResponseVote > DeleteVote error: %v", err)
		switch utils.ClassifyError(err) {
		case constants.PQInvalidTextRepresentation:
			utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage("invalid data type", constants.MSG_MALFORMED_REQUEST_DATA, "challengeResponseID"))
			return
		case constants.InvalidData:
			utils.WriteJSON(w, http.StatusNotFound, utils.NewMessage(err.Error(), constants.MSG_INVALID_REQUEST_DATA, "challengeResponseID"))
			return
		default:
			utils.WriteJSON(w, http.StatusInternalServerError, utils.NewMessage(constants.StatusInternalErrorMessage, "", ""))
			return
		}
	}

	utils.WriteJSON(w, http.StatusOK, utils.NewMessage("vote has been deleted", "", ""))
}
