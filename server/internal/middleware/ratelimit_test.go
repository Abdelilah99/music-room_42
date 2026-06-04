package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRateLimitReturns429WhenExceeded(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(NewRateLimiter("2-M")) // 2 requests per minute
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	do := func() int {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "127.0.0.1:9999"
		r.ServeHTTP(w, req)
		return w.Code
	}

	if code := do(); code != http.StatusOK {
		t.Fatalf("request 1: want 200, got %d", code)
	}
	if code := do(); code != http.StatusOK {
		t.Fatalf("request 2: want 200, got %d", code)
	}
	if code := do(); code != http.StatusTooManyRequests {
		t.Fatalf("request 3: want 429, got %d", code)
	}
}

func TestRateLimitDifferentIPsAreIndependent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(NewRateLimiter("1-M"))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	doAs := func(ip string) int {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = ip + ":1234"
		r.ServeHTTP(w, req)
		return w.Code
	}

	// First request for each IP should pass even though 1-M limit is tight
	if code := doAs("10.0.0.1"); code != http.StatusOK {
		t.Fatalf("IP 1 first request: want 200, got %d", code)
	}
	if code := doAs("10.0.0.2"); code != http.StatusOK {
		t.Fatalf("IP 2 first request: want 200, got %d", code)
	}
}

func TestInvalidRatePanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for invalid rate format")
		}
	}()
	NewRateLimiter("not-a-rate")
}
