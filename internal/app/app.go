package app

import (
	"database/sql"
	"log"
	"os"

	"github.com/RichardHoa/hack-me/internal/api"
	"github.com/RichardHoa/hack-me/internal/store"
	"github.com/RichardHoa/hack-me/migrations"
)

type Application struct {
	Logger           *log.Logger
	DB               *sql.DB
	ChallengeHandler *api.ChallengeHandler
}

func NewApplication() (*Application, error) {
	logger := log.New(os.Stdout, "", log.Ldate|log.Ltime)

	db, err := store.Open()
	if err != nil {
		panic(err)
	}

	err = store.MigrateFS(db, migrations.FS, ".")
	if err != nil {
		panic(err)
	}

	challengeStore := store.NewChallengeStore(db)

	challengeHandler := api.NewChallengeHandler(&challengeStore, logger)

	application := &Application{
		Logger:           logger,
		DB:               db,
		ChallengeHandler: challengeHandler,
	}

	return application, nil
}
