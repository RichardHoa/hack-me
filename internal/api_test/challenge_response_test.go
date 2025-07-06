package api_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"testing"

	"github.com/RichardHoa/hack-me/internal/app"
	"github.com/RichardHoa/hack-me/internal/routes"
)

func TestChallengeResponseRoute(t *testing.T) {
	// Initialize application and server
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
			name: "First user",
			steps: []TestStep{
				{
					name: "Sign up valid user",
					request: TestRequest{
						method: "POST",
						path:   "/v1/users",
						body: map[string]string{
							"userName":  "Richard Hoa",
							"password":  "ThisIsAVerySEcurePasswordThatWon'tBeStop",
							"email":     "testEmail@gmail.com",
							"imageLink": "example.image.com",
						},
					},
					expectStatus: http.StatusCreated,
				},
				{
					name: "Login test user",
					request: TestRequest{
						method: "POST",
						path:   "/v1/users/login",
						body: map[string]string{
							"email":    "testEmail@gmail.com",
							"password": "ThisIsAVerySEcurePasswordThatWon'tBeStop",
						},
					},
					expectStatus: http.StatusOK,
				},
				{
					name: "Create valid challenge",
					request: TestRequest{
						method: "POST",
						path:   "/v1/challenges",
						body: map[string]string{
							"name":     "Undefeated challenge",
							"content":  "This is a very powerful challenge that no one will be able to defeat",
							"category": "web hacking",
						},
					},
					expectStatus: http.StatusCreated,
				},
				{
					name: "New challenge response",
					request: TestRequest{
						method: "POST",
						path:   "/v1/challenges/responses",
						body: map[string]string{
							"challengeID": "1",
							"name":        "I find a method to hack into the challenge",
							"content":     "blbla bla",
						},
					},
					expectStatus: http.StatusCreated,
				},
				{
					name: "Missing challengeID",
					request: TestRequest{
						method: "POST",
						path:   "/v1/challenges/responses",
						body: map[string]string{
							"name":    "response name",
							"content": "some content",
						},
					},
					expectStatus: http.StatusBadRequest,
				},
				{
					name: "Missing name",
					request: TestRequest{
						method: "POST",
						path:   "/v1/challenges/responses",
						body: map[string]string{
							"challengeID": "1",
							"content":     "some content",
						},
					},
					expectStatus: http.StatusBadRequest,
				},
				{
					name: "Missing content",
					request: TestRequest{
						method: "POST",
						path:   "/v1/challenges/responses",
						body: map[string]string{
							"challengeID": "1",
							"name":        "valid name",
						},
					},
					expectStatus: http.StatusBadRequest,
				},
				{
					name: "All fields empty",
					request: TestRequest{
						method: "POST",
						path:   "/v1/challenges/responses",
						body: map[string]string{
							"challengeID": "",
							"name":        "",
							"content":     "",
						},
					},
					expectStatus: http.StatusBadRequest,
				},
				{
					name: "Empty name only",
					request: TestRequest{
						method: "POST",
						path:   "/v1/challenges/responses",
						body: map[string]string{
							"challengeID": "1",
							"name":        "",
							"content":     "some content",
						},
					},
					expectStatus: http.StatusBadRequest,
				},
				{
					name: "Empty content only",
					request: TestRequest{
						method: "POST",
						path:   "/v1/challenges/responses",
						body: map[string]string{
							"challengeID": "1",
							"name":        "valid name",
							"content":     "",
						},
					},
					expectStatus: http.StatusBadRequest,
				},
				{
					name: "Empty challengeID only",
					request: TestRequest{
						method: "POST",
						path:   "/v1/challenges/responses",
						body: map[string]string{
							"challengeID": "",
							"name":        "valid name",
							"content":     "valid content",
						},
					},
					expectStatus: http.StatusBadRequest,
				},
				{
					name: "Extra unexpected field in body",
					request: TestRequest{
						method: "POST",
						path:   "/v1/challenges/responses",
						body: map[string]string{
							"challengeID": "1",
							"name":        "valid name",
							"content":     "valid content",
							"extraField":  "oops",
						},
					},
					expectStatus: http.StatusBadRequest,
				},
				{
					name: "Verify the challenges response is there",
					request: TestRequest{
						method: "GET",
						path:   "/v1/challenges/responses",
						body: map[string]string{
							"challengeID": "1",
						},
					},
					validate: func(t *testing.T, body []byte) {
						var parsed map[string]any
						if err := json.Unmarshal(body, &parsed); err != nil {
							t.Fatalf("Failed to parse response: %v", err)
						}

						data, ok := parsed["data"].([]any)
						if !ok {
							t.Fatalf(`Expected "data" to be a list, got: %#v`, parsed["data"])
						}
						expectedName := "I find a method to hack into the challenge"
						expectedAuthor := "Richard Hoa"

						found := false
						for _, item := range data {
							response, ok := item.(map[string]any)
							if !ok {
								continue
							}

							if response["name"] == expectedName &&
								response["authorName"] == expectedAuthor {
								found = true
								break
							}
						}

						if !found {
							t.Errorf("Expected response not found: name=%q, authorName=%q", expectedName, expectedAuthor)
						}
					},
					expectStatus: http.StatusOK,
				},
				{
					name: "Modify challenge response",
					request: TestRequest{
						method: "PUT",
						path:   "/v1/challenges/responses",
						body: map[string]string{
							"challengeResponseID": "1",
							"name":                "This is a new name for the modify challenge response",
							"content":             "this hack works better",
						},
					},
					expectStatus: http.StatusOK,
				},
				{
					name: "Verify the challenge response has been modified",
					request: TestRequest{
						method: "GET",
						path:   "/v1/challenges/responses",
						body: map[string]string{
							"challengeID": "1",
						},
					},
					validate: func(t *testing.T, body []byte) {
						var parsed map[string]any
						if err := json.Unmarshal(body, &parsed); err != nil {
							t.Fatalf("Failed to parse response: %v", err)
						}

						data, ok := parsed["data"].([]any)
						if !ok {
							t.Fatalf(`Expected "data" to be a list, got: %#v`, parsed["data"])
						}
						expectedName := "This is a new name for the modify challenge response"
						expectedContent := "this hack works better"
						expectedAuthor := "Richard Hoa"

						found := false
						for _, item := range data {
							response, ok := item.(map[string]any)
							if !ok {
								continue
							}

							if response["name"] == expectedName &&
								response["authorName"] == expectedAuthor &&
								response["content"] == expectedContent {
								found = true
								break
							}
						}

						if !found {
							t.Errorf("Expected response not found: name=%q, authorName=%q, content=%q", expectedName, expectedAuthor, expectedContent)
						}
					},
					expectStatus: http.StatusOK,
				},
				{
					name: "Missing challengeResponseID",
					request: TestRequest{
						method: "PUT",
						path:   "/v1/challenges/responses",
						body: map[string]string{
							"name":    "Updated name",
							"content": "Updated content",
						},
					},
					expectStatus: http.StatusBadRequest,
				},
				{
					name: "Empty challengeResponseID",
					request: TestRequest{
						method: "PUT",
						path:   "/v1/challenges/responses",
						body: map[string]string{
							"challengeResponseID": "",
							"name":                "Updated name",
							"content":             "Updated content",
						},
					},
					expectStatus: http.StatusBadRequest,
				},
				{
					name: "Missing name and content",
					request: TestRequest{
						method: "PUT",
						path:   "/v1/challenges/responses",
						body: map[string]string{
							"challengeResponseID": "1",
						},
					},
					expectStatus: http.StatusBadRequest,
				},
				{
					name: "Empty name and content",
					request: TestRequest{
						method: "PUT",
						path:   "/v1/challenges/responses",
						body: map[string]string{
							"challengeResponseID": "1",
							"name":                "",
							"content":             "",
						},
					},
					expectStatus: http.StatusBadRequest,
				},
				{
					name: "Empty name only",
					request: TestRequest{
						method: "PUT",
						path:   "/v1/challenges/responses",
						body: map[string]string{
							"challengeResponseID": "1",
							"name":                "",
							"content":             "Still valid content",
						},
					},
					expectStatus: http.StatusBadRequest,
				},
				{
					name: "Empty content only",
					request: TestRequest{
						method: "PUT",
						path:   "/v1/challenges/responses",
						body: map[string]string{
							"challengeResponseID": "1",
							"name":                "Still valid name",
							"content":             "",
						},
					},
					expectStatus: http.StatusBadRequest,
				},
				{
					name: "Extra unexpected field",
					request: TestRequest{
						method: "PUT",
						path:   "/v1/challenges/responses",
						body: map[string]string{
							"challengeResponseID": "1",
							"name":                "valid name",
							"content":             "valid content",
							"extraField":          "should not be here",
						},
					},
					expectStatus: http.StatusBadRequest,
				},
				{
					name: "Delete challenge response",
					request: TestRequest{
						method: "DELETE",
						path:   "/v1/challenges/responses",
						body: map[string]string{
							"challengeResponseID": "1",
						},
					},
					expectStatus: http.StatusOK,
				},

				{
					name: "no challengeResponseID",
					request: TestRequest{
						method: "DELETE",
						path:   "/v1/challenges/responses",
						body:   map[string]string{},
					},
					expectStatus: http.StatusBadRequest,
				},

				{
					name: "empty challengeresponseID",
					request: TestRequest{
						method: "DELETE",
						path:   "/v1/challenges/responses",
						body: map[string]string{
							"challengeResponseID": "",
						},
					},
					expectStatus: http.StatusBadRequest,
				},

				{
					name: "extra unwanted field",
					request: TestRequest{
						method: "DELETE",
						path:   "/v1/challenges/responses",
						body: map[string]string{
							"malicious field": "oops",
						},
					},
					expectStatus: http.StatusBadRequest,
				},
				{
					name: "Verify the challenge has been deleted",
					request: TestRequest{
						method: "GET",
						path:   "/v1/challenges/responses",
						body: map[string]string{
							"challengeID": "1",
						},
					},
					validate: func(t *testing.T, body []byte) {
						var parsed map[string]any
						if err := json.Unmarshal(body, &parsed); err != nil {
							t.Fatalf("Failed to parse response: %v", err)
						}

						data, ok := parsed["data"].([]any)
						if !ok {
							t.Fatalf(`Expected "data" to be a list, got: %#v`, parsed["data"])
						}

						if len(data) != 0 {
							t.Fatalf(`Expected "data" to be empty but get %v`, data)
						}
					},
					expectStatus: http.StatusOK,
				},
				{
					name: "new challenge response",
					request: TestRequest{
						method: "POST",
						path:   "/v1/challenges/responses",
						body: map[string]string{
							"challengeID": "1",
							"name":        "challenge response from user 1",
							"content":     "bla bla bla content",
						},
					},
					expectStatus: http.StatusCreated,
				},
			},
		},
		{
			name: "Second user",
			steps: []TestStep{
				{
					name: "Sign up valid user",
					request: TestRequest{
						method: "POST",
						path:   "/v1/users",
						body: map[string]string{
							"userName":  "User 2",
							"password":  "ThisIsAVerySEcurePasswordThatWon'tBeStop",
							"email":     "testEmail2@gmail.com",
							"imageLink": "example.image.com",
						},
					},
					expectStatus: http.StatusCreated,
				},
				{
					name: "Login test user",
					request: TestRequest{
						method: "POST",
						path:   "/v1/users/login",
						body: map[string]string{
							"email":    "testEmail2@gmail.com",
							"password": "ThisIsAVerySEcurePasswordThatWon'tBeStop",
						},
					},
					expectStatus: http.StatusOK,
				},

				{
					name: "Cannot modify other challenge response",
					request: TestRequest{
						method: "PUT",
						path:   "/v1/challenges/responses",
						body: map[string]string{
							// this challenge response belongs to previous user
							"challengeResponseID": "2",
							"name":                "This is a new name for the modify challenge response",
							"content":             "this hack works better",
						},
					},
					expectStatus: http.StatusForbidden,
				},
				{
					name: "Cannot delete other challenge response",
					request: TestRequest{
						method: "DELETE",
						path:   "/v1/challenges/responses",
						body: map[string]string{
							"challengeResponseID": "2",
						},
					},
					expectStatus: http.StatusForbidden,
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
