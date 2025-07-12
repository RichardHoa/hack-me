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

type CommentHandler struct {
	Store  store.CommentStore
	Logger *log.Logger
}

func NewCommentHandler(store store.CommentStore, logger *log.Logger) *CommentHandler {
	return &CommentHandler{Store: store, Logger: logger}
}

func (handler *CommentHandler) PostComment(w http.ResponseWriter, r *http.Request) {

	userID, _, err := utils.ValidateTokensFromCookies(r)
	if err != nil {
		handler.Logger.Printf("ERROR: PostComment > JWT token checking: %v", err)
		utils.WriteJSON(w, http.StatusUnauthorized, utils.NewMessage(constants.UnauthorizedMessage, constants.MSG_LACKING_MANDATORY_FIELDS, "cookies"))
		return
	}

	var req store.PostCommentRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	err = decoder.Decode(&req)
	if err != nil {
		handler.Logger.Printf("ERROR: PostComment > json encoding: %v", err)
		utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage(constants.StatusInvalidJSONMessage, constants.MSG_MALFORMED_REQUEST_DATA, "request"))
		return
	}

	req.UserID = userID

	if err := validateCommentRequest(req); err != nil {
		utils.WriteJSON(w, http.StatusBadRequest, err)
		return
	}

	commentID, err := handler.Store.PostComment(req)
	if err != nil {
		switch utils.ClassifyError(err) {
		case constants.PQForeignKeyViolation:
			utils.WriteJSON(w, http.StatusNotFound, utils.NewMessage("invalid ID", constants.
				MSG_MALFORMED_REQUEST_DATA, "unknown"))
			return
		case constants.PQInvalidTextRepresentation:
			utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage("invalid data type", constants.
				MSG_MALFORMED_REQUEST_DATA, "request body"))
			return
		default:
			handler.Logger.Printf("ERROR: PostChallengeResponseVote > PostComment error: %v", err)
			utils.WriteJSON(w, http.StatusInternalServerError, utils.NewMessage(constants.StatusInternalErrorMessage, "", ""))
			return
		}
	}
	data := map[string]any{
		"commentID": commentID,
	}

	utils.WriteJSON(w, http.StatusCreated, utils.Message{"data": data, "message": "Post comment successfully"})

}

func (handler *CommentHandler) ModifyComment(w http.ResponseWriter, r *http.Request) {

	userID, _, err := utils.ValidateTokensFromCookies(r)
	if err != nil {
		handler.Logger.Printf("ERROR: ModifyComment > JWT token checking: %v", err)
		utils.WriteJSON(w, http.StatusUnauthorized, utils.NewMessage(constants.UnauthorizedMessage, constants.MSG_LACKING_MANDATORY_FIELDS, "cookies"))
		return
	}

	var req store.ModifyCommentRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	err = decoder.Decode(&req)
	if err != nil {
		handler.Logger.Printf("ERROR: ModifyComment > json encoding: %v", err)
		utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage(constants.StatusInvalidJSONMessage, constants.MSG_MALFORMED_REQUEST_DATA, "request"))
		return
	}

	err = utils.ValidateJSONFieldsNotEmpty(w, req)
	if err != nil {
		return
	}

	req.UserID = userID

	err = handler.Store.ModifyComment(req)
	if err != nil {
		switch utils.ClassifyError(err) {

		case constants.LackingPermission:
			utils.WriteJSON(w, http.StatusForbidden, utils.NewMessage(err.Error(), constants.MSG_INVALID_REQUEST_DATA, "commentId"))
			return
		case constants.InvalidData:
			utils.WriteJSON(w, http.StatusNotFound, utils.NewMessage(err.Error(), constants.MSG_INVALID_REQUEST_DATA, "commentId"))
			return
		case constants.PQInvalidTextRepresentation:
			utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage("invalid data type", constants.
				MSG_MALFORMED_REQUEST_DATA, "request body"))
			return
		default:
			handler.Logger.Printf("ERROR: ModifyChallengeResponseVote > ModifyComment error: %v", err)
			utils.WriteJSON(w, http.StatusInternalServerError, utils.NewMessage(constants.StatusInternalErrorMessage, "", ""))
			return
		}
	}

	utils.WriteJSON(w, http.StatusOK, utils.NewMessage("comment has been modified", "", ""))

}

func (handler *CommentHandler) DeleteComment(w http.ResponseWriter, r *http.Request) {

	userID, _, err := utils.ValidateTokensFromCookies(r)
	if err != nil {
		handler.Logger.Printf("ERROR: DeleteComment > JWT token checking: %v", err)
		utils.WriteJSON(w, http.StatusUnauthorized, utils.NewMessage(constants.UnauthorizedMessage, constants.MSG_LACKING_MANDATORY_FIELDS, "cookies"))
		return
	}

	var req store.DeleteCommentRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	err = decoder.Decode(&req)
	if err != nil {
		handler.Logger.Printf("ERROR: DeleteComment > json encoding: %v", err)
		utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage(constants.StatusInvalidJSONMessage, constants.MSG_MALFORMED_REQUEST_DATA, "request"))
		return
	}

	err = utils.ValidateJSONFieldsNotEmpty(w, req)
	if err != nil {
		return
	}

	req.UserID = userID

	err = handler.Store.DeleteComment(req)
	if err != nil {
		switch utils.ClassifyError(err) {
		case constants.LackingPermission:
			utils.WriteJSON(w, http.StatusForbidden, utils.NewMessage(err.Error(), constants.MSG_INVALID_REQUEST_DATA, "commentId"))
			return
		case constants.InvalidData:
			utils.WriteJSON(w, http.StatusNotFound, utils.NewMessage(err.Error(), constants.MSG_INVALID_REQUEST_DATA, "commentId"))
			return
		case constants.PQInvalidTextRepresentation:
			utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage("invalid data type", constants.
				MSG_MALFORMED_REQUEST_DATA, "commentID"))
			return
		default:
			handler.Logger.Printf("ERROR: DeleteChallengeResponseVote > DeleteComment error: %v", err)
			utils.WriteJSON(w, http.StatusInternalServerError, utils.NewMessage(constants.StatusInternalErrorMessage, "", ""))
			return
		}
	}

	utils.WriteJSON(w, http.StatusOK, utils.NewMessage("comment has been deleted", "", ""))

}

func validateCommentRequest(req store.PostCommentRequest) utils.Message {
	trimmedContent := strings.TrimSpace(req.Content)
	hasContent := trimmedContent != ""

	hasChallengeID := strings.TrimSpace(req.ChallengeID) != ""
	hasChallengeResponseID := strings.TrimSpace(req.ChallengeResponseID) != ""
	hasParentID := strings.TrimSpace(req.ParentID) != ""

	if !hasContent {
		return utils.NewMessage("content must be provided", constants.MSG_LACKING_MANDATORY_FIELDS, "content")
	}

	if !hasChallengeID && !hasChallengeResponseID {
		return utils.NewMessage("either challengeID or challengeResponseID must be provided", constants.MSG_LACKING_MANDATORY_FIELDS, "challengeID, challengeResponseID")
	}

	if hasChallengeID && hasChallengeResponseID {
		return utils.NewMessage("challengeID and challengeResponseID cannot be present at the same time", constants.MSG_MALFORMED_REQUEST_DATA, "challengeID, challengeResponseID")
	}

	if hasParentID {
		if !(hasChallengeID != hasChallengeResponseID) {
			return utils.NewMessage("parentID requires exactly one of challengeID or challengeResponseID", constants.MSG_MALFORMED_REQUEST_DATA, "parentID, challengeID, challengeResponseID")
		}
	}

	return nil
}
