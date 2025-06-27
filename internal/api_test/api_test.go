package api_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/RichardHoa/hack-me/internal/app"
	"github.com/RichardHoa/hack-me/internal/routes"
)

func cleanDB(db *sql.DB) {
	db.Exec(`TRUNCATE TABLE "user" RESTART IDENTITY CASCADE`)
	// Repeat for other tables
}

func makeRequestAndExpectStatus(t *testing.T, url string, payload map[string]string, expectedStatus int) {
	body, _ := json.Marshal(payload)
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("→ status: %d, body: %s", resp.StatusCode, string(respBody))

	if resp.StatusCode != expectedStatus {
		t.Errorf("Expected status %d, got %d", expectedStatus, resp.StatusCode)
	}
}

func TestUserSignUp(t *testing.T) {
	application, err := app.NewApplication(true)
	if err != nil {
		t.Fatalf("failed to create application: %v", err)
	}
	defer application.DB.Close()
	defer cleanDB(application.DB)

	router := routes.SetUpRoutes(application)
	server := httptest.NewServer(router)
	defer server.Close()

	tests := []struct {
		name           string
		payload        map[string]string
		expectedStatus int
	}{
		{
			name: "Valid signup",
			payload: map[string]string{
				"user_name":  "Richard Hoa",
				"password":   "ThisIsAVerySEcurePasswordThatWon'tBeStop",
				"email":      "testEmail@gmail.com",
				"image_link": "example.image.com",
				"google_id":  "",
				"github_id":  "",
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "Duplicate username and email",
			payload: map[string]string{
				"user_name":  "Richard Hoa",
				"password":   "ThisIsAVerySEcurePasswordThatWon'tBeStop",
				"email":      "testEmail@gmail.com",
				"image_link": "example.image.com",
				"google_id":  "",
				"github_id":  "",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Weak password",
			payload: map[string]string{
				"user_name":  "AnotherUser",
				"password":   "HelloThere",
				"email":      "another@gmail.com",
				"image_link": "img.com",
				"google_id":  "",
				"github_id":  "",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Empty password field",
			payload: map[string]string{
				"user_name":  "fourth user",
				"password":   "",
				"email":      "anothertest@gmail.com",
				"image_link": "img.com",
				"google_id":  "",
				"github_id":  "",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Password + Google ID",
			payload: map[string]string{
				"user_name":  "pwg_user",
				"password":   "SuperSecure123ExtremelySecure",
				"email":      "pwg_user@gmail.com",
				"image_link": "",
				"google_id":  "google-uid-123",
				"github_id":  "",
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "Password + GitHub ID",
			payload: map[string]string{
				"user_name":  "pwh_user",
				"password":   "AnotherSecurePass!UnberableSecurePassswordYouCan't",
				"email":      "pwh_user@gmail.com",
				"image_link": "",
				"google_id":  "",
				"github_id":  "github-uid-321",
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "Google ID + GitHub ID",
			payload: map[string]string{
				"user_name":  "gg_user",
				"password":   "",
				"email":      "gg_user@gmail.com",
				"image_link": "",
				"google_id":  "google-uid-456",
				"github_id":  "github-uid-654",
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "All three auth fields present",
			payload: map[string]string{
				"user_name":  "full_user",
				"password":   "AnotherPass123ExtremelyStuffPassword",
				"email":      "full_user@gmail.com",
				"image_link": "",
				"google_id":  "google-uid-789",
				"github_id":  "github-uid-987",
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "Duplicate username",
			payload: map[string]string{
				"user_name":  "full_user", // from previous test
				"password":   "AnotherPass123ExtremelyStuffPassword",
				"email":      "dup_user1@gmail.com",
				"image_link": "",
				"google_id":  "",
				"github_id":  "",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Duplicate email",
			payload: map[string]string{
				"user_name":  "dup_user2",
				"password":   "AnotherPass123ExtremelyStuffPassword",
				"email":      "full_user@gmail.com", // same email as above
				"image_link": "",
				"google_id":  "",
				"github_id":  "",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Duplicate Google ID",
			payload: map[string]string{
				"user_name":  "dup_user3",
				"password":   "AnotherPass123",
				"email":      "dup_google@gmail.com",
				"image_link": "",
				"google_id":  "google-uid-789", // same as above
				"github_id":  "",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Duplicate GitHub ID",
			payload: map[string]string{
				"user_name":  "dup_user4",
				"password":   "AnotherPass123",
				"email":      "dup_github@gmail.com",
				"image_link": "",
				"google_id":  "",
				"github_id":  "github-uid-987", // same as above
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for i, tc := range tests {
		t.Run(fmt.Sprintf("%02d-%s", i+1, tc.name), func(t *testing.T) {
			makeRequestAndExpectStatus(t, server.URL+"/users", tc.payload, tc.expectedStatus)
		})
	}
}
