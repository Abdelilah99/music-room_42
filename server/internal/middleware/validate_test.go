package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func setupValidateRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	RegisterJSONTagNames()
	r := gin.New()
	r.POST("/test", func(c *gin.Context) {
		var req struct {
			Email    string `json:"email"    binding:"required,email"`
			Password string `json:"password" binding:"required,min=8"`
		}
		if !BindAndValidate(c, &req) {
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	return r
}

func post(r *gin.Engine, body string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	return w
}

func TestBindAndValidatePassesValidBody(t *testing.T) {
	r := setupValidateRouter()
	w := post(r, `{"email":"user@example.com","password":"strongpass"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestBindAndValidateMissingRequiredField(t *testing.T) {
	r := setupValidateRouter()
	w := post(r, `{"password":"strongpass"}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "email") {
		t.Errorf("expected error to mention 'email', got: %s", body)
	}
}

func TestBindAndValidateInvalidEmail(t *testing.T) {
	r := setupValidateRouter()
	w := post(r, `{"email":"not-an-email","password":"strongpass"}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "email") {
		t.Errorf("expected error to mention 'email', got: %s", body)
	}
}

func TestBindAndValidatePasswordTooShort(t *testing.T) {
	r := setupValidateRouter()
	w := post(r, `{"email":"user@example.com","password":"short"}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "password") {
		t.Errorf("expected error to mention 'password', got: %s", body)
	}
}

func TestBindAndValidateMalformedJSON(t *testing.T) {
	r := setupValidateRouter()
	w := post(r, `{not valid json`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestBindAndValidateEmptyBody(t *testing.T) {
	r := setupValidateRouter()
	w := post(r, `{}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}
