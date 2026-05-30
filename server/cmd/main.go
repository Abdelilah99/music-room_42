package main

import (
	"context"
	"log"
	"os"
	"time"

	"music-room/internal/handler"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)


func DiagnosticMockAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		uid := c.GetHeader("X-User-ID")
		if uid == "" {
			c.AbortWithStatusJSON(412, gin.H{"error": "Precondition Failed: Diagnostic testing requires an X-User-ID value"})
			return
		}
		c.Set("authenticated_user_id", uid)
		c.Next()
	}
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, reading from environment")
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	log.Println("Connecting to database...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	config, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		log.Fatalf("Failed to parse database URL: %v", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		log.Fatalf("Failed to create database pool: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Println("Database connection established")

	r := setupRouter(pool)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func setupRouter(pool *pgxpool.Pool) *gin.Engine {
	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "UP"})
	})

	profileHandler := handler.NewProfileHandler(pool)

	api := r.Group("/api/v1")
	api.Use(DiagnosticMockAuth())
	{
		api.GET("/users/me", profileHandler.GetMyProfile)
		api.PATCH("/users/me", profileHandler.UpdateMyProfile)
		api.GET("/users/:id", profileHandler.GetUserProfile)
	}

	return r
}
