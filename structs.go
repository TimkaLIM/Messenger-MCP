package main

import "net/http"

type ResponseWriter struct {
	http.ResponseWriter
	statusCode int
}
