package DataBase

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"log"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func RunMigrations(db *sql.DB) error {
	//создаем источник из нашей embed файловой системы (откуда читать миграции)
	source, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("ошибка чтения embed миграций: %w", err)
	}
	defer func() {
		if err = source.Close(); err != nil {
			log.Println("ошибка закрытия источника миграций: ", err)
		}
	}()

	//оборачиваем пул в драйвер БД (куда накидывать миграции)
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("ошибка драйвера миграций: %w", err)
	}

	//создаем наш мигратор, передаем наш source(источник, откуда брать миграции), и driver(БД)
	m, err := migrate.NewWithInstance("iofs", source, "postgres", driver)
	if err != nil {
		return fmt.Errorf("ошибка создания мигратора: %w", err)
	}
	if err = m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("ошибка применения миграций: %w", err)
	}
	return nil

}
