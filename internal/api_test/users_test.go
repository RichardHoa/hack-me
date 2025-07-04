package api_test

import (
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"testing"

	"github.com/RichardHoa/hack-me/internal/app"
	"github.com/RichardHoa/hack-me/internal/routes"
)

func TestUserRoutes(t *testing.T) {
	application, err := app.NewApplication(true)
	if err != nil {
		t.Fatalf("failed to create application: %v", err)
	}
	defer application.DB.Close()
	defer CleanDB(application.DB)

	router := routes.SetUpRoutes(application)
	server := httptest.NewServer(router)
	defer server.Close()

	// Test cases
	tests := []struct {
		name  string
		steps []TestStep
	}{
		{
			name: "",
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
