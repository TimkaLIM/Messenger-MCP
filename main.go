package main

import (
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"

	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://admin:1234@localhost:5446/train55?sslmode=disable"
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Println("Ошибка открытия пула: ", err)
		return
	}
	defer db.Close()

	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	if err2 := db.Ping(); err2 != nil {
		log.Fatal("База данных недоступна: ", err2)
	}
	fmt.Println("База данных запущена...")

	MigrateErr := RunMigrations(db) //Вызов миграций
	if MigrateErr != nil {
		log.Fatal("Ошибка наложения миграций: ", MigrateErr)
	}

	ServerStart(db)
}
