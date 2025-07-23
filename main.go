package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/RichardHoa/hack-me/internal/app"
	"github.com/RichardHoa/hack-me/internal/constants"
	"github.com/RichardHoa/hack-me/internal/routes"
)

func main() {

	application, err := app.NewApplication(false)
	if err != nil {
		panic(err)
	}
	defer application.DB.Close()

	router := routes.SetUpRoutes(application)

	server := http.Server{
		Addr:         fmt.Sprintf(":%d", constants.AppPort),
		Handler:      router,
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Minute,
		WriteTimeout: 30 * time.Minute,
	}

	application.Logger.Printf("Server is running on port: %d", constants.AppPort)

	err = server.ListenAndServe()
	if err != nil {
		panic(err)
	}
}
