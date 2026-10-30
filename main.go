package main

import (
	"log"
	"net/http"
)

func echoHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

}

func main() {
	port := ":8080"
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{"status":"ok"}`))
		if err != nil {
			log.Println("Ошибка: ", err)
		}
	})
	mux.HandleFunc("POST /echo", echoHandler)

	err := http.ListenAndServe(port, mux)
	if err != nil {
		log.Println("Ошибка запуска сервера: ", err)
		return
	}
}
