package main

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"time"

	"github.com/RichardHoa/hack-me/internal/app"

	"github.com/RichardHoa/hack-me/internal/constants"
	"github.com/RichardHoa/hack-me/internal/routes"
)

// func init() {
// 	// Optional: enable block & mutex profiling
// 	runtime.SetBlockProfileRate(1)     // capture all blocking events
// 	runtime.SetMutexProfileFraction(1) // capture all mutex contention
// }

func main() {

	application, err := app.NewApplication(false)
	if err != nil {
		panic(err)
	}
	defer application.ConnectionPool.Close()

	router := routes.SetUpRoutes(application)

	server := http.Server{
		Addr:              fmt.Sprintf(":%d", constants.AppPort),
		Handler:           router,
		IdleTimeout:       time.Minute,
		ReadHeaderTimeout: 30 * time.Second,
		WriteTimeout:      30 * time.Second,
	}

	// go func() {
	// 	fmt.Println(http.ListenAndServe("localhost:6060", nil))
	// }()

	application.Logger.Printf("Server is running on port: %d", constants.AppPort)

	err = server.ListenAndServe()
	if err != nil {
		panic(err)
	}
}
