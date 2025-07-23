package store

import (
	"database/sql"
	"fmt"
	"io/fs"
	"os"

	"github.com/RichardHoa/hack-me/internal/constants"
	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/pressly/goose/v3"
)

/*
Open establishes and verifies a connection to the main application database.
It constructs the connection string from environment variables. In development
mode, it defaults the host to "localhost". It returns the active database
connection pool or an error if the connection fails.
*/
func Open() (*sql.DB, error) {
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")

	if constants.IsDevMode {
		host = "localhost"
	}

	connStr := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		host, user, password, dbname, port)

	db, err := sql.Open("pgx", connStr)
	if err != nil {
		return nil, fmt.Errorf("Error opening database connection: %w", err)
	}
	err = db.Ping()
	if err != nil {
		return nil, fmt.Errorf("Error opening database connection: %w", err)
	}
	fmt.Println("Connected to database")
	return db, err
}

/*
OpenTesting establishes and verifies a connection to a dedicated testing database.
It uses a hardcoded connection string suitable for a local test environment.
It returns the active database connection pool or an error if the connection fails.
*/
func OpenTesting() (*sql.DB, error) {

	db, err := sql.Open("pgx", "host=localhost user=postgres password=postgres dbname=postgres port=5433 sslmode=disable")

	if err != nil {
		return nil, fmt.Errorf("Error opening database connection: %w", err)
	}

	err = db.Ping()
	if err != nil {
		return nil, fmt.Errorf("Error opening database connection: %w", err)
	}

	fmt.Println(
		"Connected to testing database")
	return db, err

}

/*
MigrateFS runs database migrations from an embedded filesystem (fs.FS).
*/
func MigrateFS(db *sql.DB, migrationFS fs.FS, dir string) error {
	// set the base FS to migrations folder FS
	goose.SetBaseFS(migrationFS)
	defer func() {
		// reset to avoid potential side-effect
		goose.SetBaseFS(nil)
	}()

	return Migrate(db, dir)
}

/*
Migrate runs all pending "up" database migrations found in the specified directory.
It configures the goose migration tool for a PostgreSQL database.
*/
func Migrate(db *sql.DB, dir string) error {

	err := goose.SetDialect("postgres")

	if err != nil {
		return fmt.Errorf("Migration error: %w", err)
	}

	err = goose.Up(db, dir)

	if err != nil {
		return fmt.Errorf("Goose UP error: %w", err)
	}

	return nil
}
