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
	ModifyComment(req ModifyCommentRequest) error
	DeleteComment(req DeleteCommentRequest) error
}

type PostCommentRequest struct {
	ParentID            string `json:"ParentID"`
	ChallengeID         string `json:"ChallengeID"`
	ChallengeResponseID string `json:"ChallengeResponseID"`
	Content             string `json:"content"`
	UserID              string
}

type ModifyCommentRequest struct {
	CommentID string `json:"commentID"`
	Content   string `json:"content"`
	UserID    string
}

type DeleteCommentRequest struct {
	CommentID string `json:"commentID"`
	UserID    string
}

func (store *DBCommentStore) PostComment(req PostCommentRequest) error {
	// TODO: Implement depth control when posting comment

	query := `
		INSERT INTO comment (parent_id, challenge_id, challenge_response_id, user_id, content)
		VALUES ($1, $2, $3, $4, $5)
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
		panic("No err but the rows affected is 0")
	}

	return nil
}

func (store *DBCommentStore) ModifyComment(req ModifyCommentRequest) error {
	// First check if the comment exists and belongs to the user
	checkQuery := `
        SELECT user_id 
        FROM comment 
        WHERE id = $1
    `

	var commentUserID string
	err := store.DB.QueryRow(checkQuery, req.CommentID).Scan(&commentUserID)
	if err != nil {
		if err == sql.ErrNoRows {
			return utils.NewCustomAppError(
				constants.InvalidData,
				"Comment not found or doesn't belong to this resource",
			)
		}
		return err
	}

	// Verify the comment belongs to the requesting user
	if commentUserID != req.UserID {
		return utils.NewCustomAppError(
			constants.LackingPermission,
			"You don't have permission to modify this comment",
		)
	}

	// Update the comment
	updateQuery := `
        UPDATE comment 
        SET content = $1, updated_at = now()
        WHERE id = $2
    `

	result, err := store.DB.Exec(
		updateQuery,
		req.Content,
		req.CommentID,
	)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		panic("No err but the rows affected is 0")
	}

	return nil
}

func (store *DBCommentStore) DeleteComment(req DeleteCommentRequest) error {
	// First check if the comment exists and belongs to the user
	checkQuery := `
        SELECT user_id 
        FROM comment 
        WHERE id = $1
    `

	var commentUserID string
	err := store.DB.QueryRow(checkQuery, req.CommentID).Scan(&commentUserID)
	if err != nil {
		if err == sql.ErrNoRows {
			return utils.NewCustomAppError(
				constants.InvalidData,
				"Comment not found or doesn't belong to this resource",
			)
		}
		return err
	}

	// Verify the comment belongs to the requesting user
	if commentUserID != req.UserID {
		return utils.NewCustomAppError(
			constants.LackingPermission,
			"You don't have permission to delete this comment",
		)
	}

	// Update the comment
	updateQuery := `
        DELETE FROM comment 
        WHERE id = $1 AND user_id = $2
    `

	result, err := store.DB.Exec(
		updateQuery,
		req.CommentID,
		req.UserID,
	)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		panic("No err but the rows affected is 0")
	}

	return nil
}
