package integration_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func TestAuth_Register_Success(t *testing.T) {
	pool := newPool(t)
	router := newRouter(pool)

	email := "reg+" + uuid.New().String()[:8] + "@integration.test"
	w := req(router, http.MethodPost, "/api/v1/auth/register", map[string]any{
		"email":    email,
		"password": "securepassword123",
	})

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	t.Cleanup(func() {
		pool.Exec(context.Background(), `DELETE FROM users WHERE email = $1`, email)
	})
}

func TestAuth_Register_DuplicateEmail_Returns409(t *testing.T) {
	pool := newPool(t)
	router := newRouter(pool)
	user, _ := createUser(t, pool)

	w := req(router, http.MethodPost, "/api/v1/auth/register", map[string]any{
		"email":    user.Email,
		"password": "securepassword123",
	})

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuth_Register_WeakPassword_Returns400(t *testing.T) {
	pool := newPool(t)
	router := newRouter(pool)

	w := req(router, http.MethodPost, "/api/v1/auth/register", map[string]any{
		"email":    "weak+" + uuid.New().String()[:8] + "@integration.test",
		"password": "short",
	})

	// Gin binding (min=8) fires before the service, returning 400
	if w.Code != http.StatusBadRequest && w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 400/422, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuth_Register_InvalidEmail_Returns422(t *testing.T) {
	pool := newPool(t)
	router := newRouter(pool)

	w := req(router, http.MethodPost, "/api/v1/auth/register", map[string]any{
		"email":    "not-an-email",
		"password": "securepassword123",
	})

	if w.Code != http.StatusUnprocessableEntity && w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400/422, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuth_Login_Success(t *testing.T) {
	pool := newPool(t)
	router := newRouter(pool)
	user, _ := createUser(t, pool)

	w := req(router, http.MethodPost, "/api/v1/auth/login", map[string]any{
		"email":    user.Email,
		"password": "testpassword123",
	})

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body := jsonBody(t, w)
	if mustString(t, body, "access_token") == "" {
		t.Error("expected non-empty access_token")
	}
}

func TestAuth_Login_WrongPassword_Returns401(t *testing.T) {
	pool := newPool(t)
	router := newRouter(pool)
	user, _ := createUser(t, pool)

	w := req(router, http.MethodPost, "/api/v1/auth/login", map[string]any{
		"email":    user.Email,
		"password": "wrongpassword",
	})

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuth_Login_UnverifiedUser_Returns403(t *testing.T) {
	pool := newPool(t)
	router := newRouter(pool)

	id := uuid.New()
	email := "unverified+" + id.String()[:8] + "@integration.test"
	hash, _ := bcrypt.GenerateFromPassword([]byte("testpassword123"), bcrypt.MinCost)
	_, err := pool.Exec(context.Background(),
		`INSERT INTO users (id, email, password_hash, is_verified) VALUES ($1, $2, $3, false)`,
		id, email, string(hash),
	)
	if err != nil {
		t.Fatalf("insert unverified user: %v", err)
	}
	t.Cleanup(func() {
		pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, id)
	})

	w := req(router, http.MethodPost, "/api/v1/auth/login", map[string]any{
		"email":    email,
		"password": "testpassword123",
	})

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuth_Logout_Success(t *testing.T) {
	pool := newPool(t)
	router := newRouter(pool)
	user, _ := createUser(t, pool)

	// First log in to get a refresh token
	loginResp := req(router, http.MethodPost, "/api/v1/auth/login", map[string]any{
		"email":    user.Email,
		"password": "testpassword123",
	})
	if loginResp.Code != http.StatusOK {
		t.Fatalf("login failed: %d %s", loginResp.Code, loginResp.Body.String())
	}
	loginBody := jsonBody(t, loginResp)
	refreshToken := mustString(t, loginBody, "refresh_token")

	w := req(router, http.MethodPost, "/api/v1/auth/logout", map[string]any{
		"refresh_token": refreshToken,
	})

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuth_Refresh_Success(t *testing.T) {
	pool := newPool(t)
	router := newRouter(pool)
	user, _ := createUser(t, pool)

	loginResp := req(router, http.MethodPost, "/api/v1/auth/login", map[string]any{
		"email":    user.Email,
		"password": "testpassword123",
	})
	loginBody := jsonBody(t, loginResp)
	refreshToken := mustString(t, loginBody, "refresh_token")

	w := req(router, http.MethodPost, "/api/v1/auth/refresh", map[string]any{
		"refresh_token": refreshToken,
	})

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body := jsonBody(t, w)
	if mustString(t, body, "access_token") == "" {
		t.Error("expected non-empty access_token")
	}
}

func TestAuth_Refresh_InvalidToken_Returns401(t *testing.T) {
	pool := newPool(t)
	router := newRouter(pool)

	w := req(router, http.MethodPost, "/api/v1/auth/refresh", map[string]any{
		"refresh_token": "totally-invalid-token",
	})

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuth_VerifyEmail_InvalidToken_Returns400(t *testing.T) {
	pool := newPool(t)
	router := newRouter(pool)

	w := req(router, http.MethodGet, "/api/v1/auth/verify-email?token=not-a-uuid", nil)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuth_VerifyEmail_UnknownToken_Returns400(t *testing.T) {
	pool := newPool(t)
	router := newRouter(pool)

	w := req(router, http.MethodGet, "/api/v1/auth/verify-email?token="+uuid.New().String(), nil)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuth_ResetPassword_ExpiredToken_Returns400(t *testing.T) {
	pool := newPool(t)
	router := newRouter(pool)
	user, _ := createUser(t, pool)

	// Insert an expired reset token directly
	tokenID := uuid.New()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO password_reset_tokens (id, user_id, token, expires_at) VALUES ($1, $2, $3, $4)`,
		uuid.New(), user.ID, tokenID, time.Now().Add(-time.Hour),
	)
	if err != nil {
		t.Fatalf("insert reset token: %v", err)
	}
	t.Cleanup(func() {
		pool.Exec(context.Background(), `DELETE FROM password_reset_tokens WHERE token = $1`, tokenID)
	})

	w := req(router, http.MethodPost, "/api/v1/auth/reset-password", map[string]any{
		"token":    tokenID.String(),
		"password": "newpassword123",
	})

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuth_ForgotPassword_UnknownEmail_ReturnsOK(t *testing.T) {
	pool := newPool(t)
	router := newRouter(pool)

	// Unknown email should return 200 (no enumeration leak)
	w := req(router, http.MethodPost, "/api/v1/auth/forgot-password", map[string]any{
		"email": "nobody+" + uuid.New().String()[:8] + "@integration.test",
	})

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuth_ResendVerification_UnknownEmail_ReturnsOK(t *testing.T) {
	pool := newPool(t)
	router := newRouter(pool)

	w := req(router, http.MethodPost, "/api/v1/auth/resend-verification", map[string]any{
		"email": "nobody+" + uuid.New().String()[:8] + "@integration.test",
	})

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}
