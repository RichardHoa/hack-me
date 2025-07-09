package api_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"testing"
)

func CleanDB(db *sql.DB) {
	db.Exec(`TRUNCATE TABLE "user" RESTART IDENTITY CASCADE`)
	// Repeat for other tables
}

func MakeRequestAndExpectStatus(t *testing.T, client *http.Client, method, urlStr string, payload map[string]string, expectedStatus int) []byte {
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest(method, urlStr, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	// Check for CSRF token in cookies
	if client.Jar != nil {
		u, err := url.Parse(urlStr)
		if err == nil {
			for _, cookie := range client.Jar.Cookies(u) {
				if cookie.Name == "csrfToken" {
					req.Header.Set("X-CSRF-Token", cookie.Value)
					break
				}
			}
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status: %d, body: %s", resp.StatusCode, string(respBody))
	resp.Body.Close()

	if resp.StatusCode != expectedStatus {
		t.Errorf("Expected status %d, got %d", expectedStatus, resp.StatusCode)
	}

	return respBody
}

type TestRequest struct {
	method string
	path   string
	body   map[string]string
}

type TestStep struct {
	name         string
	request      TestRequest
	expectStatus int
	validate     func(t *testing.T, body []byte)
}

//NOTE: Example here

// import (
// 	"encoding/json"
// 	"fmt"
// 	"net/http"
// 	"net/http/cookiejar"
// 	"net/http/httptest"
// 	"testing"
//
// 	"github.com/RichardHoa/hack-me/internal/app"
// 	"github.com/RichardHoa/hack-me/internal/routes"
// )
//
// func TestExampleRoute(t *testing.T) {
// 	// Initialize application and server
// 	application, err := app.NewApplication(true)
// 	if err != nil {
// 		t.Fatalf("failed to create application: %v", err)
// 	}
// 	defer application.DB.Close()
// 	defer CleanDB(application.DB)
//
// 	router := routes.SetUpRoutes(application)
// 	server := httptest.NewServer(router)
// 	defer server.Close()
//
// 	// Test cases
// 	tests := []struct {
// 		name  string
// 		steps []TestStep
// 	}{
// 		{
// 			name: "First user",
// 			steps: []TestStep{
// 				{
// 					name: "Sign up valid user",
// 					request: TestRequest{
// 						method: "POST",
// 						path:   "/v1/users",
// 						body: map[string]string{
// 							"userName":  "Richard Hoa",
// 							"password":  "ThisIsAVerySEcurePasswordThatWon'tBeStop",
// 							"email":     "testEmail@gmail.com",
// 							"imageLink": "example.image.com",
// 						},
// 					},
// 					expectStatus: http.StatusCreated,
// 				},
// 				{
// 					name: "Login test user",
// 					request: TestRequest{
// 						method: "POST",
// 						path:   "/v1/users/login",
// 						body: map[string]string{
// 							"email":    "testEmail@gmail.com",
// 							"password": "ThisIsAVerySEcurePasswordThatWon'tBeStop",
// 						},
// 					},
// 					expectStatus: http.StatusOK,
// 				},
// 				{
// 					name: "Create valid challenge",
// 					request: TestRequest{
// 						method: "POST",
// 						path:   "/v1/challenges",
// 						body: map[string]string{
// 							"name":     "Undefeated challenge",
// 							"content":  "This is a very powerful challenge that no one will be able to defeat",
// 							"category": "web hacking",
// 						},
// 					},
// 					expectStatus: http.StatusCreated,
// 				},
// 				{
// 					name: "New challenge response",
// 					request: TestRequest{
// 						method: "POST",
// 						path:   "/v1/challenges/responses",
// 						body: map[string]string{
// 							"challengeID": "1",
// 							"name":        "I find a method to hack into the challenge",
// 							"content":     "blbla bla",
// 						},
// 					},
// 					expectStatus: http.StatusCreated,
// 				},
// 			},
// 		},
// 	}
//
// 	for _, test := range tests {
// 		jar, _ := cookiejar.New(nil)
// 		client := &http.Client{Jar: jar}
//
// 		test := test
//
// 		t.Run(test.name, func(t *testing.T) {
// 			for _, step := range test.steps {
// 				t.Run(fmt.Sprintf("%s-%s-%d-%s", step.request.method, step.request.path, step.expectStatus, step.name), func(t *testing.T) {
// 					body := MakeRequestAndExpectStatus(t, client, step.request.method, server.URL+step.request.path, step.request.body, step.expectStatus)
//
// 					if step.validate != nil {
// 						step.validate(t, body)
// 					}
// 				})
// 			}
// 		})
// 	}
//
// }
//
