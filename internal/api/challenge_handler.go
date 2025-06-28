package api

import (
	"encoding/json"
	"log"
	"net/http"

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
	exactName := query.Get("exact-name")

	freeQuery := store.ChallengeFreeQuery{
		Popularity: popularity,
		Category:   categories,
		Name:       name,
		ExactName:  exactName,
	}

	challenges, err := handler.ChallengeStore.GetChallenges(freeQuery)
	if err != nil {
		switch utils.ClassifyPgError(err) {
		case constants.PQInvalidTextRepresentation:
			utils.WriteJSON(w, http.StatusOK, utils.Message{"data": nil})
			return
		default:
			handler.Logger.Printf("ERROR: GetChallenges: %v", err)
			utils.WriteJSON(w, http.StatusInternalServerError, utils.Message{"message": constants.StatusInternalErrorMessage})
			return
		}
	}

	utils.WriteJSON(w, http.StatusOK, utils.Message{"data": challenges})
}

func (handler *ChallengeHandler) PostChallenge(w http.ResponseWriter, r *http.Request) {
	userID, _, err := utils.ValidateTokensFromCookies(r)
	if err != nil {
		handler.Logger.Printf("ERROR: PostChallenge > JWT token checking: %v", err)
		utils.WriteJSON(w, http.StatusBadRequest, utils.Message{"message": "Unauthorized"})
		return
	}

	var req store.PostChallengeRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	err = decoder.Decode(&req)
	if err != nil {
		handler.Logger.Printf("ERROR: PostChallenge > json encoding: %v", err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Message{"message": constants.StatusInternalErrorMessage})
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
		switch utils.ClassifyPgError(err) {
		case constants.PQUniqueViolation:
			handler.Logger.Printf("User ID: %v try to add challenge name that already exist\n", challenge.UserID)
			utils.WriteJSON(w, http.StatusBadRequest, utils.Message{"message": "Challenge name already exists"})
			return
		case constants.PQForeignKeyViolation:
			handler.Logger.Printf("ERROR: Invalid User ID: %v", challenge.UserID)
			utils.WriteJSON(w, http.StatusBadRequest, utils.Message{"message": "Invalid data"})
			return
		case constants.PQInvalidTextRepresentation:
			handler.Logger.Printf("ERROR: Invalid Data format: %v", err)
			utils.WriteJSON(w, http.StatusBadRequest, utils.Message{"message": "Invalid data"})
			return
		default:
			handler.Logger.Printf("ERROR: PostChallenge > store CreateChallenges: %v", err)
			utils.WriteJSON(w, http.StatusBadRequest, utils.Message{"message": constants.StatusInternalErrorMessage})
			return
		}

	}

	utils.WriteJSON(w, http.StatusCreated, utils.Message{"message": "success posting"})
}
