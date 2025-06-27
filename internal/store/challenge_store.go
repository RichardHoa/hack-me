package store

import (
	"database/sql"
	"fmt"
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
	UserID   string `json:"user_id"`
}

type Challenge struct {
	UserID    string    `json:"-"`
	Username  string    `json:"user_name"`
	Name      string    `json:"name"`
	Category  string    `json:"category"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
type Challenges []Challenge

type ChallengeFreeQuery struct {
	Popularity string
	Category   []string
	Name       string
}

type ChallengeStore interface {
	GetChallenges(query ChallengeFreeQuery) (*Challenges, error)
	CreateChallenges(challenge *Challenge) error
}

func (challengeStore *DBChallengeStore) GetChallenges(freeQuery ChallengeFreeQuery) (*Challenges, error) {
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
	conditions := []string{}
	args := []interface{}{}
	argIndex := 1

	// Filter by name (case-insensitive)
	if freeQuery.Name != "" {
		conditions = append(conditions, fmt.Sprintf("LOWER(name) LIKE LOWER($%d)", argIndex))
		args = append(args, "%"+freeQuery.Name+"%")
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
		baseQuery += " WHERE " + strings.Join(conditions, " AND ")
	}

	switch strings.ToLower(freeQuery.Popularity) {
	case "asc":
		baseQuery += " ORDER BY popular_score ASC"
	default:
		baseQuery += " ORDER BY popular_score DESC"
	}

	rows, err := challengeStore.DB.Query(baseQuery, args...)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var challenges Challenges

	for rows.Next() {
		var c Challenge
		err := rows.Scan(&c.Name, &c.Category, &c.Content, &c.CreatedAt, &c.UpdatedAt, &c.Username)
		if err != nil {
			return nil, err
		}
		challenges = append(challenges, c)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &challenges, nil
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
