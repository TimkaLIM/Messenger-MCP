package main

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type MockDB struct{}

func (m *MockDB) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return nil, nil
}
func (m *MockDB) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return nil, nil
}
func (m *MockDB) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return nil
}
func (m *MockDB) PingContext(ctx context.Context) error {
	return nil
}
func (m *MockDB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return nil, nil
}

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
