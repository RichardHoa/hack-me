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

func NewVoteStore(db *sql.DB) DBVoteStore {
	return DBVoteStore{DB: db}
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
	tx, err := store.DB.Begin()
	if err != nil {
		return err
	}
	defer func() {
		err = tx.Rollback()
		if err != nil && err != sql.ErrTxDone {
			fmt.Printf("Err while roll back %v", err.Error())
		}
	}()

	// Step 1: Check if the vote exists and get its type
	var existingVoteType int
	query := `
        SELECT vote_type FROM challenge_response_votes
        WHERE user_id = $1 AND challenge_response_id = $2
    `
	err = tx.QueryRow(query, req.UserID, req.ChallengeResponseID).Scan(&existingVoteType)
	if err == sql.ErrNoRows {
		// Vote doesn't exist, nothing to do
		return utils.NewCustomAppError(constants.InvalidData, "User does not have any vote for this response challenge")
	}
	if err != nil {
		return err
	}

	// Step 2: Delete the vote
	_, err = tx.Exec(`
        DELETE FROM challenge_response_votes
        WHERE user_id = $1 AND challenge_response_id = $2
    `, req.UserID, req.ChallengeResponseID)
	if err != nil {
		return err
	}

	// Step 3: Update vote counts based on the deleted vote's type
	if existingVoteType == 1 {
		// Was an upvote, decrement up_vote count
		_, err = tx.Exec(`
            UPDATE challenge_response
            SET up_vote = up_vote - 1
            WHERE id = $1
        `, req.ChallengeResponseID)
	} else if existingVoteType == -1 {
		// Was a downvote, decrement down_vote count
		_, err = tx.Exec(`
            UPDATE challenge_response
            SET down_vote = down_vote - 1
            WHERE id = $1
        `, req.ChallengeResponseID)
	}

	if err != nil {
		return err
	}

	return tx.Commit()
}
func (store *DBVoteStore) PostVote(req PostVoteRequest) error {
	tx, err := store.DB.Begin()
	if err != nil {
		return err
	}
	defer func() {
		err = tx.Rollback()
		if err != nil {
			fmt.Printf("Err while roll back %v", err.Error())
		}
	}()

	// Step 1: Check if user already voted on this response
	var existingVote int
	query := `
		SELECT vote_type FROM challenge_response_votes
		WHERE user_id = $1 AND challenge_response_id = $2
	`
	err = tx.QueryRow(query, req.UserID, req.ChallengeResponseID).Scan(&existingVote)
	hasExistingVote := err != sql.ErrNoRows
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	// Step 2: Map input string to numeric vote type
	var newVoteType int
	switch req.VoteType {
	case "upVote":
		newVoteType = 1
	case "downVote":
		newVoteType = -1
	default:
		panic("unexpected voteType get into the database level")
	}

	// Step 3: If the vote already exists and is unchanged, do nothing
	if hasExistingVote && existingVote == newVoteType {
		return nil
	}

	// Step 4: Upsert the new vote value
	_, err = tx.Exec(`
		INSERT INTO challenge_response_votes (user_id, challenge_response_id, vote_type)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, challenge_response_id)
		DO UPDATE SET vote_type = EXCLUDED.vote_type, updated_at = now()
	`, req.UserID, req.ChallengeResponseID, newVoteType)
	if err != nil {
		return err
	}

	// Step 5: Update vote counts depending on the case
	switch {
	case !hasExistingVote && newVoteType == 1:
		// Case A: First time upvoting
		_, err = tx.Exec(`
			UPDATE challenge_response
			SET up_vote = up_vote + 1
			WHERE id = $1
		`, req.ChallengeResponseID)

	case !hasExistingVote && newVoteType == -1:
		// Case B: First time downvoting
		_, err = tx.Exec(`
			UPDATE challenge_response
			SET down_vote = down_vote + 1
			WHERE id = $1
		`, req.ChallengeResponseID)

	case hasExistingVote && existingVote == 1 && newVoteType == -1:
		// Case C: Changing vote from upvote to downvote
		_, err = tx.Exec(`
			UPDATE challenge_response
			SET up_vote = up_vote - 1, down_vote = down_vote + 1
			WHERE id = $1
		`, req.ChallengeResponseID)

	case hasExistingVote && existingVote == -1 && newVoteType == 1:
		// Case D: Changing vote from downvote to upvote
		_, err = tx.Exec(`
			UPDATE challenge_response
			SET down_vote = down_vote - 1, up_vote = up_vote + 1
			WHERE id = $1
		`, req.ChallengeResponseID)
	}

	if err != nil {
		return err
	}

	return tx.Commit()
}
