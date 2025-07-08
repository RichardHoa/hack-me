package store

import (
	"database/sql"

	"github.com/RichardHoa/hack-me/internal/constants"
	"github.com/RichardHoa/hack-me/internal/utils"
)

type DBCommentStore struct {
	DB *sql.DB
}

func NewCommentStore(db *sql.DB) DBCommentStore {
	return DBCommentStore{DB: db}
}

type CommentStore interface {
	PostComment(req PostCommentRequest) error
}

type PostCommentRequest struct {
	ParentID            string `json:"ParentID"`
	ChallengeID         string `json:"ChallengeID"`
	ChallengeResponseID string `json:"ChallengeResponseID"`
	Content             string `json:"content"`
	UserID              string
}

func (store *DBCommentStore) PostComment(req PostCommentRequest) error {

	query := `
		INSERT INTO comment (parent_id, challenge_id, challenge_response_id, user_id, content)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`

	result, err := store.DB.Exec(
		query,
		utils.NullIfEmpty(req.ParentID),
		utils.NullIfEmpty(req.ChallengeID),
		utils.NullIfEmpty(req.ChallengeResponseID),
		req.UserID,
		req.Content,
	)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return utils.NewCustomAppError(constants.InternalError, "No err but the rows affected is 0")
	}

	return nil
}
