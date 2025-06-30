package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

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

	handler.Logger.Printf("page: %v, pagesize %v", page, pageSize)

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
	userID, _, err := utils.ValidateTokensFromCookies(r)
	if err != nil {
		handler.Logger.Printf("ERROR: PostChallenge > JWT token checking: %v", err)
		utils.WriteJSON(w, http.StatusUnauthorized, utils.NewMessage(constants.UnauthorizedMessage, constants.MSG_LACKING_MANDATORY_FIELDS, "cookies"))

		return
	}

	var req store.PostChallengeRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	err = decoder.Decode(&req)
	if err != nil {
		handler.Logger.Printf("ERROR: PostChallenge > json encoding: %v", err)
		utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage(constants.StatusInvalidJSONMessage, constants.MSG_MALFORMED_REQUEST_DATA, "request"))
		return
	}

	if req.Content == "" || req.Category == "" || req.Name == "" {
		utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage("content, category, name must exist", constants.MSG_LACKING_MANDATORY_FIELDS, "content and category and name"))
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
