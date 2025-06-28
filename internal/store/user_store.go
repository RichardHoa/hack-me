package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/RichardHoa/hack-me/internal/constants"
	"github.com/RichardHoa/hack-me/internal/utils"
	"github.com/alexedwards/argon2id"
	"github.com/google/uuid"
)

type DBUserStore struct {
	DB *sql.DB
}

func NewUserStore(db *sql.DB) DBUserStore {
	return DBUserStore{
		DB: db,
	}
}

type UserStore interface {
	CreateUser(user *User) (uuid.UUID, error)
	LoginAndIssueTokens(user *User) (accessToken string, refreshToken string, err error)
}

type Password struct {
	PlainText string
	HashText  string
}

type RegisterUserRequest struct {
	Username  string   `json:"user_name"`
	Password  Password `json:"password"`
	Email     string   `json:"email"`
	ImageLink string   `json:"image_link"`
	GoogleID  string   `json:"google_id"`
	GithubID  string   `json:"github_id"`
}

type User struct {
	ID        string   `json:"-"`
	Username  string   `json:"user_name"`
	Password  Password `json:"password"`
	Email     string   `json:"email"`
	ImageLink string   `json:"image_link"`
	GoogleID  string   `json:"google_id"`
	GithubID  string   `json:"github_id"`
}

func (p *Password) UnmarshalJSON(data []byte) error {
	var plain string
	if err := json.Unmarshal(data, &plain); err != nil {
		return err
	}
	p.PlainText = plain
	return nil
}

var (
	InvalidPasswordErr = errors.New("Invalid password")
)

func (userStore *DBUserStore) CreateUser(user *User) (uuid.UUID, error) {

	var hashedPassword string
	userUUID := uuid.New()
	var err error

	// Only hash the password if provided
	if user.Password.PlainText != "" {
		hashedPassword, err = argon2id.CreateHash(user.Password.PlainText, argon2id.DefaultParams)
		if err != nil {
			return uuid.UUID{}, err
		}
	}

	query := `
		INSERT INTO "user" (
			id,
			username,
			email,
			image_link,
			password,
			google_id,
			github_id,
			created_at,
			updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	// fmt.Println(utils.BeautifyJSON(user))

	_, err = userStore.DB.Exec(
		query,
		userUUID,
		utils.NullIfEmpty(user.Username),
		utils.NullIfEmpty(user.Email),
		user.ImageLink,
		utils.NullIfEmpty(hashedPassword),
		utils.NullIfEmpty(user.GoogleID),
		utils.NullIfEmpty(user.GithubID),
		time.Now(),
		time.Now(),
	)

	if err != nil {
		return uuid.UUID{}, err
	}

	return userUUID, nil

}

func (userStore *DBUserStore) LoginAndIssueTokens(user *User) (accessToken string, refreshToken string, err error) {

	var userID string

	switch {
	case user.GoogleID != "":
		err = userStore.DB.QueryRow(`SELECT id FROM "user" WHERE google_id = $1`, user.GoogleID).Scan(&userID)

	case user.GithubID != "":
		err = userStore.DB.QueryRow(`SELECT id FROM "user" WHERE github_id = $1`, user.GithubID).Scan(&userID)

	case user.Email != "" && user.Password.PlainText != "":
		var hashed string
		err = userStore.DB.QueryRow(`SELECT id, password FROM "user" WHERE email = $1`, user.Email).Scan(&userID, &hashed)
		if err == nil {
			match, cmpErr := argon2id.ComparePasswordAndHash(user.Password.PlainText, hashed)
			if cmpErr != nil {
				return "", "", cmpErr
			}
			if !match {
				// password not correct
				return "", "", utils.NewCustomAppError(constants.InvalidData, "Invalid password")
			}
		}

	default:
		return "", "", utils.NewCustomAppError(constants.InvalidData, "Missing loggin credentials")
	}

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", "", utils.NewCustomAppError(constants.InvalidData, "Cannot find user in the database")
		}
		return "", "", err
	}

	fmt.Printf("USER ID: %v\n", userID)
	user.ID = userID

	accessToken, refreshToken, err = utils.CreateTokens(userID)

	return accessToken, refreshToken, nil
}
