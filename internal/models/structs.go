package main

import (
	"context"
	"database/sql"
	"net/http"
	"time"
)

type ResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

type Application struct {
	DB        DBInterface
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

type ReceivedMessage struct {
	Text string `json:"text"`
}

type JoinChat struct {
	InviteCode string `json:"invite_code"`
}

type MessagesResponse struct {
	ID        int       `json:"id"`
	SenderID  int       `json:"sender_id"`
	Text      string    `json:"text"`
	CreatedAT time.Time `json:"created_at"`
}

//интерфейс БД (для тестов)

type DBInterface interface {
	PingContext(context.Context) error
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}
