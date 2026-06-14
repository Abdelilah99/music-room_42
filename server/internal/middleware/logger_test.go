package middleware

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func setupLoggerRouter(t *testing.T, buf *bytes.Buffer, statusCode int, userID string) *gin.Engine {
	t.Helper()
	slog.SetDefault(slog.New(slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug})))

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(NewLogger())
	r.GET("/test", func(c *gin.Context) {
		if userID != "" {
			c.Set("user_id", userID)
		}
		c.Status(statusCode)
	})
	return r
}

func loggedFields(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()
	var fields map[string]any
	if err := json.NewDecoder(buf).Decode(&fields); err != nil {
		t.Fatalf("log line is not valid JSON: %v", err)
	}
	return fields
}

func TestLogger_InfoOn2xx(t *testing.T) {
	var buf bytes.Buffer
	r := setupLoggerRouter(t, &buf, http.StatusOK, "")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/test", nil))

	fields := loggedFields(t, &buf)
	if fields["level"] != "INFO" {
		t.Errorf("expected level INFO, got %v", fields["level"])
	}
	if fields["status"] != float64(200) {
		t.Errorf("expected status 200, got %v", fields["status"])
	}
}

func TestLogger_WarnOn4xx(t *testing.T) {
	var buf bytes.Buffer
	r := setupLoggerRouter(t, &buf, http.StatusBadRequest, "")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/test", nil))

	fields := loggedFields(t, &buf)
	if fields["level"] != "WARN" {
		t.Errorf("expected level WARN, got %v", fields["level"])
	}
}

func TestLogger_ErrorOn5xx(t *testing.T) {
	var buf bytes.Buffer
	r := setupLoggerRouter(t, &buf, http.StatusInternalServerError, "")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/test", nil))

	fields := loggedFields(t, &buf)
	if fields["level"] != "ERROR" {
		t.Errorf("expected level ERROR, got %v", fields["level"])
	}
}

func TestLogger_MobileHeaders(t *testing.T) {
	var buf bytes.Buffer
	r := setupLoggerRouter(t, &buf, http.StatusOK, "")

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Platform", "android")
	req.Header.Set("X-Device-Model", "Pixel 8")
	req.Header.Set("X-App-Version", "1.2.3")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	fields := loggedFields(t, &buf)
	if fields["platform"] != "android" {
		t.Errorf("expected platform android, got %v", fields["platform"])
	}
	if fields["device_model"] != "Pixel 8" {
		t.Errorf("expected device_model 'Pixel 8', got %v", fields["device_model"])
	}
	if fields["app_version"] != "1.2.3" {
		t.Errorf("expected app_version 1.2.3, got %v", fields["app_version"])
	}
}

func TestLogger_MissingHeadersAreEmpty(t *testing.T) {
	var buf bytes.Buffer
	r := setupLoggerRouter(t, &buf, http.StatusOK, "")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/test", nil))

	fields := loggedFields(t, &buf)
	if fields["platform"] != "" {
		t.Errorf("expected empty platform, got %v", fields["platform"])
	}
	if fields["device_model"] != "" {
		t.Errorf("expected empty device_model, got %v", fields["device_model"])
	}
	if fields["app_version"] != "" {
		t.Errorf("expected empty app_version, got %v", fields["app_version"])
	}
}

func TestLogger_UserIDLogged(t *testing.T) {
	var buf bytes.Buffer
	r := setupLoggerRouter(t, &buf, http.StatusOK, "user-abc-123")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/test", nil))

	fields := loggedFields(t, &buf)
	if fields["user_id"] != "user-abc-123" {
		t.Errorf("expected user_id 'user-abc-123', got %v", fields["user_id"])
	}
}
