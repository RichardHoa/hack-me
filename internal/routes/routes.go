package routes

import (
	"fmt"
	"net/http"

	"github.com/RichardHoa/hack-me/internal/app"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func SetUpRoutes(app *app.Application) *chi.Mux {
	router := chi.NewRouter()

	// add ID to each request
	router.Use(middleware.RequestID)
	// Extracts the real client IP address from headers like X-Forwarded-For or X-Real-IP
	router.Use(middleware.RealIP)
	// Log path
	router.Use(middleware.Logger)
	// Send 500 error if server panic, output stack trace
	router.Use(middleware.Recoverer)

	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Service is available\n")
	})

	router.Route("/challenges", func(r chi.Router) {
		// GET /challenges?popularity=asc|desc&category=cat1&category=cat2&name=searchTerm
		r.Get("/", app.ChallengeHandler.GetChallenges)

		r.Post("/", app.ChallengeHandler.PostChallenge)

	})

	router.Route("/users", func(r chi.Router) {
		r.Post("/", app.UserHandler.RegisterNewUser)
		r.Post("/login", app.UserHandler.LoginUser)
	})

	return router

}
