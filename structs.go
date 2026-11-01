package main

import (
	"database/sql"
	"net/http"
	"time"
)

type ResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

type Application struct {
	DB *sql.DB
}

type RegUser struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type User struct {
	ID           int       `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"` //- означает никогда не отадавать это поле в json клиенту
	CreatedAt    time.Time `json:"created_at"`
}
