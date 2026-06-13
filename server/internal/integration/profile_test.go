package integration_test

import (
	"net/http"
	"testing"
)

func TestProfile_GetMe_Success(t *testing.T) {
	pool := newPool(t)
	router := newRouter(pool)
	_, token := createUser(t, pool)

	w := req(router, http.MethodGet, "/api/v1/users/me", nil, "Authorization", bearer(token))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body := jsonBody(t, w)
	if body["email"] == nil {
		t.Error("expected email field in response")
	}
}

func TestProfile_GetMe_NoAuth_Returns401(t *testing.T) {
	pool := newPool(t)
	router := newRouter(pool)

	w := req(router, http.MethodGet, "/api/v1/users/me", nil)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestProfile_UpdateMe_Success(t *testing.T) {
	pool := newPool(t)
	router := newRouter(pool)
	_, token := createUser(t, pool)

	w := req(router, http.MethodPatch, "/api/v1/users/me",
		map[string]any{"public_info": map[string]any{"name": "Integration Tester"}},
		"Authorization", bearer(token),
	)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestProfile_UpdateMe_NoAuth_Returns401(t *testing.T) {
	pool := newPool(t)
	router := newRouter(pool)

	w := req(router, http.MethodPatch, "/api/v1/users/me",
		map[string]any{"public_info": map[string]any{"name": "x"}},
	)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestProfile_SearchUsers_Success(t *testing.T) {
	pool := newPool(t)
	router := newRouter(pool)
	_, token := createUser(t, pool)

	w := req(router, http.MethodGet, "/api/v1/users/search?q=integration", nil,
		"Authorization", bearer(token),
	)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestProfile_SearchUsers_NoAuth_Returns401(t *testing.T) {
	pool := newPool(t)
	router := newRouter(pool)

	w := req(router, http.MethodGet, "/api/v1/users/search?q=test", nil)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestProfile_GetUserProfile_NonFriend_NoFriendsInfo(t *testing.T) {
	pool := newPool(t)
	router := newRouter(pool)
	viewer, token := createUser(t, pool)
	target, _ := createUser(t, pool)

	_ = viewer

	w := req(router, http.MethodGet, "/api/v1/users/"+target.ID.String(), nil,
		"Authorization", bearer(token),
	)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body := jsonBody(t, w)
	if _, ok := body["friends_info"]; ok {
		t.Error("friends_info should not be present for a non-friend")
	}
}
