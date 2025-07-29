package api_test

import (
	// "encoding/json"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/RichardHoa/hack-me/internal/app"
	"github.com/RichardHoa/hack-me/internal/routes"
)

func TestCommentRoute(t *testing.T) {
	// Initialize application and server
	application, err := app.NewApplication(true)
	if err != nil {
		t.Fatalf("failed to create application: %v", err)
	}
	defer application.ConnectionPool.Close()
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
			name: "Set up",
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
					name: "Sign up second user",
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
			},
		},
		{
			name: "challenge comment",
			steps: []TestStep{
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
					name: "comment to challenge",
					request: TestRequest{
						method: "POST",
						path:   "/v1/comments",
						body: map[string]string{
							"challengeID": "1",
							"content":     "comment from the first user, level 1",
						},
					},
					expectStatus: http.StatusCreated,
				},
				{
					name: "comment to challenge level 2",
					request: TestRequest{
						method: "POST",
						path:   "/v1/comments",
						body: map[string]string{
							"parentID":    "1",
							"challengeID": "1",
							"content":     "comment from the first user, level 2",
						},
					},
					expectStatus: http.StatusCreated,
				},
				{
					name: "comment to challenge level 3",
					request: TestRequest{
						method: "POST",
						path:   "/v1/comments",
						body: map[string]string{
							"parentID":    "2",
							"challengeID": "1",
							"content":     "comment from the first user, level 3",
						},
					},
					expectStatus: http.StatusCreated,
				},

				{
					name: "comment to challenge level 4",
					request: TestRequest{
						method: "POST",
						path:   "/v1/comments",
						body: map[string]string{
							"parentID":    "3",
							"challengeID": "1",
							"content":     "comment from the first user, level 4",
						},
					},
					expectStatus: http.StatusCreated,
				},
				{
					name: "comment to challenge level 5",
					request: TestRequest{
						method: "POST",
						path:   "/v1/comments",
						body: map[string]string{
							"parentID":    "4",
							"challengeID": "1",
							"content":     "comment from the first user, level 5",
						},
					},
					expectStatus: http.StatusCreated,
				},
				{
					name: "Verify that general get challenge does not produce any comment",
					request: TestRequest{
						method: "GET",
						path:   "/v1/challenges",
						body:   map[string]string{},
					},
					expectStatus: http.StatusOK,
					validate: func(t *testing.T, body []byte) {
						var parsed map[string]any
						if err := json.Unmarshal(body, &parsed); err != nil {
							t.Errorf("Failed to parse response: %v", err)
						}

						data, ok := parsed["data"].([]any)
						if !ok {
							t.Errorf(`Expected "data" to be a list, got: %#v`, parsed["data"])
						}

						for _, item := range data {
							challenge, ok := item.(map[string]any)
							if !ok {
								continue
							}
							if challenge["comment"] != nil {
								t.Errorf("expected comment key to be null but get %v", challenge["comment"])

							}
						}

					},
				},
				{
					name: "Verify nested comments up to level 5",
					request: TestRequest{
						method: "GET",
						path:   fmt.Sprintf("/v1/challenges?exactName=%s", url.QueryEscape("Undefeated challenge")),
						body:   map[string]string{},
					},
					expectStatus: http.StatusOK,
					validate: func(t *testing.T, body []byte) {
						var parsed map[string]any
						if err := json.Unmarshal(body, &parsed); err != nil {
							t.Errorf("Failed to parse response: %v", err)
						}

						data, ok := parsed["data"].([]any)
						if !ok {
							t.Errorf(`Expected "data" to be a list, got: %#v`, parsed["data"])
						}

						for _, item := range data {
							challenge, ok := item.(map[string]any)
							if !ok {
								continue
							}

							comments, ok := challenge["comments"].([]any)
							if !ok || len(comments) == 0 {
								t.Fatal("Expected comments to exist and not be empty")
							}

							// Verify level 1 comment
							verifyNestedComment(t, comments, 0, 1)

							// Recursively verify nested comments up to level 5
							currentComments := comments
							for level := 2; level <= 5; level++ {
								if len(currentComments) == 0 {
									t.Errorf("Missing level %d comment", level)
								}

								firstComment := currentComments[0].(map[string]any)
								if replies, exists := firstComment["comments"]; exists {
									if replyList, ok := replies.([]any); ok && len(replyList) > 0 {
										verifyNestedComment(t, replyList, 0, level)
										currentComments = replyList
										continue
									}
								}
								t.Errorf("Level %d comment missing or invalid structure", level)
							}
						}
					},
				},

				{
					name: "Edit first level comment",
					request: TestRequest{
						method: "PUT",
						path:   "/v1/comments",
						body: map[string]string{
							//WRONG DOING HERE
							"commentID": "1",
							"content":   "New comment that has been edited | level 1",
						},
					},
					expectStatus: http.StatusOK,
				},
				{
					name: "Verify first level comment was edited",
					request: TestRequest{
						method: "GET",
						path:   fmt.Sprintf("/v1/challenges?exactName=%s", url.QueryEscape("Undefeated challenge")),
						body:   map[string]string{},
					},
					expectStatus: http.StatusOK,
					validate:     verifyEditedComment(1),
				},

				{
					name: "Edit second level comment",
					request: TestRequest{
						method: "PUT",
						path:   "/v1/comments",
						body: map[string]string{
							"commentID": "2",
							"content":   "New comment that has been edited | level 2",
						},
					},
					expectStatus: http.StatusOK,
				},
				{
					name: "Verify second level comment was edited",
					request: TestRequest{
						method: "GET",
						path:   fmt.Sprintf("/v1/challenges?exactName=%s", url.QueryEscape("Undefeated challenge")),
						body:   map[string]string{},
					},
					expectStatus: http.StatusOK,
					validate:     verifyEditedComment(2),
				},

				{
					name: "Edit third level comment",
					request: TestRequest{
						method: "PUT",
						path:   "/v1/comments",
						body: map[string]string{
							"commentID": "3",
							"content":   "New comment that has been edited | level 3",
						},
					},
					expectStatus: http.StatusOK,
				},
				{
					name: "Verify third level comment was edited",
					request: TestRequest{
						method: "GET",
						path:   fmt.Sprintf("/v1/challenges?exactName=%s", url.QueryEscape("Undefeated challenge")),
						body:   map[string]string{},
					},
					expectStatus: http.StatusOK,
					validate:     verifyEditedComment(3),
				},

				{
					name: "Edit fourth level comment",
					request: TestRequest{
						method: "PUT",
						path:   "/v1/comments",
						body: map[string]string{
							"commentID": "4",
							"content":   "New comment that has been edited | level 4",
						},
					},
					expectStatus: http.StatusOK,
				},
				{
					name: "Verify fourth level comment was edited",
					request: TestRequest{
						method: "GET",
						path:   fmt.Sprintf("/v1/challenges?exactName=%s", url.QueryEscape("Undefeated challenge")),
						body:   map[string]string{},
					},
					expectStatus: http.StatusOK,
					validate:     verifyEditedComment(4),
				},

				{
					name: "Edit fifth level comment",
					request: TestRequest{
						method: "PUT",
						path:   "/v1/comments",
						body: map[string]string{
							"commentID": "5",
							"content":   "New comment that has been edited | level 5",
						},
					},
					expectStatus: http.StatusOK,
				},
				{
					name: "Verify fifth level comment was edited",
					request: TestRequest{
						method: "GET",
						path:   fmt.Sprintf("/v1/challenges?exactName=%s", url.QueryEscape("Undefeated challenge")),
						body:   map[string]string{},
					},
					expectStatus: http.StatusOK,
					validate:     verifyEditedComment(5),
				},
			},
		},
		{
			name: "challenge response comment",
			steps: []TestStep{
				{
					name: "Login first user",
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
					name: "comment to challenge response",
					request: TestRequest{
						method: "POST",
						path:   "/v1/comments",
						body: map[string]string{
							"challengeResponseID": "1",
							"content":             "comment from the first user, level 1",
						},
					},
					expectStatus: http.StatusCreated,
				},
				{
					name: "comment to challenge response level 2",
					request: TestRequest{
						method: "POST",
						path:   "/v1/comments",
						body: map[string]string{
							"parentID":            "6",
							"challengeResponseID": "1",
							"content":             "comment from the first user, level 2",
						},
					},
					expectStatus: http.StatusCreated,
				},
				{
					name: "comment to challenge response level 3",
					request: TestRequest{
						method: "POST",
						path:   "/v1/comments",
						body: map[string]string{
							"parentID":            "7",
							"challengeResponseID": "1",
							"content":             "comment from the first user, level 3",
						},
					},
					expectStatus: http.StatusCreated,
				},
				{
					name: "comment to challenge response level 4",
					request: TestRequest{
						method: "POST",
						path:   "/v1/comments",
						body: map[string]string{
							"parentID":            "8",
							"challengeResponseID": "1",
							"content":             "comment from the first user, level 4",
						},
					},
					expectStatus: http.StatusCreated,
				},
				{
					name: "comment to challenge response level 5",
					request: TestRequest{
						method: "POST",
						path:   "/v1/comments",
						body: map[string]string{
							"parentID":            "9",
							"challengeResponseID": "1",
							"content":             "comment from the first user, level 5",
						},
					},
					expectStatus: http.StatusCreated,
				},

				{
					name: "GET",
					request: TestRequest{
						method: "GET",
						path:   fmt.Sprintf("/v1/challenges/responses?challengeID=%s", url.QueryEscape("1")),
						body:   map[string]string{},
					},
					expectStatus: http.StatusOK,
				},

				{
					name: "Verify nested comments up to level 5",
					request: TestRequest{
						method: "GET",
						path:   fmt.Sprintf("/v1/challenges/responses?challengeID=%s", url.QueryEscape("1")),
						body:   map[string]string{},
					},
					expectStatus: http.StatusOK,
					validate: func(t *testing.T, body []byte) {
						var parsed map[string]any
						if err := json.Unmarshal(body, &parsed); err != nil {
							t.Errorf("Failed to parse response: %v", err)
						}

						data, ok := parsed["data"].([]any)
						if !ok {
							t.Errorf(`Expected "data" to be a list, got: %#v`, parsed["data"])
						}

						for _, item := range data {
							challenge, ok := item.(map[string]any)
							if !ok {
								continue
							}

							comments, ok := challenge["comments"].([]any)
							if !ok || len(comments) == 0 {
								t.Fatal("Expected comments to exist and not be empty")
							}

							// Verify level 1 comment
							verifyNestedComment(t, comments, 0, 1)

							// Recursively verify nested comments up to level 5
							currentComments := comments
							for level := 2; level <= 5; level++ {
								if len(currentComments) == 0 {
									t.Errorf("Missing level %d comment", level)
								}

								firstComment := currentComments[0].(map[string]any)
								if replies, exists := firstComment["comments"]; exists {
									if replyList, ok := replies.([]any); ok && len(replyList) > 0 {
										verifyNestedComment(t, replyList, 0, level)
										currentComments = replyList
										continue
									}
								}
								t.Errorf("Level %d comment missing or invalid structure", level)
							}
						}
					},
				},
				{
					name: "Edit first level comment",
					request: TestRequest{
						method: "PUT",
						path:   "/v1/comments",
						body: map[string]string{
							"commentID": "6",
							"content":   "New comment that has been edited | level 1",
						},
					},
					expectStatus: http.StatusOK,
				},

				{
					name: "Verify first level comment was edited",
					request: TestRequest{
						method: "GET",
						path:   fmt.Sprintf("/v1/challenges/responses?challengeID=%s", url.QueryEscape("1")),
						body:   map[string]string{},
					},
					expectStatus: http.StatusOK,
					validate:     verifyEditedComment(1),
				},

				{
					name: "Edit second level comment",
					request: TestRequest{
						method: "PUT",
						path:   "/v1/comments",
						body: map[string]string{
							"commentID": "7",
							"content":   "New comment that has been edited | level 2",
						},
					},
					expectStatus: http.StatusOK,
				},
				{
					name: "Verify second level comment was edited",
					request: TestRequest{
						method: "GET",
						path:   fmt.Sprintf("/v1/challenges/responses?challengeID=%s", url.QueryEscape("1")),
						body:   map[string]string{},
					},
					expectStatus: http.StatusOK,
					validate:     verifyEditedComment(2),
				},

				{
					name: "Edit third level comment",
					request: TestRequest{
						method: "PUT",
						path:   "/v1/comments",
						body: map[string]string{
							"commentID": "8",
							"content":   "New comment that has been edited | level 3",
						},
					},
					expectStatus: http.StatusOK,
				},
				{
					name: "Verify third level comment was edited",
					request: TestRequest{
						method: "GET",
						path:   fmt.Sprintf("/v1/challenges/responses?challengeID=%s", url.QueryEscape("1")),
						body:   map[string]string{},
					},
					expectStatus: http.StatusOK,
					validate:     verifyEditedComment(3),
				},

				{
					name: "Edit fourth level comment",
					request: TestRequest{
						method: "PUT",
						path:   "/v1/comments",
						body: map[string]string{
							"commentID": "9",
							"content":   "New comment that has been edited | level 4",
						},
					},
					expectStatus: http.StatusOK,
				},
				{
					name: "Verify fourth level comment was edited",
					request: TestRequest{
						method: "GET",
						path:   fmt.Sprintf("/v1/challenges/responses?challengeID=%s", url.QueryEscape("1")),
						body:   map[string]string{},
					},
					expectStatus: http.StatusOK,
					validate:     verifyEditedComment(4),
				},

				{
					name: "Edit fifth level comment",
					request: TestRequest{
						method: "PUT",
						path:   "/v1/comments",
						body: map[string]string{
							"commentID": "10",
							"content":   "New comment that has been edited | level 5",
						},
					},
					expectStatus: http.StatusOK,
				},
				{
					name: "Verify fifth level comment was edited",
					request: TestRequest{
						method: "GET",
						path:   fmt.Sprintf("/v1/challenges/responses?challengeID=%s", url.QueryEscape("1")),
						body:   map[string]string{},
					},
					expectStatus: http.StatusOK,
					validate:     verifyEditedComment(5),
				},
			},
		},
		{
			name: "Forbidden request",
			steps: []TestStep{
				{
					name: "Login second user",
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
					name: "Cannot delete comment user does not own",
					request: TestRequest{
						method: "DELETE",
						path:   "/v1/comments",
						body: map[string]string{
							"commentID": "1",
						},
					},
					expectStatus: http.StatusForbidden,
				},
				{
					name: "Cannot modify comment user does not own",
					request: TestRequest{
						method: "PUT",
						path:   "/v1/comments",
						body: map[string]string{
							"commentID": "1",
							"content":   "This is a new comment from user 2",
						},
					},
					expectStatus: http.StatusForbidden,
				},
			},
		},
		{
			name: "delete comments",
			steps: []TestStep{
				{
					name: "Login first user",
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
					name: "delete first level comment with challenge",
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
					name: "Verify there is no comment left",
					request: TestRequest{
						method: "GET",
						path:   fmt.Sprintf("/v1/challenges?exactName=%s", url.QueryEscape("Undefeated challenge")),
						body:   map[string]string{},
					},
					expectStatus: http.StatusOK,
					validate: func(t *testing.T, body []byte) {
						var parsed map[string]any
						if err := json.Unmarshal(body, &parsed); err != nil {
							t.Errorf("Failed to parse response: %v", err)
						}

						data, ok := parsed["data"].([]any)
						if !ok {
							t.Errorf(`Expected "data" to be a list, got: %#v`, parsed["data"])
						}

						for _, item := range data {
							challenge, ok := item.(map[string]any)
							if !ok {
								continue
							}

							if challenge["comments"] != nil {
								t.Fatal("Expect comment to be nil")

							}

						}
					},
				},
				{
					name: "delete first level comment with challenge response",
					request: TestRequest{
						method: "DELETE",
						path:   "/v1/comments",
						body: map[string]string{
							"commentID": "6",
						},
					},
					expectStatus: http.StatusOK,
				},
				{
					name: "Verify there is no comment left",
					request: TestRequest{
						method: "GET",
						path:   fmt.Sprintf("/v1/challenges/responses?challengeID=%s", url.QueryEscape("1")),
						body:   map[string]string{},
					},
					expectStatus: http.StatusOK,
					validate: func(t *testing.T, body []byte) {
						var parsed map[string]any
						if err := json.Unmarshal(body, &parsed); err != nil {
							t.Errorf("Failed to parse response: %v", err)
						}

						data, ok := parsed["data"].([]any)
						if !ok {
							t.Errorf(`Expected "data" to be a list, got: %#v`, parsed["data"])
						}

						for _, item := range data {
							challenge, ok := item.(map[string]any)
							if !ok {
								continue
							}

							if challenge["comments"] != nil {
								t.Fatal("Expect comment to be nil")

							}

						}
					},
				},
			},
		},
		{
			name: "invalid request",
			steps: []TestStep{
				{
					name: "Login first user",
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
					name: "Empty content",
					request: TestRequest{
						method: "POST",
						path:   "/v1/comments",
						body: map[string]string{
							"challengeID": "1",
							"content":     "", // Empty content
						},
					},
					expectStatus: http.StatusBadRequest,
				},
				{
					name: "Missing challengeID",
					request: TestRequest{
						method: "POST",
						path:   "/v1/comments",
						body: map[string]string{
							// Missing challengeID
							"content": "valid content",
						},
					},
					expectStatus: http.StatusBadRequest,
				},
				{
					name: "Invalid challengeID format",
					request: TestRequest{
						method: "POST",
						path:   "/v1/comments",
						body: map[string]string{
							"challengeID": "not-a-number", // Invalid ID format
							"content":     "valid content",
						},
					},
					expectStatus: http.StatusBadRequest,
				},

				{
					name: "Invalid parent ID format",
					request: TestRequest{
						method: "POST",
						path:   "/v1/comments",
						body: map[string]string{
							"challengeID": "1",
							"parentID":    "invalid", // Bad parent ID
							"content":     "valid content",
						},
					},
					expectStatus: http.StatusBadRequest,
				},
				{
					name: "Non-existent challengeID",
					request: TestRequest{
						method: "POST",
						path:   "/v1/comments",
						body: map[string]string{
							"challengeID": "999999", // Non-existent challenge
							"content":     "valid content",
						},
					},
					expectStatus: http.StatusNotFound,
				},
				{
					name: "Extra unexpected fields",
					request: TestRequest{
						method: "POST",
						path:   "/v1/comments",
						body: map[string]string{
							"challengeID": "1",
							"content":     "valid content",
							"injected":    "malicious data", // Unexpected field
						},
					},
					expectStatus: http.StatusBadRequest,
				},

				{
					name: "Empty content",
					request: TestRequest{
						method: "PUT",
						path:   "/v1/comments",
						body: map[string]string{
							"commentID": "1",
							"content":   "", // Empty content
						},
					},
					expectStatus: http.StatusBadRequest,
				},
				{
					name: "Missing commentID",
					request: TestRequest{
						method: "PUT",
						path:   "/v1/comments",
						body: map[string]string{
							"content": "valid content",
						},
					},
					expectStatus: http.StatusBadRequest,
				},
				{
					name: "Invalid commentID format",
					request: TestRequest{
						method: "PUT",
						path:   "/v1/comments",
						body: map[string]string{
							"commentID": "not-a-number", // Invalid ID format
							"content":   "valid content",
						},
					},
					expectStatus: http.StatusBadRequest,
				},
				{
					name: "Non-existent commentID",
					request: TestRequest{
						method: "PUT",
						path:   "/v1/comments",
						body: map[string]string{
							"commentID": "999999", // Non-existent challenge
							"content":   "valid content",
						},
					},
					expectStatus: http.StatusNotFound,
				},
				{
					name: "Extra unexpected fields",
					request: TestRequest{
						method: "PUT",
						path:   "/v1/comments",
						body: map[string]string{
							"challengeID": "1",
							"content":     "valid content",
							"injected":    "malicious data", // Unexpected field
						},
					},
					expectStatus: http.StatusBadRequest,
				},

				{
					name: "Empty commentID",
					request: TestRequest{
						method: "DELETE",
						path:   "/v1/comments",
						body: map[string]string{
							"commentID": "",
						},
					},
					expectStatus: http.StatusBadRequest,
				},
				{
					name: "Missing commentID",
					request: TestRequest{
						method: "DELETE",
						path:   "/v1/comments",
						body:   map[string]string{},
					},
					expectStatus: http.StatusBadRequest,
				},
				{
					name: "Invalid commentID format",
					request: TestRequest{
						method: "DELETE",
						path:   "/v1/comments",
						body: map[string]string{
							"commentID": "not-a-number",
						},
					},
					expectStatus: http.StatusBadRequest,
				},
				{
					name: "Non-existent commentID",
					request: TestRequest{
						method: "DELETE",
						path:   "/v1/comments",
						body: map[string]string{
							"commentID": "999999",
						},
					},
					expectStatus: http.StatusNotFound,
				},
				{
					name: "Extra unexpected fields",
					request: TestRequest{
						method: "DELETE",
						path:   "/v1/comments",
						body: map[string]string{
							"commentID": "1",
							"injected":  "malicious data",
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

func verifyNestedComment(t *testing.T, comments []any, index int, level int) {
	if index >= len(comments) {
		t.Errorf("Comment index %d out of bounds for level %d", index, level)
	}

	comment, ok := comments[index].(map[string]any)
	if !ok {
		t.Errorf("Invalid comment structure at level %d", level)
	}

	createdAt, createdExists := comment["createdAt"].(string)
	updatedAt, updatedExists := comment["updatedAt"].(string)

	if !createdExists || !updatedExists {
		t.Fatal("Missing createdAt or updatedAt fields")
	}

	if createdAt != updatedAt {
		t.Errorf("Timestamps should be the same after first init (createdAt: %s, updatedAt: %s)",
			createdAt, updatedAt)
	}

	expectedContent := fmt.Sprintf("comment from the first user, level %d", level)
	if comment["content"] != expectedContent {
		t.Errorf("Expected content '%s' at level %d, got '%s'",
			expectedContent, level, comment["content"])
	}

	if comment["author"] != "Richard Hoa" {
		t.Errorf("Expected author 'Richard Hoa' at level %d, got '%s'",
			level, comment["author"])
	}
}

func verifyEditedComment(level int) func(*testing.T, []byte) {
	return func(t *testing.T, body []byte) {
		var parsed map[string]any
		if err := json.Unmarshal(body, &parsed); err != nil {
			t.Errorf("Failed to parse response: %v", err)
		}

		data, ok := parsed["data"].([]any)
		if !ok {
			t.Errorf(`Expected "data" to be a list, got: %#v`, parsed["data"])
		}

		found := false
		for _, item := range data {
			challenge, ok := item.(map[string]any)
			if !ok {
				continue
			}

			comments, ok := challenge["comments"].([]any)
			if !ok {
				continue
			}

			// Recursively search for the comment at the specified level
			if comment := findNestedComment(comments, level, 1); comment != nil {
				found = true
				expectedContent := fmt.Sprintf("New comment that has been edited | level %d", level)
				if comment["content"] != expectedContent {
					t.Errorf("Expected edited content '%s', got '%s'",
						expectedContent, comment["content"])
				}

				verifyTimestamps(t, comment)
			}
		}

		if !found {
			t.Errorf("Level %d comment not found in response", level)
		}
	}
}

func findNestedComment(comments []any, targetLevel, currentLevel int) map[string]any {
	if currentLevel > targetLevel {
		return nil
	}

	for _, c := range comments {
		comment, ok := c.(map[string]any)
		if !ok {
			continue
		}

		if currentLevel == targetLevel {
			return comment
		}

		if replies, exists := comment["comments"]; exists {
			if replyList, ok := replies.([]any); ok {
				if found := findNestedComment(replyList, targetLevel, currentLevel+1); found != nil {
					return found
				}
			}
		}
	}
	return nil
}

func verifyTimestamps(t *testing.T, comment map[string]any) {
	createdAt, createdExists := comment["createdAt"].(string)
	updatedAt, updatedExists := comment["updatedAt"].(string)

	if !createdExists || !updatedExists {
		t.Fatal("Missing createdAt or updatedAt fields")
	}

	if createdAt == updatedAt {
		t.Errorf("Timestamps should differ after edit (createdAt: %s, updatedAt: %s)",
			createdAt, updatedAt)
	}

	createdTime, err := time.Parse(time.RFC3339, createdAt)
	if err != nil {
		t.Errorf("Invalid createdAt format: %v", err)
	}

	updatedTime, err := time.Parse(time.RFC3339, updatedAt)
	if err != nil {
		t.Errorf("Invalid updatedAt format: %v", err)
	}

	if !updatedTime.After(createdTime) {
		t.Errorf("updatedAt should be after createdAt (created: %v, updated: %v)",
			createdTime, updatedTime)
	}
}
