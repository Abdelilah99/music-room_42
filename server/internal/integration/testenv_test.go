// Integration tests for all HTTP handlers.
// These tests require a running PostgreSQL instance.
// Set DATABASE_URL to a test database before running, e.g.:
//
//	DATABASE_URL="postgres://postgres:postgres@localhost:5437/musicroom?sslmode=disable" go test ./internal/integration/...
//
// Mailpit must be running on localhost:1025 for auth tests that send email.
package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"music-room/internal/auth"
	"music-room/internal/handler"
	"music-room/internal/hub"
	"music-room/internal/middleware"
	"music-room/internal/model"
	"music-room/internal/repository"
	"music-room/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

func init() {
	gin.SetMode(gin.TestMode)
	middleware.RegisterJSONTagNames()
}

// newPool connects to a real PostgreSQL instance.
// The test is skipped if DATABASE_URL is not set.
func newPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set — skipping integration test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// newRouter wires up the full real router with real repos and services.
// Email is pointed at localhost:1025 (Mailpit). Rate limiting is permissive.
func newRouter(pool *pgxpool.Pool) *gin.Engine {
	authRepo := repository.NewAuthRepository(pool)
	emailSvc := service.NewEmailService("localhost", "1025", "noreply@musicroom.test", "", "")
	authSvc := service.NewAuthService(authRepo, emailSvc, "http://localhost:8081")
	authHandler := handler.NewAuthHandler(authSvc)

	userRepo := repository.NewPostgresUserRepository(pool)
	tokenRepo := repository.NewPostgresRefreshTokenRepository(pool)
	jwtService := auth.NewJWTService()
	jwtHandler := auth.NewHandler(userRepo, tokenRepo, jwtService)

	profileRepo := repository.NewProfileRepository(pool)
	profileSvc := service.NewProfileService(profileRepo)
	profileHandler := handler.NewProfileHandler(profileSvc)

	friendRepo := repository.NewFriendRepository(pool)
	friendSvc := service.NewFriendService(friendRepo)
	friendHandler := handler.NewFriendHandler(friendSvc)

	musicSvc := service.NewMusicService()
	musicHandler := handler.NewMusicHandler(musicSvc)

	oauthRepo := repository.NewOAuthRepository(pool)
	googleHandler := auth.NewGoogleHandler(oauthRepo, userRepo, tokenRepo, jwtService)

	hubManager := hub.NewHubManager()

	eventRepo := repository.NewEventRepository(pool)
	eventSvc := service.NewEventService(eventRepo)
	eventHandler := handler.NewEventHandler(eventSvc, hubManager)

	trackRepo := repository.NewTrackRepository(pool)
	trackSvc := service.NewTrackService(eventRepo, trackRepo)
	trackHandler := handler.NewTrackHandler(trackSvc, hubManager)

	deviceRepo := repository.NewDeviceRepository(pool)
	deviceSvc := service.NewDeviceService(deviceRepo)
	deviceHandler := handler.NewDeviceHandler(deviceSvc)

	delegRepo := repository.NewDelegationRepository(pool)
	delegSvc := service.NewDelegationService(deviceRepo, delegRepo)
	delegHandler := handler.NewDelegationHandler(delegSvc)

	commandHandler := handler.NewCommandHandler(deviceSvc, hubManager)

	playlistRepo := repository.NewPlaylistRepository(pool)
	playlistSvc := service.NewPlaylistService(playlistRepo)
	playlistHandler := handler.NewPlaylistHandler(playlistSvc, hubManager)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.NewCORS("*"))
	r.Use(middleware.NewRateLimiter("100000-M"))

	r.GET("/health", func(c *gin.Context) { c.JSON(200, gin.H{"status": "UP"}) })

	jwtMW := auth.NewMiddleware(jwtService)

	v1 := r.Group("/api/v1")

	authGroup := v1.Group("/auth")
	authGroup.Use(middleware.NewRateLimiter("100000-M"))
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
	users.Use(jwtMW.Authenticate())
	{
		users.GET("/me", profileHandler.GetMyProfile)
		users.PATCH("/me", profileHandler.UpdateMyProfile)
		users.GET("/search", profileHandler.SearchUsers)
		users.GET("/:id", profileHandler.GetUserProfile)
	}

	link := v1.Group("/auth/link")
	link.Use(jwtMW.Authenticate())
	{
		link.POST("/google", googleHandler.LinkGoogle)
	}

	friends := v1.Group("/friends")
	friends.Use(jwtMW.Authenticate())
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
	music.Use(jwtMW.Authenticate())
	{
		music.GET("/search", musicHandler.Search)
	}

	devices := v1.Group("/devices")
	devices.Use(jwtMW.Authenticate())
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

	events := v1.Group("/events")
	events.Use(jwtMW.Authenticate())
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

	playlists := v1.Group("/playlists")
	playlists.Use(jwtMW.Authenticate())
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

	return r
}

// createUser inserts a verified user directly into the DB and returns the user
// and a valid JWT access token for that user. No email is sent.
func createUser(t *testing.T, pool *pgxpool.Pool) (*model.User, string) {
	t.Helper()
	ctx := context.Background()

	id := uuid.New()
	email := "test+" + id.String()[:8] + "@integration.test"
	hash, err := bcrypt.GenerateFromPassword([]byte("testpassword123"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	var user model.User
	err = pool.QueryRow(ctx,
		`INSERT INTO users (id, email, password_hash, is_verified)
		 VALUES ($1, $2, $3, true)
		 RETURNING id, email, password_hash, is_verified, subscription_tier, created_at`,
		id, email, string(hash),
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.IsVerified, &user.SubscriptionTier, &user.CreatedAt)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}

	t.Cleanup(func() {
		pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, user.ID)
	})

	token, err := auth.NewJWTService().GenerateAccessToken(&user)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	return &user, token
}

// bearer returns the Authorization header value for the given token.
func bearer(token string) string {
	return "Bearer " + token
}

// req fires an HTTP request against the router and returns the recorder.
func req(router http.Handler, method, path string, body any, headers ...string) *httptest.ResponseRecorder {
	var bodyReader *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(b)
	} else {
		bodyReader = bytes.NewReader(nil)
	}

	r, _ := http.NewRequest(method, path, bodyReader)
	if body != nil {
		r.Header.Set("Content-Type", "application/json")
	}
	for i := 0; i+1 < len(headers); i += 2 {
		r.Header.Set(headers[i], headers[i+1])
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	return w
}

// jsonBody parses the response body as a map.
func jsonBody(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &m); err != nil {
		t.Fatalf("parse body: %v\nbody was: %s", err, w.Body.String())
	}
	return m
}

// mustString extracts a string field from a decoded JSON map.
func mustString(t *testing.T, m map[string]any, key string) string {
	t.Helper()
	v, ok := m[key]
	if !ok {
		t.Fatalf("key %q not found in %v", key, m)
	}
	s, ok := v.(string)
	if !ok {
		t.Fatalf("key %q is not a string: %T", key, v)
	}
	return s
}

// createEvent inserts an event owned by the given user and registers cleanup.
func createEvent(t *testing.T, pool *pgxpool.Pool, ownerID uuid.UUID) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	err := pool.QueryRow(context.Background(),
		`INSERT INTO events (owner_id, name, visibility, license) VALUES ($1, $2, 'public', 0) RETURNING id`,
		ownerID, "Test Event "+uuid.New().String()[:8],
	).Scan(&id)
	if err != nil {
		t.Fatalf("insert event: %v", err)
	}
	t.Cleanup(func() {
		pool.Exec(context.Background(), `DELETE FROM events WHERE id = $1`, id)
	})
	return id
}

// createTrack inserts a track in an event and registers cleanup.
func createTrack(t *testing.T, pool *pgxpool.Pool, eventID, suggestedBy uuid.UUID) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	err := pool.QueryRow(context.Background(),
		`INSERT INTO tracks (event_id, external_id, title, artist, suggested_by)
		 VALUES ($1, $2, 'Test Track', 'Test Artist', $3) RETURNING id`,
		eventID, "ext-"+uuid.New().String()[:8], suggestedBy,
	).Scan(&id)
	if err != nil {
		t.Fatalf("insert track: %v", err)
	}
	return id
}

// createDevice inserts a device owned by the given user and registers cleanup.
func createDevice(t *testing.T, pool *pgxpool.Pool, ownerID uuid.UUID) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	err := pool.QueryRow(context.Background(),
		`INSERT INTO devices (user_id, name, platform, model) VALUES ($1, 'Test Device', 'android', 'Pixel') RETURNING id`,
		ownerID,
	).Scan(&id)
	if err != nil {
		t.Fatalf("insert device: %v", err)
	}
	t.Cleanup(func() {
		pool.Exec(context.Background(), `DELETE FROM devices WHERE id = $1`, id)
	})
	return id
}

// createPlaylist inserts a playlist owned by the given user and registers cleanup.
func createPlaylist(t *testing.T, pool *pgxpool.Pool, ownerID uuid.UUID) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	err := pool.QueryRow(context.Background(),
		`INSERT INTO playlists (owner_id, name, visibility, license) VALUES ($1, $2, 'public', 0) RETURNING id`,
		ownerID, "Test Playlist "+uuid.New().String()[:8],
	).Scan(&id)
	if err != nil {
		t.Fatalf("insert playlist: %v", err)
	}
	t.Cleanup(func() {
		pool.Exec(context.Background(), `DELETE FROM playlists WHERE id = $1`, id)
	})
	return id
}

// addPlaylistTrack inserts a track into a playlist and registers cleanup.
func addPlaylistTrack(t *testing.T, pool *pgxpool.Pool, playlistID, addedBy uuid.UUID) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	extID := "ext-" + strings.ReplaceAll(uuid.New().String(), "-", "")[:12]
	err := pool.QueryRow(context.Background(),
		`INSERT INTO playlist_tracks (playlist_id, external_id, title, artist, position, added_by)
		 VALUES ($1, $2, 'Test Track', 'Test Artist', 1, $3) RETURNING id`,
		playlistID, extID, addedBy,
	).Scan(&id)
	if err != nil {
		t.Fatalf("insert playlist track: %v", err)
	}
	return id
}
