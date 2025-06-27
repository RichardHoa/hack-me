package store

import (
	"database/sql"
	"fmt"
	"io/fs"

	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/pressly/goose/v3"
)

func Open() (*sql.DB, error) {

	db, err := sql.Open("pgx", "host=localhost user=postgres password=postgres dbname=postgres port=5432 sslmode=disable")

	if err != nil {
		return nil, fmt.Errorf("Error opening database connection: %w", err)
	}

	err = db.Ping()
	if err != nil {
		return nil, fmt.Errorf("Error opening database connection: %w", err)
	}

	fmt.Println(
		"Connected to database")
	return db, err

}

func MigrateFS(db *sql.DB, migrationFS fs.FS, dir string) error {
	// set the base FS to migrations folder FS
	goose.SetBaseFS(migrationFS)
	defer func() {
		// reset to avoid potential side-effect
		goose.SetBaseFS(nil)
	}()

	return Migrate(db, dir)
}

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
