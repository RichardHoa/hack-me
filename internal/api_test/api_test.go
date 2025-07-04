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
