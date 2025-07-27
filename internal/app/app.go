package app

import (
	"database/sql"
	"log"
	"os"
	"time"

	"github.com/RichardHoa/hack-me/internal/api"
	"github.com/RichardHoa/hack-me/internal/constants"
	"github.com/RichardHoa/hack-me/internal/middleware"
	"github.com/RichardHoa/hack-me/internal/store"
	"github.com/RichardHoa/hack-me/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Application struct {
	Logger         *log.Logger
	InfoLogger     *log.Logger
	DB             *sql.DB
	ConnectionPool *pgxpool.Pool
	/*
		use pointer for handler to make sure the handler never get copies,
		thus all the handler data is always up-to-date and there is no local lag
	*/
	ChallengeHandler             *api.ChallengeHandler
	UserHandler                  *api.UserHandler
	ChallengeResponseHandler     *api.ChallengeResponseHandler
	ChallengeresponseVoteHandler *api.ChallengeResponseVoteHandler
	CommentHandler               *api.CommentHandler
	Middleware                   middleware.MiddleWare
}

func NewApplication(isTesting bool) (*Application, error) {
	logger := log.New(os.Stderr, "", log.Ldate|log.Ltime)
	infoLogger := log.New(os.Stdout, "", log.Ldate|log.Ltime)

	var (
		db       *sql.DB
		connPool *pgxpool.Pool
		err      error
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
		db, connPool, err = store.Open()
		if err != nil {
			panic(err)
		}
	}

	err = store.MigrateFS(db, migrations.FS, ".")
	if err != nil {
		panic(err)
	}

	/*
		all the stores are returned as pointers
		because we satisfy all interface by functions with pointer receiver
	*/
	//NOTE: store creation
	userStore := store.NewUserStore(db)
	tokenStore := store.NewTokenStore(db)
	challengeResponseVoteStore := store.NewVoteStore(db)
	commentStore := store.NewCommentStore(db)
	challengeResponseStore := store.NewChallengeResponseStore(db, commentStore)
	challengeStore := store.NewChallengeStore(db, commentStore)

	//NOTE: Handler creation
	challengeHandler := api.NewChallengeHandler(challengeStore, logger)
	userHandler := api.NewUserHandler(userStore, tokenStore, logger)
	challengeResponseHandler := api.NewChallengeResponseHandler(challengeResponseStore, logger)
	challengeResponseVoteHandler := api.NewChallengeResponseVoteHandler(challengeResponseVoteStore, logger)
	commentHandler := api.NewCommentHandler(commentStore, logger)

	//NOTE: Middleware creation
	middleware := middleware.NewMiddleWare(logger)

	//NOTE: Jobs
	logger.Println("Start clean up jobs")
	StartTokenCleanupJob(logger, tokenStore)

	application := &Application{
		Logger:                       logger,
		InfoLogger:                   infoLogger,
		DB:                           db,
		ConnectionPool:               connPool,
		ChallengeHandler:             challengeHandler,
		ChallengeResponseHandler:     challengeResponseHandler,
		ChallengeresponseVoteHandler: challengeResponseVoteHandler,
		CommentHandler:               commentHandler,
		UserHandler:                  userHandler,
		Middleware:                   middleware,
	}

	return application, nil
}

func StartTokenCleanupJob(logger *log.Logger, tokenStore store.TokenStore) {
	ticker := time.NewTicker(24 * time.Hour)

	// Launch a goroutine that runs forever.
	go func() {
		for {
			// This will block until the next tick from the ticker.
			<-ticker.C

			logger.Println("Running scheduled job: cleaning up expired refresh tokens...")

			rowsDeleted, err := tokenStore.DeleteExpiredTokens()
			if err != nil {
				logger.Printf("ERROR: failed to clean up expired tokens: %v", err)
			} else {
				logger.Printf("Background job finished. Deleted %d stale tokens.", rowsDeleted)
			}
		}
	}()
}
