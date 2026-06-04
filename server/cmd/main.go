// @title        Music Room API
// @version      1.0
// @description  REST API for the Music Room collaborative listening application.
// @host         localhost:8081
// @BasePath     /api/v1
// @securityDefinitions.apikey BearerAuth
// @in           header
// @name         Authorization
// @description  Enter your Bearer token: "Bearer <token>"

package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	_ "music-room/docs"
	"music-room/internal/auth"
	"music-room/internal/handler"
	"music-room/internal/hub"
	"music-room/internal/middleware"
	"music-room/internal/repository"
	"music-room/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	if err := godotenv.Load(); err != nil {
		slog.Info("no .env file found, reading from environment")
	}

	middleware.RegisterJSONTagNames()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		slog.Error("DATABASE_URL environment variable is required")
		os.Exit(1)
	}

	slog.Info("connecting to database")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	config, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		slog.Error("failed to parse database URL", "error", err)
		os.Exit(1)
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		slog.Error("failed to create database pool", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		slog.Error("failed to ping database", "error", err)
		os.Exit(1)
	}
	slog.Info("database connection established")

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

	// Music search service and handler
	musicSvc := service.NewMusicService()
	musicHandler := handler.NewMusicHandler(musicSvc)

	// Google OAuth
	oauthRepo := repository.NewOAuthRepository(pool)
	googleHandler := auth.NewGoogleHandler(oauthRepo, userRepo, tokenRepo, jwtService)

	// WebSocket hub manager (shared across all real-time services)
	hubManager := hub.NewHubManager()

	allowedOrigins := os.Getenv("ALLOWED_ORIGINS")
	globalLimit := getEnvOrDefault("RATE_LIMIT_GLOBAL", "100-M")
	authLimit := getEnvOrDefault("RATE_LIMIT_AUTH", "10-M")

	r := setupRouter(authHandler, jwtHandler, jwtService, profileHandler, friendHandler, musicHandler, googleHandler, hubManager, allowedOrigins, globalLimit, authLimit)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	slog.Info("server starting", "port", port)
	if err := r.Run(":" + port); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func setupRouter(
	authHandler *handler.AuthHandler,
	jwtHandler *auth.Handler,
	jwtService *auth.JWTService,
	profileHandler *handler.ProfileHandler,
	friendHandler *handler.FriendHandler,
	musicHandler *handler.MusicHandler,
	googleHandler *auth.GoogleHandler,
	hubManager *hub.HubManager,
	allowedOrigins string,
	globalLimitRate string,
	authLimitRate string,
) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())

	r.Use(middleware.NewCORS(allowedOrigins))
	r.Use(middleware.NewLogger())
	r.Use(middleware.NewRateLimiter(globalLimitRate))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "UP"})
	})

	r.GET("/api/v1/docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	jwtMiddleware := auth.NewMiddleware(jwtService)

	v1 := r.Group("/api/v1")
	{
		authGroup := v1.Group("/auth")
		authGroup.Use(middleware.NewRateLimiter(authLimitRate))
		{
			authGroup.POST("/register", authHandler.Register)
			authGroup.GET("/verify-email", authHandler.VerifyEmail)
			authGroup.POST("/resend-verification", authHandler.ResendVerification)
			authGroup.POST("/forgot-password", authHandler.ForgotPassword)
			authGroup.POST("/reset-password", authHandler.ResetPassword)
			authGroup.POST("/login", jwtHandler.Login)
			authGroup.POST("/refresh", jwtHandler.Refresh)
			authGroup.POST("/logout", jwtHandler.Logout)
			authGroup.POST("/google", googleHandler.SignIn)
		}

		users := v1.Group("/users")
		users.Use(jwtMiddleware.Authenticate())
		{
			users.GET("/me", profileHandler.GetMyProfile)
			users.PATCH("/me", profileHandler.UpdateMyProfile)
			users.GET("/search", profileHandler.SearchUsers)
			users.GET("/:id", profileHandler.GetUserProfile)
		}

		link := v1.Group("/auth/link")
		link.Use(jwtMiddleware.Authenticate())
		{
			link.POST("/google", googleHandler.LinkGoogle)
		}

		friends := v1.Group("/friends")
		friends.Use(jwtMiddleware.Authenticate())
		{
			friends.POST("/request", friendHandler.SendRequest)
			friends.POST("/accept/:id", friendHandler.AcceptRequest)
			friends.DELETE("/reject/:id", friendHandler.RejectRequest)
			friends.DELETE("/:id", friendHandler.Unfriend)
			friends.GET("", friendHandler.ListFriends)
			friends.GET("/requests", friendHandler.ListRequests)
			friends.GET("/outgoing", friendHandler.ListOutgoing)
		}

		music := v1.Group("/music")
		music.Use(jwtMiddleware.Authenticate())
		{
			music.GET("/search", musicHandler.Search)
		}

		ws := v1.Group("/ws")
		ws.Use(jwtMiddleware.Authenticate())
		{
			ws.GET("/:entityID", func(c *gin.Context) {
				hub.ServeWS(hubManager, c.Param("entityID"), c)
			})
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
