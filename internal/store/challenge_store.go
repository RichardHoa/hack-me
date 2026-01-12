package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/RichardHoa/hack-me/internal/constants"
	"github.com/RichardHoa/hack-me/internal/domains"
	"github.com/RichardHoa/hack-me/internal/utils"
)

type DBChallengeStore struct {
	DB           *sql.DB
	CommentStore *DBCommentStore
}

func NewChallengeStore(db *sql.DB, commentStore *DBCommentStore) *DBChallengeStore {
	return &DBChallengeStore{
		DB:           db,
		CommentStore: commentStore,
	}
}

type PostChallengeParams struct {
	UserID   string
	Name     domains.ChallengeName
	Category string
	Content  string
}

type PostChallengeRequest struct {
	Name     string `json:"name"`
	Category string `json:"category"`
	Content  string `json:"content"`
}

type DeleteChallengeRequest struct {
	Name string `json:"name"`
}

type DeleteChallengeParams struct {
	ChallengeName domains.ChallengeName
	UserID        string
}

type ModifyChallengeRequest struct {
	NewName  string `json:"name"`
	OldName  string `json:"oldName"`
	Category string `json:"category"`
	Content  string `json:"content"`
}

type ModifyChallengeParams struct {
	OldName  domains.ChallengeName
	NewName  *domains.ChallengeName
	Category *string
	Content  *string
	UserID   string
}

type Challenge struct {
	ID        string                `json:"id"`
	UserName  string                `json:"userName"`
	Name      domains.ChallengeName `json:"name"`
	Category  string                `json:"category"`
	Content   string                `json:"content"`
	CreatedAt time.Time             `json:"createdAt"`
	UpdatedAt time.Time             `json:"updatedAt"`
	Comments  []Comment             `json:"comments"`
}
type Challenges []Challenge

type GetChallengeParams struct {
	Popularity *string
	Category   *[]string
	Name       *domains.ChallengeName
	ExactName  *domains.ChallengeName
	PageSize   *int
	Page       *int
}

type MetaDataPage struct {
	PageSize    string `json:"pageSize"`
	CurrentPage string `json:"currentPage"`
	MaxPage     string `json:"maxPage"`
}

type ChallengeStore interface {
	GetChallenges(params GetChallengeParams) (*Challenges, *MetaDataPage, error)
	CreateChallenges(params *PostChallengeParams) error
	DeleteChallenge(params DeleteChallengeParams) error
	ModifyChallenge(params ModifyChallengeParams) error
}

func (Store *DBChallengeStore) GetChallenges(params GetChallengeParams) (*Challenges, *MetaDataPage, error) {
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
	conditions := make([]string, 0, 3)
	args := []any{}
	argIndex := 1

	// Filter by name (case-insensitive)
	if params.Name != nil {
		conditions = append(conditions, fmt.Sprintf("LOWER(c.name) LIKE LOWER($%d)", argIndex))
		args = append(args, "%"+params.Name.String()+"%")
		argIndex++
	}

	// Filter by name (case sensitive, filter by exact)
	if params.ExactName != nil {
		isExactQuery = true
		conditions = append(conditions, fmt.Sprintf("c.name = $%d", argIndex))
		args = append(args, params.ExactName)
		argIndex++
	}

	// Filter by category
	if params.Category != nil && len(*params.Category) > 0 {
		placeholders := make([]string, len(*params.Category))
		for i, cat := range *params.Category {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, cat)
			argIndex++
		}
		conditions = append(conditions, fmt.Sprintf("c.category IN (%s)", strings.Join(placeholders, ", ")))
	}

	if len(conditions) > 0 {
		whereClause := " WHERE " + strings.Join(conditions, " AND ")
		baseQuery += whereClause
		countQuery += whereClause
	}

	if params.Popularity != nil {
		switch strings.ToLower(*params.Popularity) {
		case "asc":
			baseQuery += " ORDER BY c.popular_score ASC"
		case "desc":
			baseQuery += " ORDER BY c.popular_score DESC"
		default:
			return &Challenges{}, &MetaDataPage{}, errors.New("Invalid popularity parameters")
		}
	}

	pageSize := constants.DefaultPageSize
	if params.PageSize != nil {
		pageSize = *params.PageSize
	}

	page := constants.DefaultPage
	if params.Page != nil {
		page = *params.Page
	}

	offset := (page - 1) * pageSize
	baseQuery += fmt.Sprintf(" LIMIT %d OFFSET %d", pageSize, offset)

	var total int
	err := Store.DB.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return &Challenges{}, &MetaDataPage{}, err
	}

	// Early return if no results
	if total == 0 {
		return &Challenges{}, &MetaDataPage{
			MaxPage:     "0",
			PageSize:    strconv.Itoa(pageSize),
			CurrentPage: strconv.Itoa(page),
		}, nil
	}

	maxPage := (total + pageSize - 1) / pageSize
	if page > maxPage {
		return &Challenges{}, &MetaDataPage{}, nil
	}

	metaPage := MetaDataPage{
		MaxPage:     strconv.Itoa(maxPage),
		PageSize:    strconv.Itoa(pageSize),
		CurrentPage: strconv.Itoa(page),
	}

	challenges := make(Challenges, 0, pageSize) // Pre-allocate with capacity
	rows, err := Store.DB.Query(baseQuery, args...)
	if err != nil {
		return &Challenges{}, &MetaDataPage{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var c Challenge
		err := rows.Scan(&c.ID, &c.Name, &c.Category, &c.Content, &c.CreatedAt, &c.UpdatedAt, &c.UserName)
		if err != nil {
			return nil, nil, err
		}
		challenges = append(challenges, c)
	}

	if err := rows.Err(); err != nil {
		return &Challenges{}, &MetaDataPage{}, err
	}

	// Fetch comments only for exact queries (single challenge)
	if isExactQuery && len(challenges) > 0 {
		for i := range challenges {
			challenges[i].Comments, err = Store.CommentStore.GetRootComments(ForeignChallengeIDKey, challenges[i].ID)
			if err != nil {
				return &Challenges{}, &MetaDataPage{}, err
			}
		}
	}

	return &challenges, &metaPage, nil
}

func (challengeStore *DBChallengeStore) CreateChallenges(params *PostChallengeParams) error {
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
		params.Name,
		params.Content,
		params.UserID,
		params.Category,
	)

	if err != nil {
		return err
	}

	return nil

}

func (challengeStore *DBChallengeStore) DeleteChallenge(params DeleteChallengeParams) error {
	var challengeExists bool

	err := challengeStore.DB.QueryRow(`
        SELECT EXISTS (SELECT 1 FROM challenge WHERE name = $1)
    `, params.ChallengeName).Scan(&challengeExists)
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

	result, err := challengeStore.DB.Exec(query, params.ChallengeName, params.UserID)
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

func (challengeStore *DBChallengeStore) ModifyChallenge(params ModifyChallengeParams) error {
	query := `UPDATE challenge SET `
	queryParams := []any{}
	paramCount := 1

	if params.NewName != nil {
		query += fmt.Sprintf("name = $%d, ", paramCount)
		queryParams = append(queryParams, params.NewName)
		paramCount++

		var count int
		err := challengeStore.DB.QueryRow(`
			SELECT COUNT(*) FROM challenge 
			 WHERE lower(name) = lower($1)
			`, params.NewName).Scan(&count)

		if err != nil {
			return utils.NewCustomAppError(constants.InternalError, fmt.Sprintf("check name conflict failed: %v", err))
		}

		if count > 0 {
			return utils.NewCustomAppError(constants.InvalidData, "challenge name already exists")
		}

	}

	if params.Category != nil {
		query += fmt.Sprintf("category = $%d, ", paramCount)
		queryParams = append(queryParams, params.Category)
		paramCount++
	}

	if params.Content != nil {
		query += fmt.Sprintf("content = $%d, ", paramCount)
		queryParams = append(queryParams, params.Content)
		paramCount++
	}

	if paramCount == 1 {
		return utils.NewCustomAppError(constants.InvalidData, "No valid field provided for challenge update")
	}

	query = query[:len(query)-2]

	query += fmt.Sprintf(", updated_at = now() WHERE name = $%d AND user_id = $%d", paramCount, paramCount+1)
	queryParams = append(queryParams, params.OldName, params.UserID)

	// Execute the update
	result, err := challengeStore.DB.Exec(query, queryParams...)
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
