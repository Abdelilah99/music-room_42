package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func corsRouter(allowedOrigins string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(NewCORS(allowedOrigins))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	return r
}

func TestCORSAllowsWhitelistedOrigin(t *testing.T) {
	r := corsRouter("https://app.example.com")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "https://app.example.com")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	if w.Header().Get("Access-Control-Allow-Origin") != "https://app.example.com" {
		t.Errorf("expected CORS header for whitelisted origin, got: %q",
			w.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestCORSBlocksUnlistedOrigin(t *testing.T) {
	r := corsRouter("https://app.example.com")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "https://evil.com")
	r.ServeHTTP(w, req)

	// Server responds normally but must NOT echo the origin back
	if w.Header().Get("Access-Control-Allow-Origin") == "https://evil.com" {
		t.Error("unlisted origin must not receive CORS header")
	}
}

func TestCORSPreflightReturns204(t *testing.T) {
	r := corsRouter("https://app.example.com")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "https://app.example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "Authorization,Content-Type")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("preflight want 204, got %d", w.Code)
	}
	if w.Header().Get("Access-Control-Allow-Origin") != "https://app.example.com" {
		t.Errorf("preflight missing CORS header")
	}
}

func TestCORSMultipleOriginsAllowed(t *testing.T) {
	r := corsRouter("https://app.example.com, https://admin.example.com")

	for _, origin := range []string{"https://app.example.com", "https://admin.example.com"} {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Origin", origin)
		r.ServeHTTP(w, req)

		if w.Header().Get("Access-Control-Allow-Origin") != origin {
			t.Errorf("origin %q should be allowed, got header: %q",
				origin, w.Header().Get("Access-Control-Allow-Origin"))
		}
	}
}

func TestCORSEmptyOriginsBlocksAll(t *testing.T) {
	r := corsRouter("")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "https://anywhere.com")
	r.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") == "https://anywhere.com" {
		t.Error("empty origins list must not allow any origin")
	}
}
