package app

import (
	"database/sql"
	"log"
	"os"

	"github.com/RichardHoa/hack-me/internal/api"
	"github.com/RichardHoa/hack-me/internal/constants"
	"github.com/RichardHoa/hack-me/internal/middleware"
	"github.com/RichardHoa/hack-me/internal/store"
	"github.com/RichardHoa/hack-me/migrations"
)

type Application struct {
	Logger                       *log.Logger
	DB                           *sql.DB
	ChallengeHandler             *api.ChallengeHandler
	UserHandler                  *api.UserHandler
	ChallengeResponseHandler     *api.ChallengeResponseHandler
	ChallengeresponseVoteHandler *api.ChallengeResponseVoteHandler
	Middleware                   middleware.MiddleWare
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

	//NOTE: store creation
	challengeStore := store.NewChallengeStore(db)
	userStore := store.NewUserStore(db)
	tokenStore := store.NewTokenStore(db)
	challengeResponseStore := store.NewChallengeResponseStore(db)
	challengeResponseVoteStore := store.NewVoteStore(db)

	//NOTE: Handler creation
	challengeHandler := api.NewChallengeHandler(&challengeStore, logger)
	userHandler := api.NewUserHandler(&userStore, &tokenStore, logger)
	challengeResponseHandler := api.NewChallengeResponseHandler(&challengeResponseStore, logger)
	challengeResponseVoteHandler := api.NewChallengeResponseVoteHandler(&challengeResponseVoteStore, logger)

	//NOTE: Middleware creation
	middleware := middleware.NewMiddleWare(logger)

	application := &Application{
		Logger:                       logger,
		DB:                           db,
		ChallengeHandler:             challengeHandler,
		ChallengeResponseHandler:     challengeResponseHandler,
		ChallengeresponseVoteHandler: challengeResponseVoteHandler,
		UserHandler:                  userHandler,
		Middleware:                   middleware,
	}

	return application, nil
}
