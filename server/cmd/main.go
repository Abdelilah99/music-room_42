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

	// Profile repositories and services
	profileRepo := repository.NewProfileRepository(pool)
	profileSvc := service.NewProfileService(profileRepo)
	profileHandler := handler.NewProfileHandler(profileSvc)

	// Friend repositories and services
	friendRepo := repository.NewFriendRepository(pool)
	friendSvc := service.NewFriendService(friendRepo)
	friendHandler := handler.NewFriendHandler(friendSvc)

	r := setupRouter(authHandler, jwtHandler, jwtService, profileHandler, friendHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func setupRouter(
	authHandler *handler.AuthHandler,
	jwtHandler *auth.Handler,
	jwtService *auth.JWTService,
	profileHandler *handler.ProfileHandler,
	friendHandler *handler.FriendHandler,
) *gin.Engine {
	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "UP"})
	})

	jwtMiddleware := auth.NewMiddleware(jwtService)

	v1 := r.Group("/api/v1")
	{
		// Registration and email verification (public)
		authGroup := v1.Group("/auth")
		{
			authGroup.POST("/register", authHandler.Register)
			authGroup.GET("/verify-email", authHandler.VerifyEmail)
			authGroup.POST("/resend-verification", authHandler.ResendVerification)
			authGroup.POST("/forgot-password", authHandler.ForgotPassword)
			authGroup.POST("/reset-password", authHandler.ResetPassword)
			authGroup.POST("/login", jwtHandler.Login)
			authGroup.POST("/refresh", jwtHandler.Refresh)
			authGroup.POST("/logout", jwtHandler.Logout)
		}

		// Profile endpoints (JWT protected)
		users := v1.Group("/users")
		users.Use(jwtMiddleware.Authenticate())
		{
			users.GET("/me", profileHandler.GetMyProfile)
			users.PATCH("/me", profileHandler.UpdateMyProfile)
			users.GET("/:id", profileHandler.GetUserProfile)
		}

		// Friend endpoints (JWT protected)
		friends := v1.Group("/friends")
		friends.Use(jwtMiddleware.Authenticate())
		{
			friends.POST("/request", friendHandler.SendRequest)
			friends.POST("/accept/:id", friendHandler.AcceptRequest)
			friends.DELETE("/reject/:id", friendHandler.RejectRequest)
			friends.DELETE("/:id", friendHandler.Unfriend)
			friends.GET("", friendHandler.ListFriends)
			friends.GET("/requests", friendHandler.ListRequests)
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
