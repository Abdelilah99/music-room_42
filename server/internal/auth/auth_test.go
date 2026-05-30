package auth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"music-room/internal/auth"
	"music-room/internal/model"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// Mock User Repository
type mockUserRepo struct {
	mu    sync.RWMutex
	users map[string]*model.User
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{
		users: make(map[string]*model.User),
	}
}

func (m *mockUserRepo) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, u := range m.users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, nil
}

func (m *mockUserRepo) GetByID(ctx context.Context, id string) (*model.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	u, ok := m.users[id]
	if !ok {
		return nil, nil
	}
	return u, nil
}

func (m *mockUserRepo) Create(ctx context.Context, email, passwordHash string) (*model.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	u := &model.User{
		ID:               "user-uuid-" + email,
		Email:            email,
		PasswordHash:     passwordHash,
		IsVerified:       true,
		SubscriptionTier: "free",
		CreatedAt:        time.Now(),
	}
	m.users[u.ID] = u
	return u, nil
}

// Mock Refresh Token Repository
type mockRefreshTokenRepo struct {
	mu     sync.RWMutex
	tokens map[string]*model.RefreshToken
}

func newMockRefreshTokenRepo() *mockRefreshTokenRepo {
	return &mockRefreshTokenRepo{
		tokens: make(map[string]*model.RefreshToken),
	}
}

func (m *mockRefreshTokenRepo) Create(ctx context.Context, token *model.RefreshToken) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	token.ID = "token-uuid-" + token.TokenHash[:8]
	token.CreatedAt = time.Now()
	m.tokens[token.TokenHash] = token
	return nil
}

func (m *mockRefreshTokenRepo) GetByHash(ctx context.Context, tokenHash string) (*model.RefreshToken, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	t, ok := m.tokens[tokenHash]
	if !ok {
		return nil, nil
	}
	return t, nil
}

func (m *mockRefreshTokenRepo) Revoke(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, t := range m.tokens {
		if t.ID == id {
			now := time.Now()
			t.RevokedAt = &now
			return nil
		}
	}
	return errors.New("token not found")
}

func (m *mockRefreshTokenRepo) RevokeAllForUser(ctx context.Context, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	for _, t := range m.tokens {
		if t.UserID == userID {
			t.RevokedAt = &now
		}
	}
	return nil
}

func TestJWTService(t *testing.T) {
	os.Setenv("JWT_SECRET", "test-secret-key-123456789-0")
	os.Setenv("JWT_ACCESS_TTL", "1s")
	os.Setenv("JWT_REFRESH_TTL", "10s")
	defer func() {
		os.Unsetenv("JWT_SECRET")
		os.Unsetenv("JWT_ACCESS_TTL")
		os.Unsetenv("JWT_REFRESH_TTL")
	}()

	s := auth.NewJWTService()
	user := &model.User{
		ID:               "123",
		Email:            "test@example.com",
		SubscriptionTier: "premium",
	}

	token, err := s.GenerateAccessToken(user)
	if err != nil {
		t.Fatalf("failed to generate access token: %v", err)
	}

	claims, err := s.ValidateAccessToken(token)
	if err != nil {
		t.Fatalf("failed to validate access token: %v", err)
	}

	if claims.UserID != user.ID {
		t.Errorf("expected user ID %s, got %s", user.ID, claims.UserID)
	}

	if claims.SubscriptionTier != "premium" {
		t.Errorf("expected premium tier, got %s", claims.SubscriptionTier)
	}

	// Test expiration
	time.Sleep(1500 * time.Millisecond)
	_, err = s.ValidateAccessToken(token)
	if err == nil {
		t.Error("expected token to be expired and validation to fail")
	}

	// Test refresh token generation/hashing
	refStr, err := s.GenerateRefreshTokenString()
	if err != nil {
		t.Fatalf("failed to generate refresh token string: %v", err)
	}

	if len(refStr) == 0 {
		t.Error("refresh token string is empty")
	}

	hash1 := s.HashRefreshToken(refStr)
	hash2 := s.HashRefreshToken(refStr)
	if hash1 != hash2 {
		t.Error("hashing same token should yield same result")
	}
}

func TestAuthMiddleware(t *testing.T) {
	os.Setenv("JWT_SECRET", "middleware-test-secret")
	defer os.Unsetenv("JWT_SECRET")

	s := auth.NewJWTService()
	mw := auth.NewMiddleware(s)

	r := gin.New()
	r.Use(mw.Authenticate())
	r.GET("/test", func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		c.String(http.StatusOK, userID.(string))
	})

	// Test 1: Missing Header
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}

	// Test 2: Invalid Format
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "invalid-format")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}

	// Test 3: Valid Access Token
	user := &model.User{ID: "user-456", Email: "test@example.com"}
	token, _ := s.GenerateAccessToken(user)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if w.Body.String() != "user-456" {
		t.Errorf("expected user-456 in body, got %s", w.Body.String())
	}
}

func TestRequireOwnership(t *testing.T) {
	r := gin.New()
	r.GET("/users/:id", func(c *gin.Context) {
		c.Set("user_id", "user-123")
		c.Next()
	}, auth.RequireOwnership("id"), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// Test 1: Match
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/users/user-123", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	// Test 2: Mismatch
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/users/user-456", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestLoginHandler(t *testing.T) {
	os.Setenv("JWT_SECRET", "handler-login-test")
	defer os.Unsetenv("JWT_SECRET")

	userRepo := newMockUserRepo()
	tokenRepo := newMockRefreshTokenRepo()
	jwtService := auth.NewJWTService()
	h := auth.NewHandler(userRepo, tokenRepo, jwtService)

	password := "password123"
	hashed, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	_, _ = userRepo.Create(context.Background(), "user@example.com", string(hashed))

	r := gin.New()
	r.POST("/login", h.Login)

	// Test 1: Success login
	body, _ := json.Marshal(map[string]string{
		"email":    "user@example.com",
		"password": password,
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var res map[string]string
	_ = json.Unmarshal(w.Body.Bytes(), &res)
	if res["access_token"] == "" || res["refresh_token"] == "" {
		t.Error("tokens should not be empty")
	}

	// Test 2: Wrong Password
	body, _ = json.Marshal(map[string]string{
		"email":    "user@example.com",
		"password": "wrong-password",
	})
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestRefreshHandler(t *testing.T) {
	os.Setenv("JWT_SECRET", "handler-refresh-test")
	defer os.Unsetenv("JWT_SECRET")

	userRepo := newMockUserRepo()
	tokenRepo := newMockRefreshTokenRepo()
	jwtService := auth.NewJWTService()
	h := auth.NewHandler(userRepo, tokenRepo, jwtService)

	user, _ := userRepo.Create(context.Background(), "ref@example.com", "dummy-hash")

	// Generate initial pair
	w := httptest.NewRecorder()
	r := gin.New()
	r.POST("/refresh", h.Refresh)

	_, refStr, _ := h.TestGenerateTokenPair(context.Background(), user)

	// Test 1: Successful Refresh Rotation
	body, _ := json.Marshal(map[string]string{
		"refresh_token": refStr,
	})
	req, _ := http.NewRequest(http.MethodPost, "/refresh", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var res map[string]string
	_ = json.Unmarshal(w.Body.Bytes(), &res)
	newAccessToken := res["access_token"]
	newRefreshToken := res["refresh_token"]

	if newAccessToken == "" || newRefreshToken == "" {
		t.Error("new token pair should not be empty")
	}

	// Test 2: Double Spend (Replay attack detection)
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodPost, "/refresh", bytes.NewBuffer(body))
	req2.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 on reused refresh token, got %d", w2.Code)
	}

	// Verify all tokens for the user were revoked on double spend detection
	tokenRepo.mu.RLock()
	for _, tk := range tokenRepo.tokens {
		if tk.UserID == user.ID && tk.RevokedAt == nil {
			t.Error("all tokens should be revoked after double spend detection")
		}
	}
	tokenRepo.mu.RUnlock()
}

func TestLogoutHandler(t *testing.T) {
	userRepo := newMockUserRepo()
	tokenRepo := newMockRefreshTokenRepo()
	jwtService := auth.NewJWTService()
	h := auth.NewHandler(userRepo, tokenRepo, jwtService)

	user, _ := userRepo.Create(context.Background(), "logout@example.com", "dummy-hash")
	_, refStr, _ := h.TestGenerateTokenPair(context.Background(), user)

	r := gin.New()
	r.POST("/logout", h.Logout)

	body, _ := json.Marshal(map[string]string{
		"refresh_token": refStr,
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/logout", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	hash := jwtService.HashRefreshToken(refStr)
	token, _ := tokenRepo.GetByHash(context.Background(), hash)
	if token == nil || token.RevokedAt == nil {
		t.Error("expected token to be revoked")
	}
}
