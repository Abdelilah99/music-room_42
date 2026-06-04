package main

import (
	"database/sql"
	"log/slog"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	_ = godotenv.Load()

	action := "up"
	if len(os.Args) > 1 {
		action = os.Args[1]
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://postgres:postgres@localhost:5432/musicroom?sslmode=disable"
	}

	slog.Info("connecting to database for migrations")
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		slog.Error("failed to open database connection", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		slog.Error("failed to ping database", "error", err)
		os.Exit(1)
	}

	slog.Info("initializing migrations", "source", "file://migrations")
	m, err := migrate.New("file://migrations", databaseURL)
	if err != nil {
		slog.Error("failed to initialize migrations system", "error", err)
		os.Exit(1)
	}
	defer m.Close()

	switch action {
	case "up":
		slog.Info("applying database migrations (UP)")
		if err := m.Up(); err != nil {
			if err == migrate.ErrNoChange {
				slog.Info("database is already up to date, no migrations to apply")
			} else {
				slog.Error("failed to apply migrations", "error", err)
				os.Exit(1)
			}
		} else {
			slog.Info("database migrations applied successfully")
		}
	case "down":
		slog.Info("reverting database migrations (DOWN)")
		if err := m.Down(); err != nil {
			if err == migrate.ErrNoChange {
				slog.Info("no migrations to revert")
			} else {
				slog.Error("failed to revert migrations", "error", err)
				os.Exit(1)
			}
		} else {
			slog.Info("database migrations reverted successfully")
		}
	default:
		slog.Error("invalid migration action", "action", action, "valid", []string{"up", "down"})
		os.Exit(1)
	}
}
