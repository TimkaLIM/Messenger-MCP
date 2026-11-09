package main

import (
	"context"
	"database/sql"
)

type MockDB struct {
	ReturnError error
}

func (m *MockDB) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	//если мы положили ошибку в поле, возвращаем её
	if m.ReturnError != nil {
		return nil, m.ReturnError
	} //иначе все окей
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
