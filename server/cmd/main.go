package main

import (
	"context"
	"log"
	"os"
	"time"

	"music-room/internal/auth"
	"music-room/internal/handler"
	"music-room/internal/repository"
	"music-room/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

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

	// Registration repositories and services
	authRepo := repository.NewAuthRepository(pool)
	emailSvc := service.NewEmailService(
		getEnvOrDefault("SMTP_HOST", "mailpit"),
		getEnvOrDefault("SMTP_PORT", "1025"),
		getEnvOrDefault("SMTP_FROM", "noreply@musicroom.local"),
		os.Getenv("SMTP_USER"),
		os.Getenv("SMTP_PASSWORD"),
	)
	authSvc := service.NewAuthService(authRepo, emailSvc, getEnvOrDefault("APP_URL", "http://localhost:8081"))
	authHandler := handler.NewAuthHandler(authSvc)

	// JWT repositories and services
	userRepo := repository.NewPostgresUserRepository(pool)
	tokenRepo := repository.NewPostgresRefreshTokenRepository(pool)
	jwtService := auth.NewJWTService()
	jwtHandler := auth.NewHandler(userRepo, tokenRepo, jwtService)

	profileRepo := repository.NewProfileRepository(pool)
	profileSvc := service.NewProfileService(profileRepo)
	profileHandler := handler.NewProfileHandler(profileSvc)

	r := setupRouter(authHandler, jwtHandler, jwtService, profileHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func setupRouter(authHandler *handler.AuthHandler, jwtHandler *auth.Handler, jwtService *auth.JWTService, profileHandler *handler.ProfileHandler) *gin.Engine {
	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "UP"})
	})

	v1 := r.Group("/api/v1")
	{
		// Registration and email verification
		auth := v1.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.GET("/verify-email", authHandler.VerifyEmail)
			auth.POST("/resend-verification", authHandler.ResendVerification)
			auth.POST("/forgot-password", authHandler.ForgotPassword)
			auth.POST("/reset-password", authHandler.ResetPassword)
			// JWT login/refresh/logout
			auth.POST("/login", jwtHandler.Login)
			auth.POST("/refresh", jwtHandler.Refresh)
			auth.POST("/logout", jwtHandler.Logout)
		}

		// Profile endpoints - mock auth until JWT middleware is wired in
		users := v1.Group("/users")
		users.Use(func(c *gin.Context) {
			uid := c.GetHeader("X-User-ID")
			if uid == "" {
				c.AbortWithStatusJSON(401, gin.H{"error": "Unauthorized"})
				return
			}
			c.Set("authenticated_user_id", uid)
			c.Next()
		})
		{
			users.GET("/me", profileHandler.GetMyProfile)
			users.PATCH("/me", profileHandler.UpdateMyProfile)
			users.GET("/:id", profileHandler.GetUserProfile)
		}
	}

	return r
}

func getEnvOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
