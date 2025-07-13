package routes

import (
	"fmt"
	"net/http"

	"github.com/RichardHoa/hack-me/internal/app"
	"github.com/RichardHoa/hack-me/internal/constants"
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

	router.Route("/v1", func(outerRouter chi.Router) {

		if constants.IsDevMode {
			outerRouter.Use(app.Middleware.CorsMiddleware)
		}

		outerRouter.Route("/challenges", func(r chi.Router) {
			// GET /challenges?popularity=asc|desc&category=cat1&category=cat2&name=searchTerm
			r.Get("/", app.ChallengeHandler.GetChallenges)

			r.Group(func(csrfRouter chi.Router) {
				csrfRouter.Use(app.Middleware.RequireCSRFToken)
				csrfRouter.Post("/", app.ChallengeHandler.PostChallenge)
				csrfRouter.Put("/", app.ChallengeHandler.ModifyChallenge)
				csrfRouter.Delete("/", app.ChallengeHandler.DeleteChallege)
			})

			r.Route("/responses", func(innerRouter chi.Router) {
				innerRouter.Get("/", app.ChallengeResponseHandler.GetChallengeResponse)

				innerRouter.Group(func(csrfRouter chi.Router) {
					csrfRouter.Use(app.Middleware.RequireCSRFToken)
					csrfRouter.Post("/", app.ChallengeResponseHandler.PostChallengeResponse)
					csrfRouter.Put("/", app.ChallengeResponseHandler.ModifyChallengeResponse)
					csrfRouter.Delete("/", app.ChallengeResponseHandler.DeleteChallengeResponse)
				})

				innerRouter.Route("/votes", func(router chi.Router) {
					router.Use(app.Middleware.RequireCSRFToken)
					router.Post("/", app.ChallengeresponseVoteHandler.PostVote)
					router.Delete("/", app.ChallengeresponseVoteHandler.DeleteVote)

				})

			})

		})

		outerRouter.Route("/auth", func(r chi.Router) {
			r.Post("/tokens", app.UserHandler.RefreshTokenRotation)
		})

		outerRouter.Route("/users", func(r chi.Router) {
			r.Post("/", app.UserHandler.RegisterNewUser)
			r.Post("/login", app.UserHandler.LoginUser)
			r.Post("/logout", app.UserHandler.LogoutUser)
		})

		outerRouter.Route("/comments", func(r chi.Router) {
			r.Use(app.Middleware.RequireCSRFToken)
			r.Put("/", app.CommentHandler.ModifyComment)
			r.Post("/", app.CommentHandler.PostComment)
			r.Delete("/", app.CommentHandler.DeleteComment)

		})

	})

	return router

}
