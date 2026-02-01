package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/RichardHoa/hack-me/internal/constants"
	"github.com/RichardHoa/hack-me/internal/domains"
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
	nameDTO := query.Get("name")
	exactNameDTO := query.Get("exactName")
	pageSize := query.Get("pageSize")
	page := query.Get("page")

	getChallengeParams := store.GetChallengeParams{}

	if nameDTO != "" {
		name, err := domains.NewChallengeName(nameDTO)
		if err != nil {
			utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage(err.Error(), constants.MSG_INVALID_REQUEST_DATA, "query"))
			return
		}
		getChallengeParams.Name = &name
	}

	if exactNameDTO != "" {
		exactName, err := domains.NewChallengeName(exactNameDTO)
		if err != nil {
			utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage(err.Error(), constants.MSG_INVALID_REQUEST_DATA, "query"))
			return
		}
		getChallengeParams.ExactName = &exactName

	}

	if getChallengeParams.Name != nil && getChallengeParams.ExactName != nil {
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

		getChallengeParams.Page = &pageNum
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

		getChallengeParams.PageSize = &pageSizeNum
	}

	if popularity != "" {
		getChallengeParams.Popularity = &popularity
	}

	if len(categories) != 0 {
		getChallengeParams.Category = &categories
	}

	challenges, metaPage, err := handler.ChallengeStore.GetChallenges(getChallengeParams)

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
	result, err := utils.GetValuesFromCookie(r, []string{constants.JWTUserID})
	if err != nil {
		handler.Logger.Printf("ERROR: PostChallenge > JWT token checking: %v", err)
		utils.WriteJSON(w, http.StatusUnauthorized, utils.NewMessage(constants.UnauthorizedMessage, constants.MSG_LACKING_MANDATORY_FIELDS, ""))

		return
	}

	userID := result[0]

	var dto store.PostChallengeRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	err = decoder.Decode(&dto)
	if err != nil {
		handler.Logger.Printf("ERROR: PostChallenge > jsonDecoding: %v", err)
		utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage(constants.StatusInvalidBodyMessage, constants.MSG_MALFORMED_REQUEST_DATA, "request"))
		return
	}

	err = utils.ValidateJSONFieldsNotEmpty(w, dto)
	if err != nil {
		return
	}

	challengeName, err := domains.NewChallengeName(dto.Name)
	if err != nil {
		utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage(err.Error(), constants.MSG_MALFORMED_REQUEST_DATA, "request"))
		return
	}

	postChallengeParams := store.NewPostChallengeParams(userID, challengeName, dto.Content, dto.Category)

	err = handler.ChallengeStore.CreateChallenges(postChallengeParams)
	if err != nil {
		switch utils.ClassifyError(err) {
		case constants.PQUniqueViolation:
			handler.Logger.Printf("ERROR: postchallenge > store createchallenges: User ID: %v try to add challenge name that already exist\n", postChallengeParams.UserID())
			utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage("Challenge name already exist", constants.MSG_INVALID_REQUEST_DATA, "name"))
			return
		case constants.PQCheckViolation:
			utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage("Name must be below 1000 characters", constants.MSG_INVALID_REQUEST_DATA, "name"))
			return
		case constants.PQForeignKeyViolation:
			handler.Logger.Printf("ERROR: postchallenge > store createchallenges: Invalid User ID: %v", postChallengeParams.UserID())
			utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage(constants.UnauthorizedMessage, constants.MSG_LACKING_MANDATORY_FIELDS, "cookies"))
			return
		case constants.PQInvalidTextRepresentation:
			handler.Logger.Printf("ERROR: postchallenge > store createchallenges: Invalid Data format: %v", err)
			utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage("category data is invalid", constants.MSG_INVALID_REQUEST_DATA, "category"))
			return
		default:
			handler.Logger.Printf("ERROR: postchallenge > store createchallenges: %v", err)
			utils.WriteJSON(w, http.StatusInternalServerError, utils.NewMessage(constants.StatusInternalErrorMessage, "", ""))
			return
		}

	}

	utils.WriteJSON(w, http.StatusCreated, utils.NewMessage("Post challenge successfully", "", ""))
}

func (handler *ChallengeHandler) DeleteChallege(w http.ResponseWriter, r *http.Request) {
	result, err := utils.GetValuesFromCookie(r, []string{constants.JWTUserID})

	if err != nil {
		handler.Logger.Printf("ERROR: DeleteChallenge > JWT token checking: %v", err)
		utils.WriteJSON(w, http.StatusUnauthorized, utils.NewMessage(constants.UnauthorizedMessage, constants.MSG_LACKING_MANDATORY_FIELDS, ""))
		return
	}
	userID := result[0]

	var dto store.DeleteChallengeRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	err = decoder.Decode(&dto)
	if err != nil {
		handler.Logger.Printf("ERROR: DeleteChallenge > jsonDecoding: %v", err)
		utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage(constants.StatusInvalidBodyMessage, constants.MSG_MALFORMED_REQUEST_DATA, "request"))
		return
	}

	challengeName, err := domains.NewChallengeName(dto.Name)
	if err != nil {
		utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage(err.Error(), constants.MSG_MALFORMED_REQUEST_DATA, "request"))
		return
	}

	deleteChallengeParams := store.DeleteChallengeParams{
		ChallengeName: challengeName,
		UserID:        userID,
	}

	err = handler.ChallengeStore.DeleteChallenge(deleteChallengeParams)
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

	result, err := utils.GetValuesFromCookie(r, []string{constants.JWTUserID})
	if err != nil {
		handler.Logger.Printf("ERROR: ModifyChallenge > JWT token checking: %v", err)
		utils.WriteJSON(w, http.StatusUnauthorized, utils.NewMessage(constants.UnauthorizedMessage, constants.MSG_LACKING_MANDATORY_FIELDS, ""))
		return
	}
	userID := result[0]

	var dto store.ModifyChallengeRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	err = decoder.Decode(&dto)
	if err != nil {
		handler.Logger.Printf("ERROR: ModifyChallenge > jsonDecoding: %v", err)
		utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage(constants.StatusInvalidBodyMessage, constants.MSG_MALFORMED_REQUEST_DATA, "request"))
		return
	}

	oldName, err := domains.NewChallengeName(dto.OldName)
	if err != nil {
		utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage(err.Error(), constants.MSG_LACKING_MANDATORY_FIELDS, "oldName"))
		return
	}

	modifyChallengeParams := store.ModifyChallengeParams{
		UserID:  userID,
		OldName: oldName,
	}

	if dto.NewName != "" {
		newName, err := domains.NewChallengeName(dto.NewName)
		if err != nil {
			utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage(err.Error(), constants.MSG_LACKING_MANDATORY_FIELDS, "oldName"))
			return
		}
		modifyChallengeParams.NewName = &newName
	}

	if dto.Category != "" {
		modifyChallengeParams.Category = &dto.Category
	}

	if dto.Content != "" {
		modifyChallengeParams.Content = &dto.Content
	}

	err = handler.ChallengeStore.ModifyChallenge(modifyChallengeParams)
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
