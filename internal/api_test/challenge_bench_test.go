package api_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/RichardHoa/hack-me/internal/app"
	"github.com/RichardHoa/hack-me/internal/routes"
)

func BenchmarkCreateChallenge(b *testing.B) {
	// --- 1. One-Time Setup (Timer Stopped) ---
	b.StopTimer()
	application, _ := app.NewApplication(true)
	defer application.ConnectionPool.Close()
	defer CleanDB(application.DB)
	router := routes.SetUpRoutes(application)

	// Step A: Register User
	regBody, _ := json.Marshal(map[string]string{
		"userName": "Benchmarker",
		"email":    "bench@test.com",
		"password": "StrongPassword123!",
	})
	regReq := httptest.NewRequest("POST", "/v1/users", bytes.NewReader(regBody))
	router.ServeHTTP(httptest.NewRecorder(), regReq)

	// Step B: Login to get cookies
	loginBody, _ := json.Marshal(map[string]string{
		"email":    "bench@test.com",
		"password": "StrongPassword123!",
	})
	loginReq := httptest.NewRequest("POST", "/v1/users/login", bytes.NewReader(loginBody))
	loginRec := httptest.NewRecorder()
	router.ServeHTTP(loginRec, loginReq)

	// Capture cookies (tokens) for the challenge request
	cookies := loginRec.Result().Cookies()

	// --- 2. The Benchmark Loop ---
	for i := 0; i < b.N; i++ {
		// Prepare unique payload for this iteration
		payload := map[string]string{
			"name":     fmt.Sprintf("Challenge %d", i),
			"content":  "Benchmarking content...",
			"category": "web hacking",
		}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/v1/challenges", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		// Attach the authentication cookies
		for _, cookie := range cookies {
			req.AddCookie(cookie)
		}

		// (Optional) If your app requires X-CSRF-Token header:
		for _, cookie := range cookies {
			if cookie.Name == "csrfToken" {
				req.Header.Set("X-CSRF-Token", cookie.Value)
			}
		}

		// --- 3. Actual Measurement ---
		b.StartTimer() // Measure only the handler execution
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		b.StopTimer() // Stop before preparing the next iteration's unique name

		if rr.Code != http.StatusCreated {
			b.Fatalf("Request failed with status %d: %s", rr.Code, rr.Body.String())
		}
	}
}
