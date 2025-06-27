package api

import (
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

	freeQuery := store.ChallengeFreeQuery{
		Popularity: popularity,
		Category:   categories,
		Name:       name,
	}

	challenges, err := handler.ChallengeStore.GetChallenges(freeQuery)
	if err != nil {
		handler.Logger.Println(err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Message{"message": constants.StatusInternalErrorMessage})
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Message{"data": challenges})

}
