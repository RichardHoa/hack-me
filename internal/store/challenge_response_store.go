package store

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/RichardHoa/hack-me/internal/constants"
	"github.com/RichardHoa/hack-me/internal/utils"
)

type DBChallengeResponseStore struct {
	DB           *sql.DB
	CommentStore *DBCommentStore
}

func NewChallengeResponseStore(db *sql.DB, commentStore *DBCommentStore) *DBChallengeResponseStore {
	return &DBChallengeResponseStore{
		DB:           db,
		CommentStore: commentStore,
	}
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
	ChallengeID         string `json:"challengeID"`
	ChallengeResponseID string `json:"challengeResponseID"`
}

type ChallengeResponseOut []DetailChallengeResponse

type DetailChallengeResponse struct {
	ID            string    `json:"id"`
	ChallengeName string    `json:"challengeName"`
	AuthorName    string    `json:"authorName"`
	Name          string    `json:"name"`
	Content       string    `json:"content"`
	UpVote        string    `json:"upVote"`
	DownVote      string    `json:"downVote"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
	Comments      []Comment `json:"comments"`
}

type PostChallengeResponseRequest struct {
	ChallengeID string `json:"challengeID"`
	UserID      string
	Name        string `json:"name"`
	Content     string `json:"content"`
}

type ChallengeResponseStore interface {
	PostResponse(response PostChallengeResponseRequest) (challengeResponseID string, err error)
	ModifyResponse(response PutChallengeResponseRequest) error
	DeleteResponse(deleteRequest DeleteChallengeResponseRequest) error
	GetResponses(req GetChallengeResponseRequest) (*ChallengeResponseOut, error)
}

func (store *DBChallengeResponseStore) GetResponses(req GetChallengeResponseRequest) (*ChallengeResponseOut, error) {
	var (
		whereClause string
		arg         any
	)

	switch {
	case req.ChallengeResponseID != "":
		whereClause = "cr.id = $1"
		arg = req.ChallengeResponseID
	case req.ChallengeID != "":
		whereClause = "cr.challenge_id = $1"
		arg = req.ChallengeID
	default:
		panic("The handler is supposed to reject if there is no challengeID or challengeResponseID")
	}

	// nosemgrep
	query := fmt.Sprintf(`
    SELECT
        cr.id,
        cr.name,
        cr.content,
        cr.up_vote,
        cr.down_vote,
        cr.created_at,
        cr.updated_at,
        u.username,
        c.name AS challenge_name
    FROM
        challenge_response AS cr
    JOIN
        "user" AS u ON cr.user_id = u.id
    JOIN
        challenge AS c ON cr.challenge_id = c.id
    WHERE %s
    ORDER BY cr.created_at ASC
`, whereClause) // #nosec G201 - static where clause

	rows, err := store.DB.Query(query, arg)
	if err != nil {
		return nil, utils.NewCustomAppError(constants.InternalError, fmt.Sprintf("fail to query challenge_response: %v", err))
	}
	defer rows.Close()

	responses := ChallengeResponseOut{}

	for rows.Next() {
		var r DetailChallengeResponse
		if err := rows.Scan(&r.ID, &r.Name, &r.Content, &r.UpVote, &r.DownVote, &r.CreatedAt, &r.UpdatedAt, &r.AuthorName, &r.ChallengeName); err != nil {
			return nil, utils.NewCustomAppError(constants.InternalError, fmt.Sprintf("fail to scan challenge response: %v", err))
		}

		r.Comments, err = store.CommentStore.GetRootComments(ForeignChallengeResponseIDKey, r.ID)
		if err != nil {
			return nil, utils.NewCustomAppError(constants.InternalError, fmt.Sprintf("fail to get comments: %v", err))
		}

		responses = append(responses, r)
	}

	return &responses, nil
}

func (store *DBChallengeResponseStore) PostResponse(request PostChallengeResponseRequest) (challengeResponseID string, err error) {
	query := `
		INSERT INTO challenge_response (challenge_id, user_id, name, content)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`
	err = store.DB.QueryRow(query, request.ChallengeID, request.UserID, request.Name, request.Content).Scan(&challengeResponseID)
	if err != nil {
		return "", err
	}

	return challengeResponseID, err
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
