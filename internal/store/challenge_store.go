package store

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type DBChallengeStore struct {
	DB *sql.DB
}

func NewChallengeStore(db *sql.DB) DBChallengeStore {
	return DBChallengeStore{DB: db}
}

type PostChallengeRequest struct {
	Name     string `json:"name"`
	Category string `json:"category"`
	Content  string `json:"content"`
	UserID   string `json:"userID"`
}

type Challenge struct {
	UserID    string    `json:"-"`
	UserName  string    `json:"userName"`
	Name      string    `json:"name"`
	Category  string    `json:"category"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
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
}

func (challengeStore *DBChallengeStore) GetChallenges(freeQuery ChallengeFreeQuery) (challenges Challenges, metaPage MetaDataPage, err error) {
	baseQuery := `
		SELECT 
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
	err = challengeStore.DB.QueryRow(countQuery, args...).Scan(&total)
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

	rows, err := challengeStore.DB.Query(baseQuery, args...)
	if err != nil {
		return nil, MetaDataPage{}, err
	}

	defer rows.Close()

	for rows.Next() {
		var c Challenge
		err := rows.Scan(&c.Name, &c.Category, &c.Content, &c.CreatedAt, &c.UpdatedAt, &c.UserName)
		if err != nil {
			return nil, MetaDataPage{}, err
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
			category,
			created_at, 
			updated_at
		) VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := challengeStore.DB.Exec(
		query,
		challenge.Name,
		challenge.Content,
		challenge.UserID,
		challenge.Category,
		time.Now(),
		time.Now(),
	)

	if err != nil {
		return err
	}

	return nil

}
