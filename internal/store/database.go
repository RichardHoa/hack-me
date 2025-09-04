package store

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"os"

	"github.com/RichardHoa/hack-me/internal/constants"
	"github.com/pressly/goose/v3"

	// _ "github.com/jackc/pgx/v4/stdlib"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
)

/*
Open establishes and verifies a connection to the main application database.
*/
func Open() (*sql.DB, *pgxpool.Pool, error) {
	var connStr string

	if constants.IsDevMode {
		fmt.Println("DEV MODE")
		host := "localhost"
		user := "postgres"
		password := "postgres"
		port := "5432"
		dbname := "postgres"

		connStr = fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
			host, user, password, dbname, port)

	} else {
		fmt.Println("PRODUCTION MODE")
		password := os.Getenv("DB_PASSWORD")
		if password == "" {
			return nil, nil, fmt.Errorf("DB_PASSWORD environment variable not set")
		}

		host := os.Getenv("DB_HOST")
		user := os.Getenv("DB_USER")
		port := "5432"
		dbname := "postgres"

		connStr = fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=require",
			host, user, password, dbname, port)

	}

	dbPool, err := pgxpool.New(context.Background(), connStr)
	if err != nil {
		return nil, nil, fmt.Errorf("Error opening database connection: %w", err)
	}

	db := stdlib.OpenDBFromPool(dbPool)

	err = db.Ping()
	if err != nil {
		return nil, nil, fmt.Errorf("Error opening database connection: %w", err)
	}
	fmt.Println("Establish database connection")
	return db, dbPool, err
}

/*
OpenTesting establishes and verifies a connection to a dedicated testing database.
*/
func OpenTesting() (*sql.DB, *pgxpool.Pool, error) {
	connStr := "host=localhost user=postgres password=postgres dbname=postgres port=5433 sslmode=disable"

	dbPool, err := pgxpool.New(context.Background(), connStr)
	if err != nil {
		return nil, nil, fmt.Errorf("Error opening database connection: %w", err)
	}

	db := stdlib.OpenDBFromPool(dbPool)

	err = db.Ping()
	if err != nil {
		return nil, nil, fmt.Errorf("Error opening database connection: %w", err)
	}

	fmt.Println(
		"Connected to testing database")
	return db, dbPool, err

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
