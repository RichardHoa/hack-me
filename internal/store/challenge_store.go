package store

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/RichardHoa/hack-me/internal/constants"
	"github.com/RichardHoa/hack-me/internal/utils"
)

type DBChallengeStore struct {
	DB           *sql.DB
	CommentStore *DBCommentStore
}

func NewChallengeStore(db *sql.DB, commentStore *DBCommentStore) DBChallengeStore {
	return DBChallengeStore{
		DB:           db,
		CommentStore: commentStore,
	}
}

type PostChallengeRequest struct {
	Name     string `json:"name"`
	Category string `json:"category"`
	Content  string `json:"content"`
	UserID   string
}

type DeleteChallengeRequest struct {
	Name string `json:"name"`
}

type PutChallengeRequest struct {
	Name     string `json:"name"`
	OldName  string `json:"oldName"`
	Category string `json:"category"`
	Content  string `json:"content"`
	UserID   string
}

type Challenge struct {
	ID        string    `json:"ID"`
	UserID    string    `json:"-"`
	UserName  string    `json:"userName"`
	Name      string    `json:"name"`
	Category  string    `json:"category"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	Comments  []Comment `json:"comments"`
}
type Challenges []Challenge

type ChallengeFreeQuery struct {
	Popularity string
	Category   []string
	Name       string
	ExactName  string
	PageSize   string
	Page       string
}

type MetaDataPage struct {
	PageSize    string `json:"pageSize"`
	CurrentPage string `json:"currentPage"`
	MaxPage     string `json:"maxPage"`
}

type ChallengeStore interface {
	GetChallenges(freeQuery ChallengeFreeQuery) (challenges Challenges, metaPage MetaDataPage, err error)
	CreateChallenges(challenge *Challenge) error
	DeleteChallenge(challengeName string, userID string) error
	ModifyChallenge(updatedChallenge PutChallengeRequest) error
}

func (Store *DBChallengeStore) GetChallenges(freeQuery ChallengeFreeQuery) (challenges Challenges, metaPage MetaDataPage, err error) {
	baseQuery := `
		SELECT 
			c.id,
			c.name, 
			c.category,
			c.content, 
			c.created_at, 
			c.updated_at, 
			u.username
		FROM challenge c
		JOIN "user" u ON c.user_id = u.id
	`

	countQuery := `SELECT COUNT(*) FROM challenge c`

	isExactQuery := false

	conditions := []string{}
	args := []any{}
	argIndex := 1

	if freeQuery.Name != "" && freeQuery.ExactName != "" {
		return Challenges{}, MetaDataPage{}, nil
	}
	// Filter by name (case-insensitive)
	if freeQuery.Name != "" {
		conditions = append(conditions, fmt.Sprintf("LOWER(name) LIKE LOWER($%d)", argIndex))
		args = append(args, "%"+freeQuery.Name+"%")
		argIndex++
	}

	// Filter by name (case sensitive, filter by exact)
	if freeQuery.ExactName != "" {
		isExactQuery = true
		conditions = append(conditions, fmt.Sprintf("name = $%d", argIndex))
		args = append(args, freeQuery.ExactName)
		argIndex++
	}

	// Filter by category
	if len(freeQuery.Category) > 0 {
		placeholders := []string{}
		for _, cat := range freeQuery.Category {
			placeholders = append(placeholders, fmt.Sprintf("$%d", argIndex))
			args = append(args, cat)
			argIndex++
		}
		conditions = append(conditions, fmt.Sprintf("category IN (%s)", strings.Join(placeholders, ", ")))
	}

	if len(conditions) > 0 {
		whereClause := " WHERE " + strings.Join(conditions, " AND ")
		baseQuery += whereClause
		countQuery += whereClause
	}

	switch strings.ToLower(freeQuery.Popularity) {
	case "asc":
		baseQuery += " ORDER BY popular_score ASC"
	default:
		baseQuery += " ORDER BY popular_score DESC"
	}

	pageSize := 10
	if ps, err := strconv.Atoi(freeQuery.PageSize); err == nil && ps > 0 {
		pageSize = ps
	}

	page := 1
	if p, err := strconv.Atoi(freeQuery.Page); err == nil && p > 0 {
		page = p
	}

	offset := (page - 1) * pageSize

	// Add pagination to the main query
	baseQuery += fmt.Sprintf(" LIMIT %d OFFSET %d", pageSize, offset)

	var total int
	err = Store.DB.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, MetaDataPage{}, err
	}
	maxPage := (total + pageSize - 1) / pageSize

	if page > maxPage {
		return Challenges{}, MetaDataPage{}, nil
	}

	metaPage = MetaDataPage{
		MaxPage:     strconv.Itoa(maxPage),
		PageSize:    strconv.Itoa(pageSize),
		CurrentPage: strconv.Itoa(page),
	}

	rows, err := Store.DB.Query(baseQuery, args...)
	if err != nil {
		return nil, MetaDataPage{}, err
	}

	defer rows.Close()

	for rows.Next() {
		var c Challenge
		err := rows.Scan(&c.ID, &c.Name, &c.Category, &c.Content, &c.CreatedAt, &c.UpdatedAt, &c.UserName)
		if err != nil {
			return nil, MetaDataPage{}, err
		}

		if isExactQuery == true {
			// Fetch comments for this challenge
			c.Comments, err = Store.CommentStore.GetRootComments("challenge_id", c.ID)
			if err != nil {
				return nil, MetaDataPage{}, err
			}
		}

		challenges = append(challenges, c)
	}

	if err := rows.Err(); err != nil {
		return nil, MetaDataPage{}, err
	}

	return challenges, metaPage, nil
}

func (challengeStore *DBChallengeStore) CreateChallenges(challenge *Challenge) error {
	query := `
		INSERT INTO challenge (
			name, 
			content, 
			user_id,
			category
		) VALUES ($1, $2, $3, $4)
	`

	_, err := challengeStore.DB.Exec(
		query,
		challenge.Name,
		challenge.Content,
		challenge.UserID,
		challenge.Category,
	)

	if err != nil {
		return err
	}

	return nil

}

func (challengeStore *DBChallengeStore) DeleteChallenge(challengeName string, userID string) error {
	var challengeExists bool

	err := challengeStore.DB.QueryRow(`
        SELECT EXISTS (SELECT 1 FROM challenge WHERE name = $1)
    `, challengeName).Scan(&challengeExists)
	if err != nil {
		return fmt.Errorf("failed to check challenge existence: %v", err)
	}

	if !challengeExists {
		return utils.NewCustomAppError(
			constants.InvalidData,
			"challengeName does not exist",
		)
	}

	query := `
        DELETE FROM challenge 
        WHERE name = $1 AND user_id = $2
    `

	result, err := challengeStore.DB.Exec(query, challengeName, userID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return utils.NewCustomAppError(constants.InternalError, fmt.Sprintf("fail to check rows affected %v", err.Error()))
	}

	if rowsAffected == 0 {
		return utils.NewCustomAppError(constants.LackingPermission, "User don't have permission to delete it")
	}

	return nil
}

func (challengeStore *DBChallengeStore) ModifyChallenge(updatedChallenge PutChallengeRequest) error {
	query := `UPDATE challenge SET `
	params := []interface{}{}
	paramCount := 1

	if updatedChallenge.Name != "" {
		query += fmt.Sprintf("name = $%d, ", paramCount)
		params = append(params, updatedChallenge.Name)
		paramCount++

		var count int
		err := challengeStore.DB.QueryRow(`
			SELECT COUNT(*) FROM challenge 
			 WHERE lower(name) = lower($1)
			`, updatedChallenge.Name).Scan(&count)

		if err != nil {
			return utils.NewCustomAppError(constants.InternalError, fmt.Sprintf("check name conflict failed: %v", err))
		}

		if count > 0 {
			return utils.NewCustomAppError(constants.InvalidData, "challenge name already exists")
		}

	}

	if updatedChallenge.Category != "" {
		query += fmt.Sprintf("category = $%d, ", paramCount)
		params = append(params, updatedChallenge.Category)
		paramCount++
	}

	if updatedChallenge.Content != "" {
		query += fmt.Sprintf("content = $%d, ", paramCount)
		params = append(params, updatedChallenge.Content)
		paramCount++
	}

	// If no valid fields to update
	if paramCount == 1 {
		return utils.NewCustomAppError(constants.InvalidData, "No valid field provided for challenge update")
	}

	// Remove trailing comma and space
	query = query[:len(query)-2]

	// Add WHERE clause with both name and user_id check
	query += fmt.Sprintf(", updated_at = now() WHERE name = $%d AND user_id = $%d", paramCount, paramCount+1)
	params = append(params, updatedChallenge.OldName, updatedChallenge.UserID)

	fmt.Printf("query: %v, params: %v", query, params)
	// Execute the update
	result, err := challengeStore.DB.Exec(query, params...)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return utils.NewCustomAppError(constants.InternalError, fmt.Sprintf("fail to check rows affected %v", err.Error()))
	}

	if rowsAffected == 0 {
		return utils.NewCustomAppError(constants.LackingPermission, "user does not have permission to modify the challenge")
	}

	return nil
}
