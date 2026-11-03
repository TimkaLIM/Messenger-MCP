package main

import (
	"database/sql"

	"golang.org/x/crypto/bcrypt"
)

func GetUserByUsername(db *sql.DB, login string, password string) error {
	var hash_password string
	err := db.QueryRow("SELECT password_hash FROM users WHERE username = $1",
		login).Scan(&hash_password)
	if err != nil {
		return err
	}
	err = bcrypt.CompareHashAndPassword([]byte(hash_password), []byte(password))
	return err

}
