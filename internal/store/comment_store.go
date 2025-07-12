package store

import (
	"database/sql"
	"fmt"
	"time"

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
	PostComment(req PostCommentRequest) (commentID string, err error)
	ModifyComment(req ModifyCommentRequest) error
	DeleteComment(req DeleteCommentRequest) error
	GetRootComments(foreignKey string, id string) ([]Comment, error)
}

type Comment struct {
	ID        string    `json:"id"`
	Content   string    `json:"content"`
	Author    string    `json:"author"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	Comments  []Comment `json:"comments,omitempty"`
}

type PostCommentRequest struct {
	ParentID            string `json:"parentID"`
	ChallengeID         string `json:"challengeID"`
	ChallengeResponseID string `json:"challengeResponseID"`
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

func (store *DBCommentStore) PostComment(req PostCommentRequest) (commentID string, err error) {
	// TODO: Implement depth control when posting comment

	query := `
		INSERT INTO comment (parent_id, challenge_id, challenge_response_id, user_id, content)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`

	err = store.DB.QueryRow(
		query,
		utils.NullIfEmpty(req.ParentID),
		utils.NullIfEmpty(req.ChallengeID),
		utils.NullIfEmpty(req.ChallengeResponseID),
		req.UserID,
		req.Content,
	).Scan(&commentID)

	if err != nil {
		return "", err
	}

	return commentID, nil
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

func (store *DBCommentStore) GetRootComments(foreignKey string, id string) ([]Comment, error) {
	if foreignKey != "challenge_id" && foreignKey != "challenge_response_id" {
		return nil, fmt.Errorf("unsupported foreign key: %s", foreignKey)
	}

	query := fmt.Sprintf(`
		SELECT 
			c.id, c.content, u.username, c.created_at, c.updated_at
		FROM comment c
		JOIN "user" u ON c.user_id = u.id
		WHERE c.%s = $1 AND c.parent_id IS NULL
		ORDER BY c.created_at ASC
	`, foreignKey)

	rows, err := store.DB.Query(query, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []Comment
	for rows.Next() {
		var comment Comment
		err := rows.Scan(&comment.ID, &comment.Content, &comment.Author, &comment.CreatedAt, &comment.UpdatedAt)
		if err != nil {
			return nil, err
		}

		replies, err := store.getRepliesRecursive(comment.ID, 1)
		if err != nil {
			return nil, err
		}
		comment.Comments = replies

		comments = append(comments, comment)
	}

	return comments, nil
}

func (store *DBCommentStore) getRepliesRecursive(parentID string, depth int) ([]Comment, error) {
	if depth >= constants.CommentNestedLevel {
		return nil, nil
	}

	query := `
		SELECT 
			c.id, c.content, u.username, c.created_at, c.updated_at
		FROM comment c
		JOIN "user" u ON c.user_id = u.id
		WHERE c.parent_id = $1
		ORDER BY c.created_at ASC
	`

	rows, err := store.DB.Query(query, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var replies []Comment
	for rows.Next() {
		var reply Comment
		err := rows.Scan(&reply.ID, &reply.Content, &reply.Author, &reply.CreatedAt, &reply.UpdatedAt)
		if err != nil {
			return nil, err
		}

		children, err := store.getRepliesRecursive(reply.ID, depth+1)
		if err != nil {
			return nil, err
		}
		reply.Comments = children

		replies = append(replies, reply)
	}

	return replies, nil
}
