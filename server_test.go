package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestHealthHandler(t *testing.T) {
	req := httptest.NewRequest("GET", "/health", nil) //аналог r *http.Request
	rr := httptest.NewRecorder()                      //аналог w http.ResponseWriter
	app := &Application{
		DB: &MockDB{},
	}
	app.healthHandler(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("Ожидали код 200, получили %d", rr.Code)
	}
	if rr.Body.String() != `{"status":"ok"}` {
		t.Errorf("Ожидали {\"status\":\"ok\"}, получили %s", rr.Body.String())
	}
}

func TestEchoHandler(t *testing.T) {
	message := `{"message":"hello world!"}`
	req := httptest.NewRequest("POST", "/echo", strings.NewReader(message))
	rr := httptest.NewRecorder()
	req.Header.Set("Content-Type", "application/json")
	echoHandler(rr, req)
	if rr.Code != 200 {
		t.Errorf("Ожидали код 200, получили %d", rr.Code)
	}
	if !(strings.Contains(rr.Body.String(), "hello world!")) {
		t.Errorf("В ответе нет ожидаемого сообщения: %s", rr.Body.String())
	}
}

func TestRegisterUser(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    string
		mockReturnErr  error
		expectedStatus int
	}{
		{
			name:           "Успешная регистрация",
			requestBody:    `{"username":"newuser","password":"123"}`,
			mockReturnErr:  nil,
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "Битый JSON",
			requestBody:    `{"username":"newuser2",password:1234`,
			mockReturnErr:  nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Пользователь уже существует",
			requestBody:    `{"username":"admin","password":"123"}`,
			mockReturnErr:  &pgconn.PgError{Code: "23505"}, //Имитируем ошибку дубликата
			expectedStatus: http.StatusConflict,            // 409
		},
		{
			name:           "Ошибка базы данных",
			requestBody:    `{"username":"newuser","password":"123"}`,
			mockReturnErr:  errors.New("connection refused"), //Имитируем падение базы
			expectedStatus: http.StatusInternalServerError,   //500

		},
		{
			name:           "Ошибка создания хэша пароля",
			requestBody:    `{"username":"newuser","password":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}`,
			mockReturnErr:  nil,
			expectedStatus: http.StatusInternalServerError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := &MockDB{ReturnError: tt.mockReturnErr}
			app := &Application{
				DB:        mockDB,
				SecretKey: "test",
			}
			req := httptest.NewRequest("POST", "/register", strings.NewReader(tt.requestBody))
			rr := httptest.NewRecorder()

			app.RegisterUser(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Ожидали код %d, получили %d", tt.expectedStatus, rr.Code)
			}
		})
	}
}

func TestLoginUser(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    string
		mockReturnErr  error
		expectedStatus int
	}{
		{
			name:           "Битый JSON",
			requestBody:    `{"username":"newuser2",password:1234`,
			mockReturnErr:  nil,
			expectedStatus: http.StatusBadRequest,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := &MockDB{ReturnError: tt.mockReturnErr}
			app := &Application{
				DB:        mockDB,
				SecretKey: "test",
			}
			req := httptest.NewRequest("POST", "/login",
				strings.NewReader(tt.requestBody))
			rr := httptest.NewRecorder()
			app.LoginUser(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Ожидали код: %d, получили: %d", tt.expectedStatus, rr.Code)
			}
		})
	}

}
