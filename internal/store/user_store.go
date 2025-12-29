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

func NewUserStore(db *sql.DB) *DBUserStore {
	return &DBUserStore{
		DB: db,
	}
}

type UserStore interface {
	CreateUser(user *User) (uuid.UUID, error)
	LoginAndIssueTokens(user *User) (accessToken, refreshToken, csrfToken string, err error)
	GetUserActivity(userID string) (*UserActivityData, error)
	ChangePassword(req ChangePasswordRequest) error
	ChangeUsername(req ChangeUsernameRequest) error
	DeleteUser(userID string) error
	GetUserName(userID string) (userName string, err error)
}

type Password struct {
	PlainText string
	HashText  string
}

func (p *Password) UnmarshalJSON(data []byte) error {
	var plain string
	if err := json.Unmarshal(data, &plain); err != nil {
		return err
	}
	p.PlainText = plain
	return nil
}

type RegisterUserRequest struct {
	Username  string   `json:"userName"`
	Password  Password `json:"password"`
	Email     string   `json:"email"`
	ImageLink string   `json:"imageLink"`
	GoogleID  string   `json:"googleID"`
	GithubID  string   `json:"githubID"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"oldPassword"`
	NewPassword string `json:"newPassword"`
	UserID      string `json:"-"`
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

type UserProfile struct {
	Username  string `json:"userName"`
	ImageLink string `json:"imageLink"`
}

type UserChallengeSummary struct {
	Name          string    `json:"name"`
	Category      string    `json:"category"`
	CommentCount  string    `json:"commentCount"`
	ResponseCount string    `json:"responseCount"`
	PopularScore  string    `json:"popularScore"`
	UpdatedAt     time.Time `json:"updatedAt"`
	CreatedAt     time.Time `json:"createdAt"`
}

type UserResponseSummary struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	UpVote    string    `json:"upVote"`
	DownVote  string    `json:"downVote"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type UserActivityData struct {
	User               UserProfile            `json:"user"`
	Challenges         []UserChallengeSummary `json:"challenges"`
	ChallengeResponses []UserResponseSummary  `json:"challengeResponses"`
}

type ChangeUsernameRequest struct {
	NewUsername string `json:"newUsername"`
	UserID      string `json:"-"`
}

func (userStore *DBUserStore) ChangeUsername(req ChangeUsernameRequest) error {
	query := `UPDATE "user" SET username = $1, updated_at = now() WHERE id = $2`
	result, err := userStore.DB.Exec(query, req.NewUsername, req.UserID)
	if err != nil {
		// This will catch unique constraint violations for the username
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return utils.NewCustomAppError(constants.ResourceNotFound, "user not found")
	}

	return nil
}

func (userStore *DBUserStore) DeleteUser(userID string) error {
	query := `DELETE FROM "user" WHERE id = $1`
	result, err := userStore.DB.Exec(query, userID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return utils.NewCustomAppError(constants.ResourceNotFound, "user not found")
	}

	return nil
}

func (userStore *DBUserStore) ChangePassword(req ChangePasswordRequest) error {
	// 1. Fetch current password
	var currentHashedPassword sql.NullString
	query := `SELECT password FROM "user" WHERE id = $1`
	err := userStore.DB.QueryRow(query, req.UserID).Scan(&currentHashedPassword)
	if err != nil {
		if err == sql.ErrNoRows {
			return utils.NewCustomAppError(constants.ResourceNotFound, fmt.Sprintf("userID: %v does not exist in database", req.UserID))
		}
		return err
	}

	// google and github login people do not have password
	// if they want they can create their password
	if req.OldPassword == "" {
		if currentHashedPassword.Valid && currentHashedPassword.String != "" {
			return utils.NewCustomAppError(constants.InvalidData, "old password is required to change existing password")
		}
	} else {
		if !currentHashedPassword.Valid || currentHashedPassword.String == "" {
			return utils.NewCustomAppError(constants.InvalidData, "user does not have a password set up, but old password was provided")
		}

		// Compare old password
		match, err := argon2id.ComparePasswordAndHash(req.OldPassword, currentHashedPassword.String)
		if err != nil {
			return err
		}
		if !match {
			return utils.NewCustomAppError(constants.InvalidData, "incorrect old password")
		}
		// Passwords match, proceed to update.
	}

	newHashedPassword, err := argon2id.CreateHash(req.NewPassword, argon2id.DefaultParams)
	if err != nil {
		return err
	}

	updateQuery := `UPDATE "user" SET password = $1, updated_at = now() WHERE id = $2`
	result, err := userStore.DB.Exec(updateQuery, newHashedPassword, req.UserID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return utils.NewCustomAppError(constants.InternalError, "Change password > Valid data, no err, but no updated rows")
	}

	return nil
}

func (userStore *DBUserStore) GetUserActivity(userID string) (*UserActivityData, error) {
	activityData := UserActivityData{
		User:               UserProfile{},
		Challenges:         []UserChallengeSummary{},
		ChallengeResponses: []UserResponseSummary{},
	}

	// 1. Get user info
	userQuery := `SELECT username, image_link FROM "user" WHERE id = $1`
	err := userStore.DB.QueryRow(userQuery, userID).Scan(&activityData.User.Username, &activityData.User.ImageLink)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, utils.NewCustomAppError(constants.ResourceNotFound, "user not found")
		}
		return nil, err
	}

	// 2. Get user's challenges with counts
	challengesQuery := `
		SELECT 
			c.name, c.updated_at, c.created_at, c.category, c.popular_score,
			(SELECT COUNT(*) FROM comment WHERE challenge_id = c.id) as comment_count,
			(SELECT COUNT(*) FROM challenge_response WHERE challenge_id = c.id) as response_count
		FROM challenge c
		WHERE c.user_id = $1
		ORDER BY c.created_at DESC
	`
	rows, err := userStore.DB.Query(challengesQuery, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var summary UserChallengeSummary
		err := rows.Scan(
			&summary.Name, &summary.UpdatedAt, &summary.CreatedAt, &summary.Category,
			&summary.PopularScore, &summary.CommentCount, &summary.ResponseCount,
		)
		if err != nil {
			return nil, err
		}
		activityData.Challenges = append(activityData.Challenges, summary)
	}

	// 3. Get user's challenge responses
	responsesQuery := `
		SELECT id, name, up_vote, down_vote, created_at, updated_at
		FROM challenge_response
		WHERE user_id = $1
		ORDER BY created_at DESC
	`
	rows, err = userStore.DB.Query(responsesQuery, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var summary UserResponseSummary
		err := rows.Scan(
			&summary.ID, &summary.Name, &summary.UpVote, &summary.DownVote,
			&summary.CreatedAt, &summary.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		activityData.ChallengeResponses = append(activityData.ChallengeResponses, summary)
	}

	return &activityData, nil
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

func (userStore *DBUserStore) GetUserName(userID string) (userName string, err error) {
	query := `
	SELECT username FROM "user" WHERE id = $1;
	`
	err = userStore.DB.QueryRow(query, userID).Scan(&userName)

	return userName, err

}

func (userStore *DBUserStore) LoginAndIssueTokens(user *User) (accessToken, refreshToken, csrfToken string, err error) {

	var (
		userID     string
		userName   string
		errMessage string
	)

	switch {
	case user.GoogleID != "":
		err = userStore.DB.QueryRow(`SELECT id, username FROM "user" WHERE google_id = $1`, user.GoogleID).Scan(&userID, &userName)
		errMessage = "Your google auth has problem"
	case user.GithubID != "":
		err = userStore.DB.QueryRow(`SELECT id, username FROM "user" WHERE github_id = $1`, user.GithubID).Scan(&userID, &userName)
		errMessage = "Your github auth has problem"
	case user.Email != "" && user.Password.PlainText != "":
		var hashed sql.NullString

		err = userStore.DB.QueryRow(`SELECT id, username, password FROM "user" WHERE email = $1`, user.Email).Scan(&userID, &userName, &hashed)

		errMessage = "Either password is not correct or user email is not found"

		if errors.Is(err, sql.ErrNoRows) {
			// cannot find user in database
			return "", "", "", utils.NewCustomAppError(constants.InvalidData, errMessage)
		}

		// user does not have a password
		if !hashed.Valid {
			return "", "", "", utils.NewCustomAppError(constants.InvalidData, "Please login through google or github")
		}

		match, cmpErr := argon2id.ComparePasswordAndHash(user.Password.PlainText, hashed.String)
		if cmpErr != nil {
			return "", "", "", cmpErr
		}
		if !match {
			return "", "", "", utils.NewCustomAppError(constants.InvalidData, errMessage)
		}

	default:
		panic("user_store > Missing login credentials while login")
	}

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// cannot find user in database
			return "", "", "", utils.NewCustomAppError(constants.InvalidData, errMessage)
		}
		return "", "", "", err
	}

	user.ID = userID

	accessToken, refreshToken, err = utils.CreateTokens(userID, userName, 0)

	result, err := utils.ExtractClaimsFromJWT(refreshToken, []string{constants.JWTRefreshTokenID})
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
