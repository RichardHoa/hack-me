package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/RichardHoa/hack-me/internal/constants"
	"github.com/RichardHoa/hack-me/internal/store"
	"github.com/RichardHoa/hack-me/internal/utils"
)

type ChallengeHandler struct {
	ChallengeStore store.ChallengeStore
	Logger         *log.Logger
}

func NewChallengeHandler(challengeStore store.ChallengeStore, logger *log.Logger) *ChallengeHandler {
	return &ChallengeHandler{
		ChallengeStore: challengeStore,
		Logger:         logger,
	}
}

func (handler *ChallengeHandler) GetChallenges(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	popularity := query.Get("popularity")
	categories := query["category"]
	name := query.Get("name")
	exactName := query.Get("exactName")
	pageSize := query.Get("pageSize")
	page := query.Get("page")

	trimmedName := strings.TrimSpace(name)
	trimmedExactName := strings.TrimSpace(exactName)

	if trimmedName != "" && trimmedExactName != "" {
		utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage("name and exactName cannot both be present", constants.MSG_INVALID_REQUEST_DATA, "query"))
		return
	}

	if page != "" {
		pageNum, err := strconv.Atoi(page)
		if err != nil {
			utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage("page can only be number", constants.MSG_MALFORMED_REQUEST_DATA, "page"))
			return
		}
		if pageNum <= 0 {
			utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage("page cannot be negative or 0", constants.MSG_MALFORMED_REQUEST_DATA, "page"))
			return
		}
	}

	if pageSize != "" {
		pageSizeNum, err := strconv.Atoi(pageSize)
		if err != nil {
			utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage("pageSize can only be number", constants.MSG_MALFORMED_REQUEST_DATA, "pageSize"))
			return
		}

		if pageSizeNum <= 0 {
			utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage("pageSize cannot be negative or 0", constants.MSG_MALFORMED_REQUEST_DATA, "pageSize"))
			return
		}
	}

	freeQuery := store.ChallengeFreeQuery{
		Popularity: popularity,
		Category:   categories,
		Name:       name,
		ExactName:  exactName,
		PageSize:   pageSize,
		Page:       page,
	}

	challenges, metaPage, err := handler.ChallengeStore.GetChallenges(freeQuery)

	if err != nil {
		handler.Logger.Printf("ERROR: GetChallenges -> storeGetChallenges: %v", err)
		switch utils.ClassifyError(err) {
		// invalid category query
		case constants.PQInvalidTextRepresentation:
			utils.WriteJSON(w, http.StatusOK, utils.Message{
				"metadata": metaPage,
				"data":     challenges,
			})
			return
		default:
			utils.WriteJSON(w, http.StatusInternalServerError, utils.NewMessage(constants.StatusInternalErrorMessage, "", ""))
			return
		}
	}

	utils.WriteJSON(w, http.StatusOK, utils.Message{
		"metadata": metaPage,
		"data":     challenges,
	})
}

func (handler *ChallengeHandler) PostChallenge(w http.ResponseWriter, r *http.Request) {
	result, err := utils.GetValuesFromCookie(r, []string{constants.TokenUserID})
	if err != nil {
		handler.Logger.Printf("ERROR: PostChallenge > JWT token checking: %v", err)
		utils.WriteJSON(w, http.StatusUnauthorized, utils.NewMessage(constants.UnauthorizedMessage, constants.MSG_LACKING_MANDATORY_FIELDS, ""))

		return
	}

	userID := result[0]

	var req store.PostChallengeRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	err = decoder.Decode(&req)
	if err != nil {
		handler.Logger.Printf("ERROR: PostChallenge > json encoding: %v", err)
		utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage(constants.StatusInvalidJSONMessage, constants.MSG_MALFORMED_REQUEST_DATA, "request"))
		return
	}

	err = utils.ValidateJSONFieldsNotEmpty(w, req)
	if err != nil {
		return
	}

	nameLength := utf8.RuneCountInString(req.Name)

	if nameLength > constants.MaxLengthName {
		utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage(
			fmt.Sprintf("Challenge name has the maximum of %d characters, your name %d characters", constants.MaxLengthName, nameLength),
			constants.MSG_INVALID_REQUEST_DATA,
			"name"))
		return
	}

	if strings.Contains(req.Name, "#") {
		utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage("You can't have symbol # in challenge name", constants.MSG_INVALID_REQUEST_DATA, "name"))
		return
	}

	challenge := store.Challenge{
		UserID:   userID,
		Name:     req.Name,
		Content:  req.Content,
		Category: req.Category,
	}

	err = handler.ChallengeStore.CreateChallenges(&challenge)
	if err != nil {
		switch utils.ClassifyError(err) {
		case constants.PQUniqueViolation:
			handler.Logger.Printf("User ID: %v try to add challenge name that already exist\n", challenge.UserID)
			utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage("Challenge name already exist", constants.MSG_INVALID_REQUEST_DATA, "name"))
			return
		case constants.PQCheckViolation:
			utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage("Name must be at least 3 characters long with no leading or trailing whitespace", constants.MSG_INVALID_REQUEST_DATA, "name"))
			return
		case constants.PQForeignKeyViolation:
			handler.Logger.Printf("ERROR: Invalid User ID: %v", challenge.UserID)
			utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage(constants.UnauthorizedMessage, constants.MSG_LACKING_MANDATORY_FIELDS, "cookies"))
			return
		case constants.PQInvalidTextRepresentation:
			handler.Logger.Printf("ERROR: Invalid Data format: %v", err)
			utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage("category data is invalid", constants.MSG_INVALID_REQUEST_DATA, "category"))
			return
		default:
			handler.Logger.Printf("ERROR: PostChallenge > store CreateChallenges: %v", err)
			utils.WriteJSON(w, http.StatusInternalServerError, utils.NewMessage(constants.StatusInternalErrorMessage, "", ""))
			return
		}

	}

	utils.WriteJSON(w, http.StatusCreated, utils.NewMessage("Post challenge successfully", "", ""))
}

func (handler *ChallengeHandler) DeleteChallege(w http.ResponseWriter, r *http.Request) {
	result, err := utils.GetValuesFromCookie(r, []string{constants.TokenUserID})

	if err != nil {
		handler.Logger.Printf("ERROR: DeleteChallenge > JWT token checking: %v", err)
		utils.WriteJSON(w, http.StatusUnauthorized, utils.NewMessage(constants.UnauthorizedMessage, constants.MSG_LACKING_MANDATORY_FIELDS, ""))
		return
	}
	userID := result[0]

	var req store.DeleteChallengeRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	err = decoder.Decode(&req)
	if err != nil {
		handler.Logger.Printf("ERROR: DeleteChallenge > json encoding: %v", err)
		utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage(constants.StatusInvalidJSONMessage, constants.MSG_MALFORMED_REQUEST_DATA, "request"))
		return
	}

	err = utils.ValidateJSONFieldsNotEmpty(w, req)
	if err != nil {
		return
	}

	err = handler.ChallengeStore.DeleteChallenge(req.Name, userID)
	if err != nil {
		handler.Logger.Printf("ERROR: DeleteChallenge > store Delete challenge: %v", err)
		switch utils.ClassifyError(err) {

		case constants.InvalidData:
			utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage(constants.UnauthorizedMessage, constants.MSG_INVALID_REQUEST_DATA, ""))

		case constants.LackingPermission:
			utils.WriteJSON(w, http.StatusForbidden, utils.NewMessage(constants.UnauthorizedMessage, constants.MSG_INVALID_REQUEST_DATA, ""))
			return
		default:
			utils.WriteJSON(w, http.StatusInternalServerError, utils.NewMessage(constants.StatusInternalErrorMessage, "", ""))
			return
		}
	}

	utils.WriteJSON(w, http.StatusOK, utils.NewMessage("Challenge deleted", "", ""))

}

func (handler *ChallengeHandler) ModifyChallenge(w http.ResponseWriter, r *http.Request) {

	result, err := utils.GetValuesFromCookie(r, []string{constants.TokenUserID})
	if err != nil {
		handler.Logger.Printf("ERROR: ModifyChallenge > JWT token checking: %v", err)
		utils.WriteJSON(w, http.StatusUnauthorized, utils.NewMessage(constants.UnauthorizedMessage, constants.MSG_LACKING_MANDATORY_FIELDS, ""))
		return
	}
	userID := result[0]

	var req store.PutChallengeRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	err = decoder.Decode(&req)
	if err != nil {
		handler.Logger.Printf("ERROR: ModifyChallenge > json encoding: %v", err)
		utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage(constants.StatusInvalidJSONMessage, constants.MSG_MALFORMED_REQUEST_DATA, "request"))
		return
	}

	req.UserID = userID

	if req.OldName == "" {
		utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage("Previous challenge name must be provided", constants.MSG_LACKING_MANDATORY_FIELDS, "oldName"))
		return
	}

	if req.Name == "" && req.Category == "" && req.Content == "" {
		utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage("One of the three fields must exist", constants.MSG_LACKING_MANDATORY_FIELDS, "name, category, content"))
		return
	}

	err = handler.ChallengeStore.ModifyChallenge(req)
	if err != nil {
		handler.Logger.Printf("ERROR: ModifyChallenge > store modify challenge: %v", err)
		switch utils.ClassifyError(err) {
		case constants.InvalidData:
			utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage(err.Error(), constants.MSG_INVALID_REQUEST_DATA, "name"))
			return
		case constants.PQInvalidTextRepresentation:
			utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage("invalid category value", constants.MSG_INVALID_REQUEST_DATA, "category"))
			return
		case constants.PQUniqueViolation:
			utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage("new challenge name already exist", constants.MSG_INVALID_REQUEST_DATA, "name"))
			return
		case constants.LackingPermission:
			utils.WriteJSON(w, http.StatusForbidden, utils.NewMessage(constants.UnauthorizedMessage, constants.MSG_INVALID_REQUEST_DATA, ""))
			return
		default:
			utils.WriteJSON(w, http.StatusInternalServerError, utils.NewMessage(constants.StatusInternalErrorMessage, "", ""))
			return
		}
	}

	utils.WriteJSON(w, http.StatusOK, utils.NewMessage("Challenge has been updated successfully", "", ""))

}
