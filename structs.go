package main

import (
	"database/sql"
	"net/http"
)

type ResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

type Application struct {
	DB        *sql.DB
	SecretKey string
}

type User struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type User2 struct {
	ID           int    `json:"id"`
	Username     string `json:"username"`
	PasswordHash string `json:"-"` //- означает никогда не отдавать это поле в json клиенту
}

// chat structure

type CreateChatRequest struct {
	Title string `json:"title"`
}

type ChatResponse struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
}
