package main

import (
	"crypto/rand"
	"database/sql"
	"math/big"

	"golang.org/x/crypto/bcrypt"
)

const letters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func GenerateInviteCode(length int) (string, error) {
	result := make([]byte, length)
	for i := range result {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		if err != nil {
			return "", err
		}
		result[i] = letters[num.Int64()]
	}
	return string(result), nil
}

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
