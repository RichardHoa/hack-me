package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/RichardHoa/hack-me/internal/constants"
	"github.com/RichardHoa/hack-me/internal/utils"
	"github.com/alexedwards/argon2id"
	"github.com/golang-jwt/jwt/v5"
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
	LoginUser(user *User) (accessToken string, refreshToken string, err error)
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

func (userStore *DBUserStore) LoginUser(user *User) (accessToken string, refreshToken string, err error) {

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
				fmt.Println("Invalid password")
				return "", "", errors.New(strconv.Itoa(constants.InvalidData))
			}
		}

	default:
		return "", "", errors.New("missing login credentials")
	}

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", "", errors.New(strconv.Itoa(constants.InvalidData))
		}
		return "", "", err
	}

	fmt.Printf("USER ID: %v\n", userID)
	user.ID = userID

	// 1. Create Access Token
	accessClaims := jwt.MapClaims{
		"exp": time.Now().Add(constants.AccessTokenTime).Unix(),
	}
	accessTokenObj := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessToken, err = accessTokenObj.SignedString([]byte(constants.AccessTokenSecret))
	if err != nil {
		return "", "", err
	}

	// 2. Create Refresh Token
	refreshClaims := jwt.MapClaims{
		"refresh_id": uuid.New().String(),
		"user_id":    userID,
		"exp":        time.Now().Add(constants.RefreshTokenTime).Unix(),
	}
	refreshTokenObj := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshToken, err = refreshTokenObj.SignedString([]byte(constants.RefreshTokenSecret))
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}
