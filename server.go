package main

import (
	"encoding/json"
	"log"
	"net/http"
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
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

}

func ServerStart() {
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
	mux.HandleFunc("GET /boom", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	err := http.ListenAndServe(port, mux)
	if err != nil {
		log.Println("Ошибка запуска сервера: ", err)
		return
	}
}
