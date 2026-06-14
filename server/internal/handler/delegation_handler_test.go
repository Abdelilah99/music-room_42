package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"music-room/internal/model"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type mockDelegationService struct {
	isAuthorizedFn func(ctx context.Context, deviceID, callerID uuid.UUID) (bool, error)
}

func (m *mockDelegationService) Grant(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) error {
	return nil
}
func (m *mockDelegationService) Revoke(context.Context, uuid.UUID, uuid.UUID) error { return nil }
func (m *mockDelegationService) IsAuthorized(ctx context.Context, deviceID, callerID uuid.UUID) (bool, error) {
	return m.isAuthorizedFn(ctx, deviceID, callerID)
}
func (m *mockDelegationService) ListDelegated(context.Context, uuid.UUID) ([]model.DelegatedDevice, error) {
	return nil, nil
}

func setupDelegationGuardRouter(svc *mockDelegationService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewDelegationHandler(svc)

	callerID := uuid.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", callerID.String())
		c.Next()
	})

	r.GET("/api/v1/devices/:id/ws", h.RequireDelegateOrOwner(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	return r
}

func TestRequireDelegateOrOwner_Authorized_PassesThrough(t *testing.T) {
	svc := &mockDelegationService{
		isAuthorizedFn: func(_ context.Context, _, _ uuid.UUID) (bool, error) {
			return true, nil
		},
	}
	r := setupDelegationGuardRouter(svc)

	req, _ := http.NewRequest(http.MethodGet,
		"/api/v1/devices/"+uuid.New().String()+"/ws", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for an owner or delegate, got %d", w.Code)
	}
}

func TestRequireDelegateOrOwner_NotAuthorized_Returns403(t *testing.T) {
	svc := &mockDelegationService{
		isAuthorizedFn: func(_ context.Context, _, _ uuid.UUID) (bool, error) {
			return false, nil
		},
	}
	r := setupDelegationGuardRouter(svc)

	req, _ := http.NewRequest(http.MethodGet,
		"/api/v1/devices/"+uuid.New().String()+"/ws", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for a non-owner non-delegate, got %d", w.Code)
	}
}
