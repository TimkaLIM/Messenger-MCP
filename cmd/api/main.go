package main

import (
	"MCPmessenger"
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"

	"log"
	"os"
	"time"

	"MCPmessenger/internal/api"
)

func main() {

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://admin:1234@localhost:5446/train55?sslmode=disable"
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Println("Ошибка открытия пула: ", err)
		return
	}
	defer func() {
		if err = db.Close(); err != nil {
			log.Println("Ошибка закрытия базы данных: ", err)
		}
	}()

	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	if err2 := db.Ping(); err2 != nil {
		log.Fatal("База данных недоступна: ", err2)
	}
	fmt.Println("База данных запущена...")

	MigrateErr := MCPmessenger.RunMigrations(db) //Вызов миграций
	if MigrateErr != nil {
		log.Fatal("Ошибка наложения миграций: ", MigrateErr)
	}

	api.ServerStart(db)
}
