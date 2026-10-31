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
	DB *sql.DB
}
