package app

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/RichardHoa/hack-me/internal/api"
	"github.com/RichardHoa/hack-me/internal/constants"
	"github.com/RichardHoa/hack-me/internal/middleware"
	"github.com/RichardHoa/hack-me/internal/store"
	"github.com/RichardHoa/hack-me/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/qdrant/go-client/qdrant"
	"google.golang.org/genai"
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
	ChatboxHandler               *api.ChatboxHandler
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
		db, connPool, err = store.OpenTesting()
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

	AIClient, err := InitAI(context.Background())
	if err != nil {
		panic(err)
	}

	QdrantClient, err := InitQdrant(context.Background())
	if err != nil {
		panic(err)
	}

	err = EnsureCollectionExist(context.Background(), QdrantClient)
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
	chatboxHandler := api.NewChatboxHandler(logger, AIClient, QdrantClient)

	//NOTE: Middleware creation
	middleware := middleware.NewMiddleWare(logger)

	application := &Application{
		Logger:                       logger,
		InfoLogger:                   infoLogger,
		DB:                           db,
		ConnectionPool:               connPool,
		ChallengeHandler:             challengeHandler,
		ChallengeResponseHandler:     challengeResponseHandler,
		ChallengeresponseVoteHandler: challengeResponseVoteHandler,
		CommentHandler:               commentHandler,
		ChatboxHandler:               chatboxHandler,
		UserHandler:                  userHandler,
		Middleware:                   middleware,
	}

	logger.Println("Start clean up jobs")
	application.StartTokenCleanupJob()

	return application, nil
}

func (a *Application) StartTokenCleanupJob() {
	ticker := time.NewTicker(24 * time.Hour)

	// Launch a goroutine that runs forever.
	go func() {
		for {
			// This will block until the next tick from the ticker.
			<-ticker.C

			a.Logger.Println("Running scheduled job: cleaning up expired refresh tokens...")

			rowsDeleted, err := a.UserHandler.TokenStore.DeleteExpiredTokens()
			if err != nil {
				a.Logger.Printf("ERROR: failed to clean up expired tokens: %v", err)
			} else {
				a.Logger.Printf("Background job finished. Deleted %d stale tokens.", rowsDeleted)
			}
		}
	}()
}

func InitAI(ctx context.Context) (*genai.Client, error) {
	key := constants.AISecretKey

	c, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  key,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return &genai.Client{}, err
	}
	return c, nil
}

func InitQdrant(ctx context.Context) (*qdrant.Client, error) {
	intVectorPort, err := strconv.Atoi(constants.VectorPort)
	if err != nil {
		return &qdrant.Client{}, fmt.Errorf("Qdrant port is not valid integer %d", constants.VectorPort)
	}

	client, err := qdrant.NewClient(&qdrant.Config{
		Host:   constants.VectorHost,
		Port:   intVectorPort,
		APIKey: constants.VectorSecret,
		UseTLS: true,
	})
	if err != nil {
		return &qdrant.Client{}, fmt.Errorf("qdrant: %w", err)
	}
	return client, nil
}

func EnsureCollectionExist(ctx context.Context, cli *qdrant.Client) error {
	// First try to get it
	_, err := cli.GetCollectionInfo(ctx, constants.VectorCollectionName)
	if err == nil {
		// Already exists
		return nil
	}

	// If it's not found, create it
	err = cli.CreateCollection(ctx, &qdrant.CreateCollection{
		CollectionName: constants.VectorCollectionName,
		VectorsConfig: &qdrant.VectorsConfig{
			Config: &qdrant.VectorsConfig_Params{
				Params: &qdrant.VectorParams{
					Size:     uint64(constants.VectorDim),
					Distance: qdrant.Distance_Cosine,
				},
			},
		},
	})
	return err
}
