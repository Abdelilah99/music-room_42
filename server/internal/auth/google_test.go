package auth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"music-room/internal/auth"
	"music-room/internal/model"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ── Mock OAuthRepository ──────────────────────────────────────────────────────

type mockOAuthRepo struct {
	mu        sync.RWMutex
	providers map[string]*model.UserProvider // key: provider+":"+providerID
	userRepo  *mockUserRepo                  // shared so GetByID can find OAuth-created users
}

func newMockOAuthRepo(userRepo *mockUserRepo) *mockOAuthRepo {
	return &mockOAuthRepo{
		providers: make(map[string]*model.UserProvider),
		userRepo:  userRepo,
	}
}

func (m *mockOAuthRepo) GetProviderByProviderID(ctx context.Context, provider, providerID string) (*model.UserProvider, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	key := provider + ":" + providerID
	p, ok := m.providers[key]
	if !ok {
		return nil, nil
	}
	return p, nil
}

func (m *mockOAuthRepo) CreateProvider(ctx context.Context, userID uuid.UUID, provider, providerID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := provider + ":" + providerID
	m.providers[key] = &model.UserProvider{
		ID:         uuid.New(),
		UserID:     userID,
		Provider:   provider,
		ProviderID: providerID,
		CreatedAt:  time.Now(),
	}
	return nil
}

func (m *mockOAuthRepo) CreateOAuthUser(ctx context.Context, email string) (*model.User, error) {
	u := &model.User{
		ID:               uuid.New(),
		Email:            email,
		PasswordHash:     "",
		IsVerified:       true,
		SubscriptionTier: "free",
		CreatedAt:        time.Now(),
	}
	// Store in the shared userRepo so GetByID works on subsequent sign-ins
	m.userRepo.mu.Lock()
	m.userRepo.users[u.ID] = u
	m.userRepo.mu.Unlock()
	return u, nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func setupGoogleRouter(h *auth.GoogleHandler, jwtService *auth.JWTService) *gin.Engine {
	r := gin.New()
	r.POST("/auth/google", h.SignIn)

	mw := auth.NewMiddleware(jwtService)
	protected := r.Group("")
	protected.Use(mw.Authenticate())
	protected.POST("/auth/link/google", h.LinkGoogle)
	return r
}

func fakeVerifier(email, sub string) func(ctx context.Context, idToken string) (*auth.TestGoogleClaims, error) {
	return func(ctx context.Context, idToken string) (*auth.TestGoogleClaims, error) {
		if idToken == "invalid" {
			return nil, auth.ErrInvalidGoogleToken
		}
		return &auth.TestGoogleClaims{
			Sub:           sub,
			Email:         email,
			EmailVerified: "true",
			Aud:           "test-client-id",
		}, nil
	}
}

func signInBody(idToken string) *bytes.Buffer {
	b, _ := json.Marshal(map[string]string{"id_token": idToken})
	return bytes.NewBuffer(b)
}

// ── Tests ─────────────────────────────────────────────────────────────────────

func TestGoogleSignIn_NewUser(t *testing.T) {
	os.Setenv("JWT_SECRET", "google-test-secret")
	defer os.Unsetenv("JWT_SECRET")

	userRepo := newMockUserRepo()
	oauthRepo := newMockOAuthRepo(userRepo)
	tokenRepo := newMockRefreshTokenRepo()
	jwtService := auth.NewJWTService()

	h := auth.NewGoogleHandlerForTest(oauthRepo, userRepo, tokenRepo, jwtService,
		fakeVerifier("newuser@gmail.com", "google-sub-001"))

	r := setupGoogleRouter(h, jwtService)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/auth/google", signInBody("valid-token"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var res map[string]string
	_ = json.Unmarshal(w.Body.Bytes(), &res)
	if res["access_token"] == "" || res["refresh_token"] == "" {
		t.Error("expected JWT pair in response")
	}

	// Provider row must exist
	p, _ := oauthRepo.GetProviderByProviderID(context.Background(), "google", "google-sub-001")
	if p == nil {
		t.Error("expected provider row to be created")
	}
}

func TestGoogleSignIn_ReturnsValidJWT(t *testing.T) {
	os.Setenv("JWT_SECRET", "google-test-secret")
	defer os.Unsetenv("JWT_SECRET")

	userRepo := newMockUserRepo()
	oauthRepo := newMockOAuthRepo(userRepo)
	tokenRepo := newMockRefreshTokenRepo()
	jwtService := auth.NewJWTService()

	h := auth.NewGoogleHandlerForTest(oauthRepo, userRepo, tokenRepo, jwtService,
		fakeVerifier("jwtcheck@gmail.com", "google-sub-jwt"))

	r := setupGoogleRouter(h, jwtService)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/auth/google", signInBody("valid-token"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	var res map[string]string
	_ = json.Unmarshal(w.Body.Bytes(), &res)

	// The returned access token must be verifiable
	claims, err := jwtService.ValidateAccessToken(res["access_token"])
	if err != nil {
		t.Fatalf("returned access token failed validation: %v", err)
	}
	if claims.Email != "jwtcheck@gmail.com" {
		t.Errorf("expected email jwtcheck@gmail.com, got %s", claims.Email)
	}
}

func TestGoogleSignIn_ExistingProvider_ReusesUser(t *testing.T) {
	os.Setenv("JWT_SECRET", "google-test-secret")
	defer os.Unsetenv("JWT_SECRET")

	userRepo := newMockUserRepo()
	oauthRepo := newMockOAuthRepo(userRepo)
	tokenRepo := newMockRefreshTokenRepo()
	jwtService := auth.NewJWTService()

	h := auth.NewGoogleHandlerForTest(oauthRepo, userRepo, tokenRepo, jwtService,
		fakeVerifier("returning@gmail.com", "google-sub-002"))

	r := setupGoogleRouter(h, jwtService)

	// First sign-in
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest(http.MethodPost, "/auth/google", signInBody("valid-token"))
	req1.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w1, req1)
	if w1.Code != http.StatusOK {
		t.Fatalf("first sign-in failed: %d %s", w1.Code, w1.Body.String())
	}

	var res1 map[string]string
	_ = json.Unmarshal(w1.Body.Bytes(), &res1)
	claims1, _ := jwtService.ValidateAccessToken(res1["access_token"])

	// Second sign-in
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodPost, "/auth/google", signInBody("valid-token"))
	req2.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("second sign-in failed: %d %s", w2.Code, w2.Body.String())
	}

	var res2 map[string]string
	_ = json.Unmarshal(w2.Body.Bytes(), &res2)
	claims2, _ := jwtService.ValidateAccessToken(res2["access_token"])

	// Same user ID both times - no duplicate user created
	if claims1.UserID != claims2.UserID {
		t.Errorf("expected same user ID on re-login, got %s and %s", claims1.UserID, claims2.UserID)
	}
}

func TestGoogleSignIn_AccountLinking_NoNewUser(t *testing.T) {
	os.Setenv("JWT_SECRET", "google-test-secret")
	defer os.Unsetenv("JWT_SECRET")

	userRepo := newMockUserRepo()
	oauthRepo := newMockOAuthRepo(userRepo)
	tokenRepo := newMockRefreshTokenRepo()
	jwtService := auth.NewJWTService()

	// Pre-existing email/password account
	existing, _ := userRepo.Create(context.Background(), "existing@gmail.com", "bcrypt-hash")

	h := auth.NewGoogleHandlerForTest(oauthRepo, userRepo, tokenRepo, jwtService,
		fakeVerifier("existing@gmail.com", "google-sub-003"))

	r := setupGoogleRouter(h, jwtService)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/auth/google", signInBody("valid-token"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var res map[string]string
	_ = json.Unmarshal(w.Body.Bytes(), &res)
	claims, _ := jwtService.ValidateAccessToken(res["access_token"])

	// Must return the existing user, not create a new one
	if claims.UserID != existing.ID.String() {
		t.Errorf("expected existing user ID %s, got %s", existing.ID.String(), claims.UserID)
	}

	// Provider must be linked to the existing user
	p, _ := oauthRepo.GetProviderByProviderID(context.Background(), "google", "google-sub-003")
	if p == nil || p.UserID != existing.ID {
		t.Error("expected provider linked to existing user")
	}
}

func TestGoogleSignIn_InvalidToken_Returns401(t *testing.T) {
	userRepo := newMockUserRepo()
	oauthRepo := newMockOAuthRepo(userRepo)
	tokenRepo := newMockRefreshTokenRepo()
	jwtService := auth.NewJWTService()

	h := auth.NewGoogleHandlerForTest(oauthRepo, userRepo, tokenRepo, jwtService,
		fakeVerifier("x@gmail.com", "sub-x"))

	r := setupGoogleRouter(h, jwtService)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/auth/google", signInBody("invalid"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestGoogleSignIn_MissingToken_Returns400(t *testing.T) {
	userRepo := newMockUserRepo()
	oauthRepo := newMockOAuthRepo(userRepo)
	tokenRepo := newMockRefreshTokenRepo()
	jwtService := auth.NewJWTService()

	h := auth.NewGoogleHandlerForTest(oauthRepo, userRepo, tokenRepo, jwtService,
		fakeVerifier("x@gmail.com", "sub-x"))

	r := setupGoogleRouter(h, jwtService)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/auth/google",
		bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestLinkGoogle_Success(t *testing.T) {
	os.Setenv("JWT_SECRET", "google-test-secret")
	defer os.Unsetenv("JWT_SECRET")

	userRepo := newMockUserRepo()
	oauthRepo := newMockOAuthRepo(userRepo)
	tokenRepo := newMockRefreshTokenRepo()
	jwtService := auth.NewJWTService()

	user, _ := userRepo.Create(context.Background(), "link@example.com", "hash")

	h := auth.NewGoogleHandlerForTest(oauthRepo, userRepo, tokenRepo, jwtService,
		fakeVerifier("link@gmail.com", "google-sub-link"))

	r := setupGoogleRouter(h, jwtService)

	accessToken, _ := jwtService.GenerateAccessToken(user)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/auth/link/google", signInBody("valid-token"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	p, _ := oauthRepo.GetProviderByProviderID(context.Background(), "google", "google-sub-link")
	if p == nil || p.UserID != user.ID {
		t.Error("expected provider to be linked to user")
	}
}

func TestLinkGoogle_AlreadyLinkedSameUser_Returns409(t *testing.T) {
	os.Setenv("JWT_SECRET", "google-test-secret")
	defer os.Unsetenv("JWT_SECRET")

	userRepo := newMockUserRepo()
	oauthRepo := newMockOAuthRepo(userRepo)
	tokenRepo := newMockRefreshTokenRepo()
	jwtService := auth.NewJWTService()

	user, _ := userRepo.Create(context.Background(), "self@example.com", "hash")
	_ = oauthRepo.CreateProvider(context.Background(), user.ID, "google", "google-sub-self")

	h := auth.NewGoogleHandlerForTest(oauthRepo, userRepo, tokenRepo, jwtService,
		fakeVerifier("self@gmail.com", "google-sub-self"))

	r := setupGoogleRouter(h, jwtService)
	accessToken, _ := jwtService.GenerateAccessToken(user)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/auth/link/google", signInBody("valid-token"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", w.Code)
	}
}

func TestLinkGoogle_AlreadyLinkedOtherUser_Returns409(t *testing.T) {
	os.Setenv("JWT_SECRET", "google-test-secret")
	defer os.Unsetenv("JWT_SECRET")

	userRepo := newMockUserRepo()
	oauthRepo := newMockOAuthRepo(userRepo)
	tokenRepo := newMockRefreshTokenRepo()
	jwtService := auth.NewJWTService()

	userA, _ := userRepo.Create(context.Background(), "a@example.com", "hash")
	userB, _ := userRepo.Create(context.Background(), "b@example.com", "hash")
	_ = oauthRepo.CreateProvider(context.Background(), userA.ID, "google", "google-sub-taken")

	h := auth.NewGoogleHandlerForTest(oauthRepo, userRepo, tokenRepo, jwtService,
		fakeVerifier("taken@gmail.com", "google-sub-taken"))

	r := setupGoogleRouter(h, jwtService)
	accessToken, _ := jwtService.GenerateAccessToken(userB)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/auth/link/google", signInBody("valid-token"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", w.Code)
	}
}

func TestLinkGoogle_NoJWT_Returns401(t *testing.T) {
	userRepo := newMockUserRepo()
	oauthRepo := newMockOAuthRepo(userRepo)
	tokenRepo := newMockRefreshTokenRepo()
	jwtService := auth.NewJWTService()

	h := auth.NewGoogleHandlerForTest(oauthRepo, userRepo, tokenRepo, jwtService,
		fakeVerifier("x@gmail.com", "sub-x"))

	r := setupGoogleRouter(h, jwtService)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/auth/link/google", signInBody("valid-token"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}
