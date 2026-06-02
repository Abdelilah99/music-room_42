package main

import (
	"context"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	_ = godotenv.Load()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://postgres:postgres@localhost:5432/musicroom?sslmode=disable"
	}

	log.Printf("Connecting to database for seeding...")
	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(context.Background()); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	email := "test@example.com"
	password := "password123"

	var exists bool
	err = pool.QueryRow(context.Background(), "SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)", email).Scan(&exists)
	if err != nil {
		log.Fatalf("Failed to check if seed user exists: %v", err)
	}

	if exists {
		log.Printf("Seed user %s already exists. Skipping.", email)
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("Failed to hash password: %v", err)
	}

	_, err = pool.Exec(context.Background(), "INSERT INTO users (email, password_hash) VALUES ($1, $2)", email, string(hashedPassword))
	if err != nil {
		log.Fatalf("Failed to insert seed user: %v", err)
	}

	log.Printf("Seed user %s created successfully with password: %s", email, password)
}
