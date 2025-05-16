package repositories

import (
	"database/sql"
	"fmt"
	"log"
	"time"
)

type Repository struct {
	db *sql.DB
}

type UserData struct {
	UserName     string    `json:"user_name"`
	ChatID       string    `json:"chat_id"`
	ExpiresIn    time.Time `json:"expires_in"`
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
}

func InitDB() (*Repository, error) {
	connStr := ""
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("error connecting to database: %w", err)
	}

	defer db.Close()
	err = db.Ping()
	if err != nil {
		return nil, fmt.Errorf("error pinging database: %w", err)
	}

	log.Println("Successfully connected to database")
	return &Repository{db: db}, nil
}

func (r *Repository) CreateUser(userName string, chatID string, expiresIn time.Time, accessToken string) error {
	query := `INSERT INTO users (chat_id, user_name, access_token, expires_in, access_token, refresh_token)
              VALUES ($1, $2, $3, $4, $5)`

	_, err := r.db.Exec(query, chatID, userName, accessToken, expiresIn, accessToken)
	if err != nil {
		return fmt.Errorf("error creating user: %w", err)
	}

	log.Printf("Successfully created user %s", userName)
	return nil
}

func (r *Repository) GetUser(userName string) (UserData, error) {
	query := `SELECT * FROM users WHERE user_name=$1`

	userData := UserData{}
	err := r.db.QueryRow(query, userName).
		Scan(&userData.UserName, &userData.ChatID, &userData.ExpiresIn, &userData.AccessToken)

	if err != nil {
		return UserData{}, fmt.Errorf("error getting user: %w", err)
	}

	log.Printf("Successfully got user %s", userName)
	return userData, nil
}

func (r *Repository) UpdateUserTokens(userName, refreshToken, accessToken string, expiresIn time.Time) error {
	query := `UPDATE users SET access_token=$1, refresh_token=$2, expires_in=$3 WHERE user_name=$4`
	_, err := r.db.Exec(query, accessToken, refreshToken, expiresIn, userName)
	if err != nil {
		return fmt.Errorf("error updating user: %w", err)
	}

	log.Printf("Successfully updated user %s", userName)
	return nil
}
