package handler

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"music-room/internal/hub"
	"music-room/internal/model"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// mockDeviceService satisfies service.DeviceService for handler tests.
type mockDeviceService struct {
	getFunc func(ctx context.Context, deviceID, userID uuid.UUID) (*model.Device, error)
}

func (m *mockDeviceService) Register(ctx context.Context, userID uuid.UUID, req model.CreateDeviceRequest) (*model.Device, error) {
	return nil, nil
}
func (m *mockDeviceService) List(ctx context.Context, userID uuid.UUID) ([]model.DeviceWithDelegate, error) {
	return nil, nil
}
func (m *mockDeviceService) Get(ctx context.Context, deviceID, userID uuid.UUID) (*model.Device, error) {
	return m.getFunc(ctx, deviceID, userID)
}
func (m *mockDeviceService) Delete(ctx context.Context, deviceID, userID uuid.UUID) error {
	return nil
}

func setupCommandRouter(svc *mockDeviceService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewCommandHandler(svc, hub.NewHubManager())

	// Inject a fixed user_id so extractCallerID succeeds.
	callerID := uuid.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", callerID.String())
		c.Next()
	})

	r.POST("/api/v1/devices/:id/command", h.Send)
	r.GET("/api/v1/devices/:id/ws", h.ServeDeviceWS)
	return r
}

func TestCommandSend_InvalidAction(t *testing.T) {
	svc := &mockDeviceService{}
	r := setupCommandRouter(svc)

	body := `{"action":"stop"}`
	req, _ := http.NewRequest(http.MethodPost,
		"/api/v1/devices/"+uuid.New().String()+"/command",
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("action must be one of")) {
		t.Errorf("unexpected body: %s", w.Body.String())
	}
}

func TestCommandSend_VolumeOutOfRange(t *testing.T) {
	svc := &mockDeviceService{}
	r := setupCommandRouter(svc)

	cases := []string{
		`{"action":"volume","value":101}`,
		`{"action":"volume","value":-1}`,
		`{"action":"volume"}`,
	}

	for _, body := range cases {
		req, _ := http.NewRequest(http.MethodPost,
			"/api/v1/devices/"+uuid.New().String()+"/command",
			strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("body %s: expected 400, got %d", body, w.Code)
		}
		if !bytes.Contains(w.Body.Bytes(), []byte("volume requires a value")) {
			t.Errorf("body %s: unexpected error: %s", body, w.Body.String())
		}
	}
}

func TestCommandSend_ValidCommand_Returns200(t *testing.T) {
	svc := &mockDeviceService{}
	r := setupCommandRouter(svc)

	body := `{"action":"play"}`
	req, _ := http.NewRequest(http.MethodPost,
		"/api/v1/devices/"+uuid.New().String()+"/command",
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestServeDeviceWS_NonOwner_Returns403(t *testing.T) {
	svc := &mockDeviceService{
		getFunc: func(_ context.Context, _, _ uuid.UUID) (*model.Device, error) {
			return nil, errors.New("not found")
		},
	}
	r := setupCommandRouter(svc)

	req, _ := http.NewRequest(http.MethodGet,
		"/api/v1/devices/"+uuid.New().String()+"/ws", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("NOT_AUTHORIZED")) {
		t.Errorf("unexpected body: %s", w.Body.String())
	}
}
