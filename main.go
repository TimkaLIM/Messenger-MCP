package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
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
	defer db.Close()

	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	if err2 := db.Ping(); err2 != nil {
		log.Fatal("База данных недоступна: ", err2)
	}
	fmt.Println("База данных запущена...")

	RunMigrations(db) //Вызов миграций

	ServerStart(db)
}
