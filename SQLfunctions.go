package main

import (
	"database/sql"

	"golang.org/x/crypto/bcrypt"
)

func GetUserByUsername(db *sql.DB, login string, password string) (*User2, error) {
	var user User2
	err := db.QueryRow("SELECT id,username,password_hash FROM users WHERE username = $1",
		login).Scan(&user.ID, &user.Username, &user.PasswordHash)
	if err != nil {
		return &user, err
	}
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	return &user, err

}
