package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	_ = godotenv.Load()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://postgres:postgres@localhost:5432/musicroom?sslmode=disable"
	}

	slog.Info("connecting to database for seeding")
	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(context.Background()); err != nil {
		slog.Error("failed to ping database", "error", err)
		os.Exit(1)
	}

	email := "test@example.com"
	password := "password123"

	var exists bool
	err = pool.QueryRow(context.Background(), "SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)", email).Scan(&exists)
	if err != nil {
		slog.Error("failed to check if seed user exists", "error", err)
		os.Exit(1)
	}

	if exists {
		slog.Info("seed user already exists, skipping", "email", email)
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		slog.Error("failed to hash password", "error", err)
		os.Exit(1)
	}

	_, err = pool.Exec(context.Background(), "INSERT INTO users (email, password_hash) VALUES ($1, $2)", email, string(hashedPassword))
	if err != nil {
		slog.Error("failed to insert seed user", "error", err)
		os.Exit(1)
	}

	slog.Info("seed user created successfully", "email", email)
}
