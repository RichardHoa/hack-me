package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/RichardHoa/hack-me/internal/constants"
	"github.com/RichardHoa/hack-me/internal/utils"
	"github.com/alexedwards/argon2id"
	"github.com/google/uuid"
)

type DBUserStore struct {
	DB *sql.DB
}

func NewUserStore(db *sql.DB) *DBUserStore {
	return &DBUserStore{
		DB: db,
	}
}

type UserStore interface {
	CreateUser(user *User) (uuid.UUID, error)
	LoginAndIssueTokens(user *User) (accessToken, refreshToken, csrfToken string, err error)
}

type Password struct {
	PlainText string
	HashText  string
}

type RegisterUserRequest struct {
	Username  string   `json:"userName"`
	Password  Password `json:"password"`
	Email     string   `json:"email"`
	ImageLink string   `json:"imageLink"`
	GoogleID  string   `json:"googleID"`
	GithubID  string   `json:"githubID"`
}

type User struct {
	ID        string   `json:"-"`
	Username  string   `json:"userName"`
	Password  Password `json:"password"`
	Email     string   `json:"email"`
	ImageLink string   `json:"imageLink"`
	GoogleID  string   `json:"googleID"`
	GithubID  string   `json:"githubID"`
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
			github_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
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
	)

	if err != nil {
		return uuid.UUID{}, err
	}

	return userUUID, nil

}

func (userStore *DBUserStore) LoginAndIssueTokens(user *User) (accessToken, refreshToken, csrfToken string, err error) {

	var (
		userID   string
		userName string
	)

	switch {
	case user.GoogleID != "":
		err = userStore.DB.QueryRow(`SELECT id, username FROM "user" WHERE google_id = $1`, user.GoogleID).Scan(&userID, &userName)

	case user.GithubID != "":
		err = userStore.DB.QueryRow(`SELECT id, username FROM "user" WHERE github_id = $1`, user.GithubID).Scan(&userID, &userName)

	case user.Email != "" && user.Password.PlainText != "":
		var hashed string
		err = userStore.DB.QueryRow(`SELECT id, username, password FROM "user" WHERE email = $1`, user.Email).Scan(&userID, &userName, &hashed)
		if err == nil {
			match, cmpErr := argon2id.ComparePasswordAndHash(user.Password.PlainText, hashed)
			if cmpErr != nil {
				return "", "", "", cmpErr
			}
			if !match {
				// password not correct
				return "", "", "", utils.NewCustomAppError(constants.InvalidData, "Invalid password")
			}
		}

	default:
		panic("user_store > Missing login credentials while login")
	}

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", "", "", utils.NewCustomAppError(constants.InvalidData, "Cannot find user in the database")
		}
		return "", "", "", err
	}

	user.ID = userID

	accessToken, refreshToken, err = utils.CreateTokens(userID, userName, 0)

	result, err := utils.ExtractClaimsFromJWT(refreshToken, []string{"refreshID"})
	if err != nil {
		return "", "", "", utils.NewCustomAppError(constants.InternalError, fmt.Sprintf("fail to decode decode refreshToken %v", err))
	}

	refreshtokenID := result[0]

	csrfToken, err = utils.CreateCSRFToken(refreshtokenID)
	if err != nil {
		return "", "", "", utils.NewCustomAppError(constants.InternalError, fmt.Sprintf("fail to create csrfToken %v", err))
	}

	return accessToken, refreshToken, csrfToken, nil
}
