package integration_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
)

func TestDevices_Register_Success(t *testing.T) {
	pool := newPool(t)
	router := newRouter(pool)
	_, token := createUser(t, pool)

	w := req(router, http.MethodPost, "/api/v1/devices",
		map[string]any{"name": "My Phone", "platform": "android", "model": "Pixel 8"},
		"Authorization", bearer(token),
	)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	body := jsonBody(t, w)
	if body["id"] == nil {
		t.Error("expected id in response")
	}
}

func TestDevices_Register_NoAuth_Returns401(t *testing.T) {
	pool := newPool(t)
	router := newRouter(pool)

	w := req(router, http.MethodPost, "/api/v1/devices",
		map[string]any{"name": "My Phone", "platform": "android", "model": "Pixel 8"},
	)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDevices_List_Success(t *testing.T) {
	pool := newPool(t)
	router := newRouter(pool)
	_, token := createUser(t, pool)

	w := req(router, http.MethodGet, "/api/v1/devices", nil,
		"Authorization", bearer(token),
	)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDevices_List_NoAuth_Returns401(t *testing.T) {
	pool := newPool(t)
	router := newRouter(pool)

	w := req(router, http.MethodGet, "/api/v1/devices", nil)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDevices_Get_Owner_Success(t *testing.T) {
	pool := newPool(t)
	router := newRouter(pool)
	owner, token := createUser(t, pool)
	deviceID := createDevice(t, pool, owner.ID)

	w := req(router, http.MethodGet, "/api/v1/devices/"+deviceID.String(), nil,
		"Authorization", bearer(token),
	)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDevices_Get_NotOwner_Returns403(t *testing.T) {
	pool := newPool(t)
	router := newRouter(pool)
	owner, _ := createUser(t, pool)
	other, otherToken := createUser(t, pool)
	_ = other
	deviceID := createDevice(t, pool, owner.ID)

	w := req(router, http.MethodGet, "/api/v1/devices/"+deviceID.String(), nil,
		"Authorization", bearer(otherToken),
	)

	if w.Code != http.StatusForbidden && w.Code != http.StatusNotFound {
		t.Fatalf("expected 403/404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDevices_Get_NotFound_Returns404(t *testing.T) {
	pool := newPool(t)
	router := newRouter(pool)
	_, token := createUser(t, pool)

	w := req(router, http.MethodGet, "/api/v1/devices/"+uuid.New().String(), nil,
		"Authorization", bearer(token),
	)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDevices_Delete_Owner_Success(t *testing.T) {
	pool := newPool(t)
	router := newRouter(pool)
	owner, token := createUser(t, pool)
	deviceID := createDevice(t, pool, owner.ID)

	w := req(router, http.MethodDelete, "/api/v1/devices/"+deviceID.String(), nil,
		"Authorization", bearer(token),
	)

	if w.Code != http.StatusNoContent && w.Code != http.StatusOK {
		t.Fatalf("expected 200/204, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDevices_Delete_NotOwner_Returns403(t *testing.T) {
	pool := newPool(t)
	router := newRouter(pool)
	owner, _ := createUser(t, pool)
	other, otherToken := createUser(t, pool)
	_ = other
	deviceID := createDevice(t, pool, owner.ID)

	w := req(router, http.MethodDelete, "/api/v1/devices/"+deviceID.String(), nil,
		"Authorization", bearer(otherToken),
	)

	if w.Code != http.StatusForbidden && w.Code != http.StatusNotFound {
		t.Fatalf("expected 403/404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDevices_Delegate_Success(t *testing.T) {
	pool := newPool(t)
	router := newRouter(pool)
	owner, token := createUser(t, pool)
	friend, _ := createUser(t, pool)
	deviceID := createDevice(t, pool, owner.ID)

	// Delegate requires an accepted friendship between owner and friend
	_, err := pool.Exec(context.Background(),
		`INSERT INTO friendships (requester_id, addressee_id, status) VALUES ($1, $2, 'accepted')`,
		owner.ID, friend.ID,
	)
	if err != nil {
		t.Fatalf("seed friendship: %v", err)
	}
	t.Cleanup(func() {
		pool.Exec(context.Background(),
			`DELETE FROM friendships WHERE requester_id = $1 AND addressee_id = $2`,
			owner.ID, friend.ID,
		)
	})

	w := req(router, http.MethodPost, "/api/v1/devices/"+deviceID.String()+"/delegate",
		map[string]any{"friend_user_id": friend.ID.String()},
		"Authorization", bearer(token),
	)

	if w.Code != http.StatusCreated && w.Code != http.StatusOK {
		t.Fatalf("expected 200/201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDevices_Delegate_NotOwner_Returns403(t *testing.T) {
	pool := newPool(t)
	router := newRouter(pool)
	owner, _ := createUser(t, pool)
	other, otherToken := createUser(t, pool)
	friend, _ := createUser(t, pool)
	_ = other
	deviceID := createDevice(t, pool, owner.ID)

	w := req(router, http.MethodPost, "/api/v1/devices/"+deviceID.String()+"/delegate",
		map[string]any{"friend_user_id": friend.ID.String()},
		"Authorization", bearer(otherToken),
	)

	if w.Code != http.StatusForbidden && w.Code != http.StatusNotFound {
		t.Fatalf("expected 403/404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDevices_RevokeDelegate_Success(t *testing.T) {
	pool := newPool(t)
	router := newRouter(pool)
	owner, token := createUser(t, pool)
	delegate, _ := createUser(t, pool)
	deviceID := createDevice(t, pool, owner.ID)

	// Grant first
	req(router, http.MethodPost, "/api/v1/devices/"+deviceID.String()+"/delegate",
		map[string]any{"friend_user_id": delegate.ID.String()},
		"Authorization", bearer(token),
	)

	// Then revoke
	w := req(router, http.MethodDelete, "/api/v1/devices/"+deviceID.String()+"/delegate", nil,
		"Authorization", bearer(token),
	)

	if w.Code != http.StatusNoContent && w.Code != http.StatusOK {
		t.Fatalf("expected 200/204, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDevices_ListDelegated_Success(t *testing.T) {
	pool := newPool(t)
	router := newRouter(pool)
	_, token := createUser(t, pool)

	w := req(router, http.MethodGet, "/api/v1/devices/delegated", nil,
		"Authorization", bearer(token),
	)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}
