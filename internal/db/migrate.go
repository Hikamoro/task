package db

import (
	"database/sql"
	"errors"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/mysql"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	"task/migrations"
)

// MigrateUp applies all pending migrations to the given database.
func MigrateUp(db *sql.DB) error {
	src, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return err
	}
	driver, err := mysql.WithInstance(db, &mysql.Config{})
	if err != nil {
		return err
	}
	m, err := migrate.NewWithInstance("iofs", src, "mysql", driver)
	if err != nil {
		return err
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}