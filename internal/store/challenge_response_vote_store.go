package store

import (
	"database/sql"
	"fmt"

	"github.com/RichardHoa/hack-me/internal/constants"
	"github.com/RichardHoa/hack-me/internal/utils"
)

type DBVoteStore struct {
	DB *sql.DB
}

func NewVoteStore(db *sql.DB) *DBVoteStore {
	return &DBVoteStore{DB: db}
}

type DeleteVoteRequest struct {
	ChallengeResponseID string `json:"challengeResponseID"`
	UserID              string
}

type PostVoteRequest struct {
	ChallengeResponseID string `json:"challengeResponseID"`
	VoteType            string `json:"voteType"`
	UserID              string
}

type VoteStore interface {
	PostVote(req PostVoteRequest) error
	DeleteVote(req DeleteVoteRequest) error
}

func (store *DBVoteStore) DeleteVote(req DeleteVoteRequest) error {
	var existingVoteType int
	query := `
        SELECT vote_type FROM challenge_response_votes 
        WHERE user_id = $1 AND challenge_response_id = $2
    `
	err := store.DB.QueryRow(query, req.UserID, req.ChallengeResponseID).Scan(&existingVoteType)
	if err == sql.ErrNoRows {
		return utils.NewCustomAppError(constants.InvalidData, "User does not have any vote for this response challenge")
	}
	if err != nil {
		return err
	}

	// The Database Trigger (trg_sync_votes) will automatically decrement
	// the up_vote or down_vote count in challenge_response after this execution.
	_, err = store.DB.Exec(`
        DELETE FROM challenge_response_votes 
        WHERE user_id = $1 AND challenge_response_id = $2
    `, req.UserID, req.ChallengeResponseID)
	if err != nil {
		return err
	}

	return nil
}

func (store *DBVoteStore) PostVote(req PostVoteRequest) error {
	var existingVote int
	query := `
        SELECT vote_type FROM challenge_response_votes
        WHERE user_id = $1 AND challenge_response_id = $2
    `
	err := store.DB.QueryRow(query, req.UserID, req.ChallengeResponseID).Scan(&existingVote)

	var newVoteType int
	switch req.VoteType {
	case "upVote":
		newVoteType = 1
	case "downVote":
		newVoteType = -1
	default:
		panic("unexpected voteType get into the database level")
	}

	if err == nil && existingVote == newVoteType {
		return utils.NewCustomAppError(constants.InvalidData, fmt.Sprintf("You already make a %v", req.VoteType))
	}
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	_, err = store.DB.Exec(`
        INSERT INTO challenge_response_votes (user_id, challenge_response_id, vote_type)
        VALUES ($1, $2, $3)
        ON CONFLICT (user_id, challenge_response_id)
        DO UPDATE SET vote_type = EXCLUDED.vote_type, updated_at = now()
    `, req.UserID, req.ChallengeResponseID, newVoteType)

	return err
}
