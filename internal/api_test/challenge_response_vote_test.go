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

func TestChallengeResponseVoteRoute(t *testing.T) {
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
					name: "Get challenge response",
					request: TestRequest{
						method: "GET",
						path:   "/v1/challenges/responses",
						body: map[string]string{
							"challengeID": "1",
						},
					},
					expectStatus: http.StatusOK,
				},
				{
					name: "missing challengeResponseID",
					request: TestRequest{
						method: "POST",
						path:   "/v1/challenges/responses/votes",
						body: map[string]string{
							"voteType": "upVote",
						},
					},
					expectStatus: http.StatusBadRequest,
				},
				{
					name: "missing voteType",
					request: TestRequest{
						method: "POST",
						path:   "/v1/challenges/responses/votes",
						body: map[string]string{
							"challengeResponseID": "1",
						},
					},
					expectStatus: http.StatusBadRequest,
				},
				{
					name: "empty both fields",
					request: TestRequest{
						method: "POST",
						path:   "/v1/challenges/responses/votes",
						body: map[string]string{
							"challengeResponseID": "",
							"voteType":            "",
						},
					},
					expectStatus: http.StatusBadRequest,
				},
				{
					name: "challengeResponseID empty string",
					request: TestRequest{
						method: "POST",
						path:   "/v1/challenges/responses/votes",
						body: map[string]string{
							"challengeResponseID": "",
							"voteType":            "upVote",
						},
					},
					expectStatus: http.StatusBadRequest,
				},
				{
					name: "voteType empty string",
					request: TestRequest{
						method: "POST",
						path:   "/v1/challenges/responses/votes",
						body: map[string]string{
							"challengeResponseID": "1",
							"voteType":            "",
						},
					},
					expectStatus: http.StatusBadRequest,
				},
				{
					name: "challengeResponseID whitespace",
					request: TestRequest{
						method: "POST",
						path:   "/v1/challenges/responses/votes",
						body: map[string]string{
							"challengeResponseID": "   ",
							"voteType":            "upVote",
						},
					},
					expectStatus: http.StatusBadRequest,
				},
				{
					name: "voteType whitespace",
					request: TestRequest{
						method: "POST",
						path:   "/v1/challenges/responses/votes",
						body: map[string]string{
							"challengeResponseID": "1",
							"voteType":            "   ",
						},
					},
					expectStatus: http.StatusBadRequest,
				},
				{
					name: "both fields whitespace",
					request: TestRequest{
						method: "POST",
						path:   "/v1/challenges/responses/votes",
						body: map[string]string{
							"challengeResponseID": "   ",
							"voteType":            "   ",
						},
					},
					expectStatus: http.StatusBadRequest,
				},
				{
					name: "invalid voteType value",
					request: TestRequest{
						method: "POST",
						path:   "/v1/challenges/responses/votes",
						body: map[string]string{
							"challengeResponseID": "1",
							"voteType":            "invalidVote",
						},
					},
					expectStatus: http.StatusBadRequest,
				},
				{
					name: "non-existent challengeResponseID",
					request: TestRequest{
						method: "POST",
						path:   "/v1/challenges/responses/votes",
						body: map[string]string{
							"challengeResponseID": "99999",
							"voteType":            "upVote",
						},
					},
					expectStatus: http.StatusNotFound,
				},
				{
					name: "valid up vote",
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
					name: "Verify valid up vote",
					request: TestRequest{
						method: "GET",
						path:   "/v1/challenges/responses",
						body: map[string]string{
							"challengeID": "1",
						},
					},
					expectStatus: http.StatusOK,
					validate: func(t *testing.T, body []byte) {
						var parsed map[string]any
						if err := json.Unmarshal(body, &parsed); err != nil {
							t.Fatalf("Failed to parse response: %v", err)
						}

						data, ok := parsed["data"].([]any)
						if !ok {
							t.Fatalf(`Expected "data" to be a list, got: %#v`, parsed["data"])
						}

						found := false
						for _, item := range data {
							response, ok := item.(map[string]any)
							if !ok {
								continue
							}

							if response["upVote"] == "1" {
								found = true
								break
							}
						}

						if !found {
							t.Errorf("Expect upvote to be 1, but found %v", data)
						}
					},
				},

				{
					name: "valid down vote",
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
					name: "Verify down vote",
					request: TestRequest{
						method: "GET",
						path:   "/v1/challenges/responses",
						body: map[string]string{
							"challengeID": "1",
						},
					},
					expectStatus: http.StatusOK,
					validate: func(t *testing.T, body []byte) {
						var parsed map[string]any
						if err := json.Unmarshal(body, &parsed); err != nil {
							t.Fatalf("Failed to parse response: %v", err)
						}

						data, ok := parsed["data"].([]any)
						if !ok {
							t.Fatalf(`Expected "data" to be a list, got: %#v`, parsed["data"])
						}

						found := false
						for _, item := range data {
							response, ok := item.(map[string]any)
							if !ok {
								continue
							}

							if response["upVote"] == "0" && response["downVote"] == "1" {
								found = true
								break
							}
						}

						if !found {
							t.Errorf("Expect upvote to be 0 and downVote to be 1, but found %v", data)
						}
					},
				},
				{
					name: "Non existant challengeResponseID",
					request: TestRequest{
						method: "DELETE",
						path:   "/v1/challenges/responses/votes",
						body: map[string]string{
							"challengeResponseID": "999",
						},
					},
					expectStatus: http.StatusNotFound,
				},
				{
					name: "missing challengeResponseID",
					request: TestRequest{
						method: "DELETE",
						path:   "/v1/challenges/responses/votes",
						body:   map[string]string{},
					},
					expectStatus: http.StatusBadRequest,
				},
				{
					name: "empty challengeResponseID",
					request: TestRequest{
						method: "DELETE",
						path:   "/v1/challenges/responses/votes",
						body: map[string]string{
							"challengeResponseID": "",
						},
					},
					expectStatus: http.StatusBadRequest,
				},
				{
					name: "invalid challengeResponseID format (non-numeric)",
					request: TestRequest{
						method: "DELETE",
						path:   "/v1/challenges/responses/votes",
						body: map[string]string{
							"challengeResponseID": "abc",
						},
					},
					expectStatus: http.StatusBadRequest,
				},
				{
					name: "invalid challengeResponseID format (special chars)",
					request: TestRequest{
						method: "DELETE",
						path:   "/v1/challenges/responses/votes",
						body: map[string]string{
							"challengeResponseID": "1'; DROP TABLE votes;--",
						},
					},
					expectStatus: http.StatusBadRequest,
				},
				{
					name: "valid delete vote",
					request: TestRequest{
						method: "DELETE",
						path:   "/v1/challenges/responses/votes",
						body: map[string]string{
							"challengeResponseID": "1",
						},
					},
					expectStatus: http.StatusOK,
				},
				{
					name: "Verify delete vote",
					request: TestRequest{
						method: "GET",
						path:   "/v1/challenges/responses",
						body: map[string]string{
							"challengeID": "1",
						},
					},
					expectStatus: http.StatusOK,
					validate: func(t *testing.T, body []byte) {
						var parsed map[string]any
						if err := json.Unmarshal(body, &parsed); err != nil {
							t.Fatalf("Failed to parse response: %v", err)
						}

						data, ok := parsed["data"].([]any)
						if !ok {
							t.Fatalf(`Expected "data" to be a list, got: %#v`, parsed["data"])
						}

						found := false
						for _, item := range data {
							response, ok := item.(map[string]any)
							if !ok {
								continue
							}

							if response["upVote"] == "0" && response["downVote"] == "0" {
								found = true
								break
							}
						}

						if !found {
							t.Errorf("Expect upvote to be 0 and downVote to be 0, but found %v", data)
						}
					},
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
							"userName":  "Richard Hoa 2",
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
					name: "vote that user has not made",
					request: TestRequest{
						method: "DELETE",
						path:   "/v1/challenges/responses/votes",
						body: map[string]string{
							"challengeResponseID": "1",
						},
					},
					expectStatus: http.StatusNotFound,
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
