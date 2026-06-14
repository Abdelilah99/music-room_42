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

	// Event repositories and services
	eventRepo := repository.NewEventRepository(pool)
	eventSvc := service.NewEventService(eventRepo)
	eventHandler := handler.NewEventHandler(eventSvc, hubManager)

	// Track repositories and services
	trackRepo := repository.NewTrackRepository(pool)
	trackSvc := service.NewTrackService(eventRepo, trackRepo)
	trackHandler := handler.NewTrackHandler(trackSvc, hubManager)

	// Device repositories and services
	deviceRepo := repository.NewDeviceRepository(pool)
	deviceSvc := service.NewDeviceService(deviceRepo)
	deviceHandler := handler.NewDeviceHandler(deviceSvc)

	// Delegation repositories and services
	delegRepo := repository.NewDelegationRepository(pool)
	delegSvc := service.NewDelegationService(deviceRepo, delegRepo)
	delegHandler := handler.NewDelegationHandler(delegSvc)

	// Command handler (playback relay)
	commandHandler := handler.NewCommandHandler(deviceSvc, hubManager)

	// Playlist repositories and services
	playlistRepo := repository.NewPlaylistRepository(pool)
	playlistSvc := service.NewPlaylistService(playlistRepo)
	playlistHandler := handler.NewPlaylistHandler(playlistSvc, hubManager)

	allowedOrigins := os.Getenv("ALLOWED_ORIGINS")
	globalLimit := getEnvOrDefault("RATE_LIMIT_GLOBAL", "100-M")
	authLimit := getEnvOrDefault("RATE_LIMIT_AUTH", "10-M")

	r := setupRouter(authHandler, jwtHandler, jwtService, profileHandler, friendHandler, musicHandler, googleHandler, eventHandler, trackHandler, deviceHandler, delegHandler, commandHandler, playlistHandler, allowedOrigins, globalLimit, authLimit)

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
	eventHandler *handler.EventHandler,
	trackHandler *handler.TrackHandler,
	deviceHandler *handler.DeviceHandler,
	delegHandler *handler.DelegationHandler,
	commandHandler *handler.CommandHandler,
	playlistHandler *handler.PlaylistHandler,
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
			music.GET("/tracks/:id", musicHandler.GetTrack)
		}

		devices := v1.Group("/devices")
		devices.Use(jwtMiddleware.Authenticate())
		{
			devices.POST("", deviceHandler.Register)
			devices.GET("", deviceHandler.List)
			devices.GET("/delegated", delegHandler.ListDelegated)
			devices.GET("/:id", deviceHandler.Get)
			devices.DELETE("/:id", deviceHandler.Delete)
			devices.POST("/:id/delegate", delegHandler.Grant)
			devices.DELETE("/:id/delegate", delegHandler.Revoke)
			devices.POST("/:id/command", delegHandler.RequireDelegateOrOwner(), commandHandler.Send)
		}

		deviceWs := v1.Group("/devices")
		deviceWs.Use(jwtMiddleware.AuthenticateWS())
		{
			// Owner or active delegate may subscribe, so a delegate sees the
			// owner's playback state (and vice versa) live.
			deviceWs.GET("/:id/ws", delegHandler.RequireDelegateOrOwner(), commandHandler.ServeDeviceWS)
		}

		events := v1.Group("/events")
		events.Use(jwtMiddleware.Authenticate())
		{
			events.POST("", eventHandler.Create)
			events.GET("", eventHandler.List)
			events.GET("/:id", eventHandler.Get)
			events.PUT("/:id", eventHandler.Update)
			events.DELETE("/:id", eventHandler.Delete)
			events.POST("/:id/invites", eventHandler.Invite)
			events.POST("/:id/tracks", trackHandler.Suggest)
			events.GET("/:id/queue", trackHandler.GetQueue)
			events.POST("/:id/tracks/:trackId/vote", trackHandler.Vote)
			events.DELETE("/:id/tracks/:trackId", trackHandler.DeleteTrack)
		}

		eventWs := v1.Group("/events")
		eventWs.Use(jwtMiddleware.AuthenticateWS())
		{
			eventWs.GET("/:id/ws", eventHandler.ServeWS)
		}

		playlists := v1.Group("/playlists")
		playlists.Use(jwtMiddleware.Authenticate())
		{
			playlists.POST("", playlistHandler.Create)
			playlists.GET("", playlistHandler.List)
			playlists.GET("/:id", playlistHandler.Get)
			playlists.PUT("/:id", playlistHandler.Update)
			playlists.DELETE("/:id", playlistHandler.Delete)
			playlists.POST("/:id/invites", playlistHandler.Invite)
			playlists.POST("/:id/tracks", playlistHandler.AddTrack)
			playlists.DELETE("/:id/tracks/:trackId", playlistHandler.RemoveTrack)
			playlists.PATCH("/:id/tracks/:trackId/position", playlistHandler.MoveTrack)
		}

		playlistWs := v1.Group("/playlists")
		playlistWs.Use(jwtMiddleware.AuthenticateWS())
		{
			playlistWs.GET("/:id/ws", playlistHandler.ServeWS)
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
