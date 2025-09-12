package api_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"testing"
)

// CleanDB resets the database to a clean state before a test run.
func CleanDB(db *sql.DB) {
	// "user" table connects to EVERY other table, so by truncate user we also clean all the other tables
	db.Exec(`TRUNCATE TABLE "user" RESTART IDENTITY CASCADE`)
}

// MakeRequestAndExpectStatus is a test helper that builds and sends an HTTP request,
// then asserts that the response status code matches the expected value.
func MakeRequestAndExpectStatus(t *testing.T, client *http.Client, method, urlStr string, payload map[string]string, expectedStatus int) []byte {
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest(method, urlStr, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	if client.Jar != nil {
		u, err := url.Parse(urlStr)
		if err != nil {
			t.Errorf("Parse url string failed: %v", err)
		}
		fmt.Println("client jar: ", client.Jar)

		cookies := client.Jar.Cookies(u)

		// Look for CSRF token
		for _, cookie := range cookies {
			if cookie.Name == "csrfToken" {
				req.Header.Set("X-CSRF-Token", cookie.Value)
				break
			}
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Errorf("Request failed: %v", err)
	}
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status: %d, body: %s", resp.StatusCode, string(respBody))
	resp.Body.Close()

	if resp.StatusCode != expectedStatus {
		t.Errorf("Expected status %d, got %v", expectedStatus, resp.Status)
	}

	return respBody
}

// TestRequest defines the parameters for a single HTTP request to be made during a test.
type TestRequest struct {
	method string
	path   string
	body   map[string]string
}

// TestStep represents a single step in a table-driven test scenario.
// It contains the test name, the request to perform, the status code that is
// expected, and an optional function to perform custom validation on the response body.
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
// 	defer application.ConnectionPool.Close()
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
