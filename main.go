package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/RichardHoa/hack-me/internal/app"
	"github.com/RichardHoa/hack-me/internal/routes"
)

func main() {
	const (
		PORT = 8080
	)

	application, err := app.NewApplication(false)
	if err != nil {
		panic(err)
	}
	defer application.DB.Close()

	router := routes.SetUpRoutes(application)

	application.Logger.Printf("Server is running on port: %d", PORT)

	server := http.Server{
		Addr:         fmt.Sprintf(":%d", PORT),
		Handler:      router,
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Minute,
		WriteTimeout: 30 * time.Minute,
	}

	err = server.ListenAndServe()
	if err != nil {
		panic(err)
	}
}
