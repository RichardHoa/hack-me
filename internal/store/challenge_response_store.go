package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/RichardHoa/hack-me/internal/constants"
	"github.com/RichardHoa/hack-me/internal/utils"
)

type DBChallengeResponseStore struct {
	DB *sql.DB
}

func NewChallengeResponseStore(db *sql.DB) DBChallengeResponseStore {
	return DBChallengeResponseStore{DB: db}
}

type DeleteChallengeResponseRequest struct {
	ChallengeResponseID string `json:"challengeResponseID"`
	UserID              string
}

type PutChallengeResponseRequest struct {
	ChallengeResponseID string `json:"challengeResponseID"`
	UserID              string
	Name                string `json:"name"`
	Content             string `json:"content"`
}

type GetChallengeResponseRequest struct {
	ChallengeID string `json:"challengeID"`
}

type ChallengeResponseOut []DetailChallengeResponse

type DetailChallengeResponse struct {
	ChallengeResponseID string    `json:"challengeResponseID"`
	AuthorName          string    `json:"authorName"`
	Name                string    `json:"name"`
	Content             string    `json:"content"`
	UpVote              string    `json:"upVote"`
	DownVote            string    `json:"downVote"`
	CreatedAt           time.Time `json:"createdAt"`
	UpdatedAt           time.Time `json:"updatedAt"`
}

type PostChallengeResponseRequest struct {
	ChallengeID string `json:"challengeID"`
	UserID      string
	Name        string `json:"name"`
	Content     string `json:"content"`
}

type ChallengeResponseStore interface {
	PostResponse(response PostChallengeResponseRequest) error
	ModifyResponse(response PutChallengeResponseRequest) error
	DeleteResponse(deleteRequest DeleteChallengeResponseRequest) error
	GetResponses(request GetChallengeResponseRequest) (ChallengeResponseOut, error)
}

func (store *DBChallengeResponseStore) GetResponses(request GetChallengeResponseRequest) (ChallengeResponseOut, error) {

	query := `
	SELECT cr.id, cr.name, cr.content, cr.up_vote, cr.down_vote, cr.created_at, cr.updated_at, u.username
	FROM challenge_response cr
	JOIN "user" u ON cr.user_id = u.id
	WHERE cr.challenge_id = $1
	ORDER BY cr.created_at ASC
	`

	rows, err := store.DB.Query(query, request.ChallengeID)
	if err != nil {
		return nil, utils.NewCustomAppError(constants.InternalError, fmt.Sprintf("fail to query challenge_response: %v", err.Error()))

	}
	defer rows.Close()

	responses := ChallengeResponseOut{}

	for rows.Next() {
		var r DetailChallengeResponse
		var id int
		var upVote, downVote int

		err := rows.Scan(&id, &r.Name, &r.Content, &upVote, &downVote, &r.CreatedAt, &r.UpdatedAt, &r.AuthorName)
		if err != nil {
			return nil, utils.NewCustomAppError(constants.InternalError, fmt.Sprintf("fail to scan challenge response into struct: %v", err.Error()))

		}

		r.ChallengeResponseID = fmt.Sprintf("%d", id)
		r.UpVote = fmt.Sprintf("%d", upVote)
		r.DownVote = fmt.Sprintf("%d", downVote)

		responses = append(responses, r)
	}

	// even if no rows, this will return an empty slice (not nil)
	return responses, nil
}

func (store *DBChallengeResponseStore) PostResponse(request PostChallengeResponseRequest) error {
	query := `
		INSERT INTO challenge_response (challenge_id, user_id, name, content)
		VALUES ($1, $2, $3, $4)
	`
	result, err := store.DB.Exec(query, request.ChallengeID, request.UserID, request.Name, request.Content)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("no row inserted")
	}

	return nil
}
func (store *DBChallengeResponseStore) DeleteResponse(deleteRequest DeleteChallengeResponseRequest) error {
	// Check if the challenge response exists
	var ownerID string
	checkQuery := `
	SELECT user_id FROM challenge_response 
	WHERE id = $1
`
	err := store.DB.QueryRow(checkQuery, deleteRequest.ChallengeResponseID).Scan(&ownerID)
	if err == sql.ErrNoRows {
		return utils.NewCustomAppError(constants.InvalidData, "challenge_id does not exist")
	}
	if err != nil {
		return err
	}

	// Check if the user owns the challenge response
	if ownerID != deleteRequest.UserID {
		return utils.NewCustomAppError(constants.LackingPermission, "User does not have permission to delete this challenge response")
	}

	deleteQuery := `
	DELETE FROM challenge_response
	WHERE id = $1 AND user_id = $2
	`

	result, err := store.DB.Exec(deleteQuery, deleteRequest.ChallengeResponseID, deleteRequest.UserID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return utils.NewCustomAppError(constants.InternalError, "Internal server error, valid request but challenge response does not get deleted")
	}

	return nil

}
func (store *DBChallengeResponseStore) ModifyResponse(request PutChallengeResponseRequest) error {
	// Check if the challenge response exists
	var ownerID string
	checkQuery := `
	SELECT user_id FROM challenge_response 
	WHERE id = $1
`
	err := store.DB.QueryRow(checkQuery, request.ChallengeResponseID).Scan(&ownerID)
	if err == sql.ErrNoRows {
		return utils.NewCustomAppError(constants.InvalidData, "challenge_id does not exist")
	}
	if err != nil {
		return err
	}
	fmt.Println(ownerID)
	fmt.Println(request.UserID)

	// Check if the user owns the challenge response
	if ownerID != request.UserID {
		return utils.NewCustomAppError(constants.LackingPermission, "User does not have permission to modify this challenge response")
	}

	// Build dynamic update query
	query := `UPDATE challenge_response SET `
	params := []interface{}{}
	paramCount := 1

	if request.Name != "" {
		query += fmt.Sprintf("name = $%d, ", paramCount)
		params = append(params, request.Name)
		paramCount++
	}

	if request.Content != "" {
		query += fmt.Sprintf("content = $%d, ", paramCount)
		params = append(params, request.Content)
		paramCount++
	}

	// Finalize query
	query = strings.TrimSuffix(query, ", ")
	query += ", updated_at = now()"
	query += fmt.Sprintf(" WHERE id = $%d AND user_id = $%d", paramCount, paramCount+1)

	params = append(params, request.ChallengeResponseID, request.UserID)

	result, err := store.DB.Exec(query, params...)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return utils.NewCustomAppError(constants.InternalError, "Internal server error, valid request but database does not get updated")
	}

	return nil
}
