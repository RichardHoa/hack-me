package app

import (
	"database/sql"
	"log"
	"os"

	"github.com/RichardHoa/hack-me/internal/api"
	"github.com/RichardHoa/hack-me/internal/constants"
	"github.com/RichardHoa/hack-me/internal/store"
	"github.com/RichardHoa/hack-me/migrations"
)

type Application struct {
	Logger           *log.Logger
	DB               *sql.DB
	ChallengeHandler *api.ChallengeHandler
	UserHandler      *api.UserHandler
}

func NewApplication(isTesting bool) (*Application, error) {
	logger := log.New(os.Stdout, "", log.Ldate|log.Ltime)

	var (
		db  *sql.DB
		err error
	)

	err = constants.LoadEnv()
	if err != nil {
		panic(err)
	}

	if isTesting {
		db, err = store.OpenTesting()
		if err != nil {
			panic(err)
		}
	} else {
		db, err = store.Open()
		if err != nil {
			panic(err)
		}
	}

	err = store.MigrateFS(db, migrations.FS, ".")
	if err != nil {
		panic(err)
	}

	challengeStore := store.NewChallengeStore(db)
	userStore := store.NewUserStore(db)
	tokenStore := store.NewTokenStore(db)

	challengeHandler := api.NewChallengeHandler(&challengeStore, logger)
	userHandler := api.NewUserHandler(&userStore, &tokenStore, logger)

	application := &Application{
		Logger:           logger,
		DB:               db,
		ChallengeHandler: challengeHandler,
		UserHandler:      userHandler,
	}

	return application, nil
}
