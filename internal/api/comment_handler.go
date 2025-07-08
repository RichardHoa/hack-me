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

	if req.ChallengeID != "" && req.ChallengeResponseID != "" {
		utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage("challengeID and challengeResponseID cannot be present at the same time", constants.MSG_MALFORMED_REQUEST_DATA, "challengeID, challengeResponseID"))
		return
	}

	trimmedContent := strings.TrimSpace(req.Content)
	if trimmedContent == "" {
		utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage("content cannot be empty", constants.MSG_INVALID_REQUEST_DATA, "content"))
		return
	}

	err = handler.Store.PostComment(req)
	if err != nil {
		switch utils.ClassifyError(err) {
		default:
			handler.Logger.Printf("ERROR: PostChallengeResponseVote > PostVote error: %v", err)
			utils.WriteJSON(w, http.StatusInternalServerError, utils.NewMessage(constants.StatusInternalErrorMessage, "", ""))
			return
		}
	}

	utils.WriteJSON(w, http.StatusCreated, utils.NewMessage("comment has been created", "", ""))

}
