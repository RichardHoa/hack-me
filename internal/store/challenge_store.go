package store

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgconn"
)

type DBChallengeStore struct {
	DB *sql.DB
}

func NewChallengeStore(db *sql.DB) DBChallengeStore {
	return DBChallengeStore{DB: db}
}

type Challenge struct {
	Popular_score int       `json:"-"`
	Name          string    `json:"name"`
	Content       string    `json:"content"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
type Challenges []Challenge

type ChallengeFreeQuery struct {
	Popularity string
	Category   []string
	Name       string
}

type ChallengeStore interface {
	GetChallenges(query ChallengeFreeQuery) (*Challenges, error)
}

func (challengeStore *DBChallengeStore) GetChallenges(freeQuery ChallengeFreeQuery) (*Challenges, error) {
	baseQuery := `
		SELECT name, content, created_at, updated_at, popular_score
		FROM challenge
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
	case "desc":
		baseQuery += " ORDER BY popular_score DESC"
	}

	rows, err := challengeStore.DB.Query(baseQuery, args...)

	if err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "22P02" {
			// 22P02 = invalid_text_representation (often caused by invalid enum input)
			return &Challenges{}, nil
		}
		return nil, err
	}
	defer rows.Close()

	var challenges Challenges

	for rows.Next() {
		var c Challenge
		err := rows.Scan(&c.Name, &c.Content, &c.CreatedAt, &c.UpdatedAt, &c.Popular_score)
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
