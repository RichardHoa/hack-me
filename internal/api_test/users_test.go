package api_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"runtime/debug"
	"testing"

	"github.com/RichardHoa/hack-me/internal/app"
	"github.com/RichardHoa/hack-me/internal/routes"
)

func TestUserRoutes(t *testing.T) {
	application, err := app.NewApplication(true)
	if err != nil {
		t.Fatalf("failed to create application: %v", err)
	}
	defer application.ConnectionPool.Close()
	defer CleanDB(application.DB)

	router := routes.SetUpRoutes(application)
	server := httptest.NewServer(router)
	defer server.Close()

	tests := []struct {
		name  string
		steps []TestStep
	}{
		{
			name: "set up",
			steps: []TestStep{
				{
					name: "Valid sign up",
					request: TestRequest{
						method: "POST",
						path:   "/v1/users",
						body: map[string]string{
							"userName":  "Richard Hoa",
							"password":  "StrongSecurePasswordThatWon'tBemarkAsInvalid",
							"email":     "testEmail@gmail.com",
							"imageLink": "example.image.com",
							"googleID":  "",
							"githubID":  "",
						},
					},
					expectStatus: http.StatusCreated,
				},

				{
					name: "Valid log in",
					request: TestRequest{
						method: "POST",
						path:   "/v1/users/login",
						body: map[string]string{
							"password": "StrongSecurePasswordThatWon'tBemarkAsInvalid",
							"email":    "testEmail@gmail.com",
						},
					},
					expectStatus: http.StatusOK,
				},
			},
		},
		{
			name: "user_page",
			steps: []TestStep{
				{
					name: "Sign up user for activity test",
					request: TestRequest{
						method: "POST",
						path:   "/v1/users",
						body: map[string]string{
							"userName":  "activityUser",
							"password":  "PasswordForActivityTest",
							"email":     "activity@test.com",
							"imageLink": "activity.image.com",
						},
					},
					expectStatus: http.StatusCreated,
				},
				{
					name: "Login activity user",
					request: TestRequest{
						method: "POST",
						path:   "/v1/users/login",
						body: map[string]string{
							"email":    "activity@test.com",
							"password": "PasswordForActivityTest",
						},
					},
					expectStatus: http.StatusOK,
				},
				{
					name: "Create a challenge for activity user",
					request: TestRequest{
						method: "POST",
						path:   "/v1/challenges",
						body: map[string]string{
							"name":     "Activity Challenge 1",
							"content":  "Content for activity challenge",
							"category": "web hacking",
						},
					},
					expectStatus: http.StatusCreated,
				},
				{
					name: "Create a challenge response for activity user",
					request: TestRequest{
						method: "POST",
						path:   "/v1/challenges/responses",
						body: map[string]string{
							"challengeID": "1",
							"name":        "Activity Response 1",
							"content":     "Content for activity response",
						},
					},
					expectStatus: http.StatusCreated,
				},
				{
					name: "Get user activity",
					request: TestRequest{
						method: "GET",
						path:   "/v1/users/me",
					},
					expectStatus: http.StatusOK,

					validate: func(t *testing.T, body []byte) {
						var resp struct {
							Data struct {
								User struct {
									UserName  string `json:"userName"`
									ImageLink string `json:"imageLink"`
								} `json:"user"`
								Challenges []struct {
									Name          string `json:"name"`
									Category      string `json:"category"`
									CommentCount  string `json:"commentCount"`
									ResponseCount string `json:"responseCount"`
									PopularScore  string `json:"popularScore"`
								} `json:"challenges"`
								ChallengeResponses []struct {
									Name     string `json:"name"`
									UpVote   string `json:"upVote"`
									DownVote string `json:"downVote"`
								} `json:"challengeResponses"`
							} `json:"data"`
						}

						if err := json.Unmarshal(body, &resp); err != nil {
							t.Errorf("Failed to parse response: %v", err)
						}

						// Check user data
						if resp.Data.User.UserName != "activityUser" {
							t.Errorf("Expected username 'activityUser', got '%s'", resp.Data.User.UserName)
						}
						if resp.Data.User.ImageLink != "activity.image.com" {
							t.Errorf("Expected image link 'activity.image.com', got '%s'", resp.Data.User.ImageLink)
						}

						// Check challenges
						if len(resp.Data.Challenges) != 1 {
							t.Errorf("Expected 1 challenge, got %d", len(resp.Data.Challenges))
						}
						challenge := resp.Data.Challenges[0]
						if challenge.Name != "Activity Challenge 1" {
							t.Errorf("Expected challenge name 'Activity Challenge 1', got '%s'", challenge.Name)
						}
						if challenge.Category != "web hacking" {
							t.Errorf("Expected category 'web hacking', got '%s'", challenge.Category)
						}
						if challenge.CommentCount != "0" {
							t.Errorf("Expected comment count 0, got %v", challenge.CommentCount)
						}
						if challenge.ResponseCount != "1" {
							t.Errorf("Expected response count 1, got %v", challenge.ResponseCount)
						}
						if challenge.PopularScore != "0" {
							t.Errorf("Expected popular score 0, got %v", challenge.PopularScore)
						}

						// Check challenge responses
						if len(resp.Data.ChallengeResponses) != 1 {
							t.Errorf("Expected 1 challenge response, got %d", len(resp.Data.ChallengeResponses))
						}
						response := resp.Data.ChallengeResponses[0]
						if response.Name != "Activity Response 1" {
							t.Errorf("Expected response name 'Activity Response 1', got '%s'", response.Name)
						}
						if response.UpVote != "0" {
							t.Errorf("Expected up votes 0, got %v", response.UpVote)
						}
						if response.DownVote != "0" {
							t.Errorf("Expected down votes 0, got %v", response.DownVote)
						}
					},
				},
				{
					name: "Upvote the response",
					request: TestRequest{
						method: "POST",
						path:   "/v1/challenges/responses/votes",
						body: map[string]string{
							"challengeResponseID": "1",
							"voteType":            "upVote",
						},
					},
					expectStatus: http.StatusCreated,
				},
				{
					name: "Verify upvote",
					request: TestRequest{
						method: "GET",
						path:   "/v1/users/me",
					},
					expectStatus: http.StatusOK,
					validate: func(t *testing.T, body []byte) {
						var resp struct {
							Data struct {
								ChallengeResponses []struct {
									UpVote   string `json:"upVote"`
									DownVote string `json:"downVote"`
								} `json:"challengeResponses"`
							} `json:"data"`
						}
						json.Unmarshal(body, &resp)
						response := resp.Data.ChallengeResponses[0]
						if response.UpVote != "1" {
							t.Errorf("Expected up votes '1', got '%s'", response.UpVote)
						}
						if response.DownVote != "0" {
							t.Errorf("Expected down votes '0', got '%s'", response.DownVote)
						}
					},
				},
				{
					name: "Downvote the response",
					request: TestRequest{
						method: "POST",
						path:   "/v1/challenges/responses/votes",
						body: map[string]string{
							"challengeResponseID": "1",
							"voteType":            "downVote",
						},
					},
					expectStatus: http.StatusCreated,
				},
				{
					name: "Verify downvote",
					request: TestRequest{
						method: "GET",
						path:   "/v1/users/me",
					},
					expectStatus: http.StatusOK,
					validate: func(t *testing.T, body []byte) {
						var resp struct {
							Data struct {
								ChallengeResponses []struct {
									UpVote   string `json:"upVote"`
									DownVote string `json:"downVote"`
								} `json:"challengeResponses"`
							} `json:"data"`
						}
						json.Unmarshal(body, &resp)
						response := resp.Data.ChallengeResponses[0]
						if response.UpVote != "0" {
							t.Errorf("Expected up votes '0', got '%s'", response.UpVote)
						}
						if response.DownVote != "1" {
							t.Errorf("Expected down votes '1', got '%s'", response.DownVote)
						}
					},
				},
				{
					name: "Add a comment to the challenge",
					request: TestRequest{
						method: "POST",
						path:   "/v1/comments",
						body: map[string]string{
							"challengeID": "1",
							"content":     "This is a test comment.",
						},
					},
					expectStatus: http.StatusCreated,
				},
				{
					name: "Verify comment count",
					request: TestRequest{
						method: "GET",
						path:   "/v1/users/me",
					},
					expectStatus: http.StatusOK,
					validate: func(t *testing.T, body []byte) {
						var resp struct {
							Data struct {
								Challenges []struct {
									CommentCount string `json:"commentCount"`
								} `json:"challenges"`
							} `json:"data"`
						}
						json.Unmarshal(body, &resp)
						challenge := resp.Data.Challenges[0]
						if challenge.CommentCount != "1" {
							t.Errorf("Expected comment count '1', got '%s'", challenge.CommentCount)
						}
					},
				},
				{
					name: "Delete the challenge comment",
					request: TestRequest{
						method: "DELETE",
						path:   "/v1/comments",
						body: map[string]string{
							"commentID": "1",
						},
					},
					expectStatus: http.StatusOK,
				},
				{
					name: "Verify challenge comment count is zero",
					request: TestRequest{
						method: "GET",
						path:   "/v1/users/me",
					},
					expectStatus: http.StatusOK,
					validate: func(t *testing.T, body []byte) {
						var resp struct {
							Data struct {
								Challenges []struct {
									CommentCount string `json:"commentCount"`
								} `json:"challenges"`
							} `json:"data"`
						}
						json.Unmarshal(body, &resp)
						challenge := resp.Data.Challenges[0]
						if challenge.CommentCount != "0" {
							t.Errorf("Expected comment count '0' after deletion, got '%s'", challenge.CommentCount)
						}
					},
				},
				{
					name: "Change username",
					request: TestRequest{
						method: "PUT",
						path:   "/v1/users/username",
						body:   map[string]string{"newUsername": "newActivityUser"},
					},
					expectStatus: http.StatusOK,
				},
				{
					name: "Verify username change",
					request: TestRequest{
						method: "GET",
						path:   "/v1/users/me",
					},
					expectStatus: http.StatusOK,
					validate: func(t *testing.T, body []byte) {
						var resp map[string]map[string]interface{}
						if err := json.Unmarshal(body, &resp); err != nil {
							t.Errorf("Failed to parse response: %v", err)
						}
						user := resp["data"]["user"].(map[string]interface{})
						if user["userName"] != "newActivityUser" {
							t.Errorf("Expected username 'newActivityUser', got '%s'", user["userName"])
						}
					},
				},
				{
					name: "Change password",
					request: TestRequest{
						method: "PUT",
						path:   "/v1/users/password",
						body: map[string]string{
							"oldPassword": "PasswordForActivityTest",
							"newPassword": "NewPasswordForActivity",
						},
					},
					expectStatus: http.StatusOK,
				},
				{
					name: "Login with new password",
					request: TestRequest{
						method: "POST",
						path:   "/v1/users/login",
						body: map[string]string{
							"email":    "activity@test.com",
							"password": "NewPasswordForActivity",
						},
					},
					expectStatus: http.StatusOK,
				},
				{
					name: "Delete user",
					request: TestRequest{
						method: "DELETE",
						path:   "/v1/users/me",
					},
					expectStatus: http.StatusOK,
				},
				{
					name: "Verify user deletion by trying to log in",
					request: TestRequest{
						method: "POST",
						path:   "/v1/users/login",
						body: map[string]string{
							"email":    "activity@test.com",
							"password": "NewPasswordForActivity",
						},
					},
					expectStatus: http.StatusBadRequest,
				},
			},
		},

		{
			name: "Social login",
			steps: []TestStep{
				{
					name: "Sign up social user (no password)",
					request: TestRequest{
						method: "POST",
						path:   "/v1/users",
						body: map[string]string{
							"userName":  "socialUser",
							"email":     "social@test.com",
							"googleID":  "social-google-id-123",
							"imageLink": "Random image link",
						},
					},
					expectStatus: http.StatusCreated,
				},
				{
					name: "Login social user",
					request: TestRequest{
						method: "POST",
						path:   "/v1/users/login",
						body: map[string]string{
							"email":    "social@test.com",
							"googleID": "social-google-id-123",
						},
					},
					expectStatus: http.StatusOK,
				},
				{
					name: "Set password for social user",
					request: TestRequest{
						method: "PUT",
						path:   "/v1/users/password",
						body: map[string]string{
							"oldPassword": "", // old password is empty
							"newPassword": "PasswordForSocialUser",
						},
					},
					expectStatus: http.StatusOK,
				},
				{
					name: "Login with newly set password",
					request: TestRequest{
						method: "POST",
						path:   "/v1/users/login",
						body: map[string]string{
							"email":    "social@test.com",
							"password": "PasswordForSocialUser",
						},
					},
					expectStatus: http.StatusOK,
				},
			},
		},

		{
			name: "Malicious",
			steps: []TestStep{
				{
					name: "Duplicated user name",
					request: TestRequest{
						method: "POST",
						path:   "/v1/users",
						body: map[string]string{
							"userName":  "Richard Hoa",
							"password":  "StrongSecurePasswordThatWon'tBemarkAsInvalid",
							"email":     "testEmail@gmail.com",
							"imageLink": "example.image.com",
							"googleID":  "",
							"githubID":  "",
						},
					},
					expectStatus: http.StatusBadRequest,
				},
				{
					name: "Lacking password",
					request: TestRequest{
						method: "POST",
						path:   "/v1/users",
						body: map[string]string{
							"userName":  "fourth user",
							"password":  "",
							"email":     "anothertest@gmail.com",
							"imageLink": "img.com",
							"googleID":  "",
							"githubID":  "",
						},
					},
					expectStatus: http.StatusBadRequest,
				},
				{
					name: "Lacking email",
					request: TestRequest{
						method: "POST",
						path:   "/v1/users",
						body: map[string]string{
							"userName":  "lacking_email_user_1",
							"password":  "StrongSecurePasswordThatWon'tBemarkAsInvalid",
							"email":     "",
							"imageLink": "",
							"googleID":  "",
							"githubID":  "github-uid-very-unique",
						},
					},
					expectStatus: http.StatusBadRequest,
				},
				{
					name: "Lacking username",
					request: TestRequest{
						method: "POST",
						path:   "/v1/users",
						body: map[string]string{
							"userName":  "",
							"password":  "StrongSecurePasswordThatWon'tBemarkAsInvalid",
							"email":     "lackingusername@gmail.com",
							"imageLink": "",
							"googleID":  "",
							"githubID":  "",
						},
					},
					expectStatus: http.StatusBadRequest,
				},
				{
					name: "Password and Google ID",
					request: TestRequest{
						method: "POST",
						path:   "/v1/users",
						body: map[string]string{
							"userName":  "pwg_user",
							"password":  "StrongSecurePasswordThatWon'tBemarkAsInvalid",
							"email":     "pwg_user@gmail.com",
							"imageLink": "",
							"googleID":  "google-uid-123",
							"githubID":  "",
						},
					},
					expectStatus: http.StatusCreated,
				},
				{
					name: "Password and Github ID",
					request: TestRequest{
						method: "POST",
						path:   "/v1/users",
						body: map[string]string{
							"userName":  "pwh_user",
							"password":  "StrongSecurePasswordThatWon'tBemarkAsInvalid",
							"email":     "pwh_user@gmail.com",
							"imageLink": "",
							"googleID":  "",
							"githubID":  "github-uid-321",
						},
					},
					expectStatus: http.StatusCreated,
				},
				{
					name: "Google ID and GitHub ID",
					request: TestRequest{
						method: "POST",
						path:   "/v1/users",
						body: map[string]string{
							"userName":  "gg_user",
							"password":  "",
							"email":     "gg_user@gmail.com",
							"imageLink": "",
							"googleID":  "google-uid-456",
							"githubID":  "github-uid-654",
						},
					},
					expectStatus: http.StatusCreated,
				},
				{
					name: "Password, Google ID and GitHub ID",
					request: TestRequest{
						method: "POST",
						path:   "/v1/users",
						body: map[string]string{
							"userName":  "full_user",
							"password":  "StrongSecurePasswordThatWon'tBemarkAsInvalid",
							"email":     "full_user@gmail.com",
							"imageLink": "",
							"googleID":  "google-uid-789",
							"githubID":  "github-uid-987",
						},
					},
					expectStatus: http.StatusCreated,
				},
				{
					name: "Duplicate username",
					request: TestRequest{
						method: "POST",
						path:   "/v1/users",
						body: map[string]string{
							"userName":  "full_user",
							"password":  "StrongSecurePasswordThatWon'tBemarkAsInvalid",
							"email":     "dup_user1@gmail.com",
							"imageLink": "",
							"googleID":  "",
							"githubID":  "",
						},
					},
					expectStatus: http.StatusBadRequest,
				},
				{
					name: "Duplicate email",
					request: TestRequest{
						method: "POST",
						path:   "/v1/users",
						body: map[string]string{
							"userName":  "dup_user2",
							"password":  "StrongSecurePasswordThatWon'tBemarkAsInvalid",
							"email":     "full_user@gmail.com",
							"imageLink": "",
							"googleID":  "",
							"githubID":  "",
						},
					},
					expectStatus: http.StatusBadRequest,
				},
				{
					name: "Duplicate Google ID",
					request: TestRequest{
						method: "POST",
						path:   "/v1/users",
						body: map[string]string{
							"userName":  "dup_user3",
							"password":  "StrongSecurePasswordThatWon'tBemarkAsInvalid",
							"email":     "dup_google@gmail.com",
							"imageLink": "",
							"googleID":  "google-uid-789",
							"githubID":  "",
						},
					},
					expectStatus: http.StatusBadRequest,
				},
				{
					name: "Duplicate GitHub ID",
					request: TestRequest{
						method: "POST",
						path:   "/v1/users",
						body: map[string]string{
							"userName":  "dup_user4",
							"password":  "StrongSecurePasswordThatWon'tBemarkAsInvalid",
							"email":     "dup_github@gmail.com",
							"imageLink": "",
							"googleID":  "",
							"githubID":  "github-uid-987",
						},
					},
					expectStatus: http.StatusBadRequest,
				},
				{
					name: "Setup user for malicious tests",
					request: TestRequest{
						method: "POST",
						path:   "/v1/users",
						body: map[string]string{
							"userName": "malicious_user",
							"password": "AValidPassword123",
							"email":    "malicious@test.com",
						},
					},
					expectStatus: http.StatusCreated,
				},
				{
					name: "Login malicious user",
					request: TestRequest{
						method: "POST",
						path:   "/v1/users/login",
						body: map[string]string{
							"email":    "malicious@test.com",
							"password": "AValidPassword123",
						},
					},
					expectStatus: http.StatusOK,
				},

				// --- Change Username Endpoint ---
				{
					name: "empty string",
					request: TestRequest{
						method: "PUT",
						path:   "/v1/users/username",
						body:   map[string]string{"newUsername": ""},
					},
					expectStatus: http.StatusBadRequest,
				},
				{
					name: "whitespace",
					request: TestRequest{
						method: "PUT",
						path:   "/v1/users/username",
						body:   map[string]string{"newUsername": "   "},
					},
					expectStatus: http.StatusBadRequest,
				},
				{
					name: "a very long string",
					request: TestRequest{
						method: "PUT",
						path:   "/v1/users/username",
						body:   map[string]string{"newUsername": "aVeryLongUsernameThatIsDefinitelyOverAnyReasonableLimitAndShouldBeRejectedByTheServerValidationLogicPleaseRejectMe"},
					},
					expectStatus: http.StatusOK,
				},
				{
					name: "SQL injection attempt",
					request: TestRequest{
						method: "PUT",
						path:   "/v1/users/username",
						body:   map[string]string{"newUsername": "' OR 1=1; --"},
					},
					// Should be accepted as a string, but the test ensures it doesn't cause a 500 error.
					expectStatus: http.StatusOK,
				},
				{
					name: "existing name",
					request: TestRequest{
						method: "PUT",
						path:   "/v1/users/username",
						body:   map[string]string{"newUsername": "Richard Hoa"}, // From the first test case
					},
					expectStatus: http.StatusBadRequest,
				},
				{
					name: "extra unexpected fields",
					request: TestRequest{
						method: "PUT",
						path:   "/v1/users/username",
						body:   map[string]string{"newUsername": "someUser", "isAdmin": "true"},
					},
					expectStatus: http.StatusBadRequest,
				},

				// --- Change Password Endpoint ---
				{
					name: "empty new password",
					request: TestRequest{
						method: "PUT",
						path:   "/v1/users/password",
						body: map[string]string{
							"oldPassword": "AValidPassword123",
							"newPassword": "",
						},
					},
					expectStatus: http.StatusBadRequest,
				},
				{
					name: "short password",
					request: TestRequest{
						method: "PUT",
						path:   "/v1/users/password",
						body: map[string]string{
							"oldPassword": "AValidPassword123",
							"newPassword": "short",
						},
					},
					expectStatus: http.StatusBadRequest,
				},
				{
					name: "pwned password",
					request: TestRequest{
						method: "PUT",
						path:   "/v1/users/password",
						body: map[string]string{
							"oldPassword": "AValidPassword123",
							"newPassword": "password",
						},
					},
					expectStatus: http.StatusBadRequest,
				},
				{
					name: "incorrect old password",
					request: TestRequest{
						method: "PUT",
						path:   "/v1/users/password",
						body: map[string]string{
							"oldPassword": "WrongOldPassword",
							"newPassword": "ANewValidPassword123",
						},
					},
					expectStatus: http.StatusBadRequest,
				},
				{
					name: "extra unexpected fields",
					request: TestRequest{
						method: "PUT",
						path:   "/v1/users/password",
						body: map[string]string{
							"oldPassword":    "AValidPassword123",
							"newPassword":    "ANewValidPassword123",
							"userIdToUpdate": "some-other-user-id",
						},
					},
					expectStatus: http.StatusBadRequest,
				},

				// --- Delete User Endpoint ---
				{
					name: "Delete user",
					request: TestRequest{
						method: "DELETE",
						path:   "/v1/users/me",
					},
					expectStatus: http.StatusOK,
				},
				{
					name: "Verify user is gone by accessing authenticated route",
					request: TestRequest{
						method: "GET",
						path:   "/v1/users/me",
					},
					// Cookies are cleared on delete, so this should fail auth.
					expectStatus: http.StatusUnauthorized,
				},
			},
		},
	}

	for _, test := range tests {
		jar, _ := cookiejar.New(nil)
		client := &http.Client{Jar: jar}

		test := test

		t.Run(test.name, func(t *testing.T) {
			for _, step := range test.steps {
				t.Run(fmt.Sprintf("%s-%s-%d-%s", step.request.method, step.request.path, step.expectStatus, step.name), func(t *testing.T) {
					body := MakeRequestAndExpectStatus(t, client, step.request.method, server.URL+step.request.path, step.request.body, step.expectStatus)

					if step.validate != nil {
						step.validate(t, body)
					}
				})
			}
		})
	}

}

func FuzzUserLogin(f *testing.F) {
	logFile, err := os.OpenFile("fuzz_login.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		f.Logf("Failed to open debug log file: %v", err)
	}
	defer logFile.Close()
	logger := log.New(logFile, "", log.LstdFlags)

	// --- One-Time Setup ---
	application, err := app.NewApplication(true)
	if err != nil {
		logger.Printf("failed to create application: %v", err)
	}
	f.Cleanup(func() {
		application.ConnectionPool.Close()
	})

	router := routes.SetUpRoutes(application)

	// Pre-create a known valid user for testing
	validEmail := "fuzz_login@example.com"
	validPassword := "a-very-Strong-and-Valid-Password-123"

	createUserBody, _ := json.Marshal(map[string]string{
		"userName":  "login_fuzz_user",
		"password":  validPassword,
		"email":     validEmail,
		"imageLink": "", "googleID": "", "githubID": "",
	})

	// Use in-process POST to /v1/users
	createReq := httptest.NewRequest("POST", "/v1/users", bytes.NewReader(createUserBody))
	createReq.Header.Set("Content-Type", "application/json")
	createResp := httptest.NewRecorder()
	router.ServeHTTP(createResp, createReq)

	if createResp.Code != http.StatusCreated && createResp.Code != http.StatusOK {
		logger.Printf("Failed to create valid test user, got status: %s with message %v", createResp.Result().Status, createResp.Body)
	}

	f.Add(validEmail, validPassword)
	f.Add("invalid@example.com", "wrong-password")
	f.Add("fuzz@example.com", "")
	f.Add("", "some-password")

	f.Fuzz(func(t *testing.T, email, password string) {
		defer func() {
			if r := recover(); r != nil {
				stack := debug.Stack()
				logger.Printf("🔥 PANIC recovered for input: email=%q, password=%q\n%v\n%s", email, password, r, stack)
				t.Errorf("panic occurred during fuzzing, recovered")
			}
		}()

		loginBodyMap := map[string]string{"email": email, "password": password}
		jsonBody, err := json.Marshal(loginBodyMap)
		if err != nil {
			logger.Printf("❌ Failed to marshal input: %v", err)
			t.Errorf("JSON marshal error: %v", err)
			return
		}

		req := httptest.NewRequest("POST", "/v1/users/login", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		resp := rec.Result()
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusInternalServerError {
			var body struct {
				Message string `json:"message"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
				logger.Printf("[ERROR] Failed to decode body: %v", err)
			} else {
				logger.Printf(`
					[FUZZ CASE]
					status:   %s
					message:  %q
					email:    %q
					password: %q
						`, resp.Status, body.Message, email, password)
			}
		}

		if email == validEmail && password == validPassword {
			if resp.StatusCode != http.StatusOK {
				logger.Printf("❌ Expected success for valid credentials, got %s", resp.Status)
				t.Errorf("Expected success for valid credentials, got %s", resp.Status)
			}
		} else {
			if resp.StatusCode == http.StatusOK {
				logger.Printf("❌ Unexpected success for email=%q password=%q", email, password)
				t.Errorf("Unexpected success for email=%q password=%q", email, password)
			}
		}
	})
}

func FuzzUserSignUp(f *testing.F) {
	logFile, err := os.OpenFile("fuzz_signup.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		f.Fatalf("Failed to open debug log file: %v", err)
	}
	defer logFile.Close()
	logger := log.New(logFile, "", log.LstdFlags)

	application, err := app.NewApplication(true)
	if err != nil {
		logger.Fatalf("failed to create application: %v", err)
	}
	f.Cleanup(func() {
		application.ConnectionPool.Close()
	})

	router := routes.SetUpRoutes(application)

	f.Add("valid_user", "valid_user@example.com", "ValidPassword123")
	f.Add("test_user", "not-an-email", "short")
	f.Add("", "user@example.com", "password")
	f.Add("user", "", "password")
	f.Add("user", "user@example.com", "")
	f.Add("another_user", "valid_user@example.com", "some_password")
	f.Add("!@#$%^", "email@!@#$.com", "p@$$w()rd")

	f.Fuzz(func(t *testing.T, userName, email, password string) {
		defer func() {
			if r := recover(); r != nil {
				stack := debug.Stack()
				logger.Printf("🔥 PANIC recovered for input: userName=%q, email=%q, password=%q\n%v\n%s", userName, email, password, r, stack)
				t.Errorf("panic occurred with input: userName=%q, email=%q, password=%q", userName, email, password)
			}
		}()

		signUpBody, err := json.Marshal(map[string]string{
			"userName":  userName,
			"email":     email,
			"password":  password,
			"imageLink": "",
			"googleID":  "",
			"githubID":  "",
		})
		if err != nil {
			t.Errorf("Failed to marshal fuzz input to JSON: %v", err)
		}

		req := httptest.NewRequest("POST", "/v1/users", bytes.NewReader(signUpBody))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		resp := rec.Result()
		defer resp.Body.Close()

		if resp.StatusCode >= http.StatusInternalServerError {
			var body struct {
				Message string `json:"message"`
			}
			json.NewDecoder(resp.Body).Decode(&body)

			logger.Printf(`
				[FUZZ FAILURE]
				status:   %s
				message:  %q
				userName: %q
				email:    %q
				password: %q
					`, resp.Status, body.Message, userName, email, password)

		}

	})
}
