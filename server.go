package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

func echoHandler(w http.ResponseWriter, r *http.Request) {

	r.Body = http.MaxBytesReader(w, r.Body, 1048576)

	var book map[string]any

	if jsonErr := json.NewDecoder(r.Body).Decode(&book); jsonErr != nil {
		log.Println("Ошибка декодинга echo:", jsonErr)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if jsonErr2 := json.NewEncoder(w).Encode(book); jsonErr2 != nil {
		log.Println("Ошибка энкодинга echo: ", jsonErr2)
		return
	}
}

func (rw *ResponseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		//Этот код выполняется до того, как попадет Handler(любой)
		start := time.Now()
		method := r.Method
		path := r.URL.Path
		w2 := &ResponseWriter{ResponseWriter: w, statusCode: 200}

		//Передаем управление дальше в хэндлер
		next.ServeHTTP(w2, r)

		status := w2.statusCode

		//Этот код выполнится после того как хэндлер завершится
		duration := time.Since(start)
		fmt.Printf("[LOG] %s %s %d| Время: %v\n", method, path, status, duration)
	})
}

// Функция для обертки хэндлэра в логгер, а потом сразу отправить в mux
func registerRoute(mux *http.ServeMux, methodAndPath string,
	handler func(w http.ResponseWriter, r *http.Request)) {

	logger := loggingMiddleware(http.HandlerFunc(handler))
	mux.Handle(methodAndPath, logger)
}

func ServerStart() {
	port := ":8080"
	mux := http.NewServeMux()

	registerRoute(mux, "GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{"status":"ok"}`))
		if err != nil {
			log.Println("Ошибка: ", err)
		}
	})
	registerRoute(mux, "POST /echo", echoHandler)
	registerRoute(mux, "GET /boom", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	err := http.ListenAndServe(port, mux)
	if err != nil {
		log.Println("Ошибка запуска сервера: ", err)
		return
	}
}
