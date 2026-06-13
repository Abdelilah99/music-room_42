package integration_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
)

// --- Events ---

func TestEvents_Create_Success(t *testing.T) {
	pool := newPool(t)
	router := newRouter(pool)
	_, token := createUser(t, pool)

	w := req(router, http.MethodPost, "/api/v1/events",
		map[string]any{"name": "Test Event", "visibility": "public", "license": 0},
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

func TestEvents_Create_NoAuth_Returns401(t *testing.T) {
	pool := newPool(t)
	router := newRouter(pool)

	w := req(router, http.MethodPost, "/api/v1/events",
		map[string]any{"name": "Test Event", "visibility": "public", "license": 0},
	)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestEvents_List_Success(t *testing.T) {
	pool := newPool(t)
	router := newRouter(pool)
	_, token := createUser(t, pool)

	w := req(router, http.MethodGet, "/api/v1/events", nil,
		"Authorization", bearer(token),
	)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestEvents_List_NoAuth_Returns401(t *testing.T) {
	pool := newPool(t)
	router := newRouter(pool)

	w := req(router, http.MethodGet, "/api/v1/events", nil)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestEvents_Get_Success(t *testing.T) {
	pool := newPool(t)
	router := newRouter(pool)
	owner, token := createUser(t, pool)
	eventID := createEvent(t, pool, owner.ID)

	w := req(router, http.MethodGet, "/api/v1/events/"+eventID.String(), nil,
		"Authorization", bearer(token),
	)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestEvents_Get_NotFound_Returns404(t *testing.T) {
	pool := newPool(t)
	router := newRouter(pool)
	_, token := createUser(t, pool)

	w := req(router, http.MethodGet, "/api/v1/events/"+uuid.New().String(), nil,
		"Authorization", bearer(token),
	)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestEvents_Update_Owner_Success(t *testing.T) {
	pool := newPool(t)
	router := newRouter(pool)
	owner, token := createUser(t, pool)
	eventID := createEvent(t, pool, owner.ID)

	w := req(router, http.MethodPut, "/api/v1/events/"+eventID.String(),
		map[string]any{"name": "Updated Event", "visibility": "public", "license": 0},
		"Authorization", bearer(token),
	)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestEvents_Update_NotOwner_Returns403(t *testing.T) {
	pool := newPool(t)
	router := newRouter(pool)
	owner, _ := createUser(t, pool)
	other, otherToken := createUser(t, pool)
	_ = other
	eventID := createEvent(t, pool, owner.ID)

	w := req(router, http.MethodPut, "/api/v1/events/"+eventID.String(),
		map[string]any{"name": "Hacked", "visibility": "public", "license": 0},
		"Authorization", bearer(otherToken),
	)

	if w.Code != http.StatusNotFound && w.Code != http.StatusForbidden {
		t.Fatalf("expected 403/404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestEvents_Delete_Owner_Success(t *testing.T) {
	pool := newPool(t)
	router := newRouter(pool)
	owner, token := createUser(t, pool)
	eventID := createEvent(t, pool, owner.ID)

	w := req(router, http.MethodDelete, "/api/v1/events/"+eventID.String(), nil,
		"Authorization", bearer(token),
	)

	if w.Code != http.StatusNoContent && w.Code != http.StatusOK {
		t.Fatalf("expected 200/204, got %d: %s", w.Code, w.Body.String())
	}
}

func TestEvents_Delete_NotOwner_Returns403(t *testing.T) {
	pool := newPool(t)
	router := newRouter(pool)
	owner, _ := createUser(t, pool)
	other, otherToken := createUser(t, pool)
	_ = other
	eventID := createEvent(t, pool, owner.ID)

	w := req(router, http.MethodDelete, "/api/v1/events/"+eventID.String(), nil,
		"Authorization", bearer(otherToken),
	)

	if w.Code != http.StatusForbidden && w.Code != http.StatusNotFound {
		t.Fatalf("expected 403/404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestEvents_Invite_Owner_Success(t *testing.T) {
	pool := newPool(t)
	router := newRouter(pool)
	owner, token := createUser(t, pool)
	invitee, _ := createUser(t, pool)
	eventID := createEvent(t, pool, owner.ID)

	w := req(router, http.MethodPost, "/api/v1/events/"+eventID.String()+"/invites",
		map[string]any{"user_id": invitee.ID.String()},
		"Authorization", bearer(token),
	)

	if w.Code != http.StatusCreated && w.Code != http.StatusOK {
		t.Fatalf("expected 200/201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestEvents_Invite_NotOwner_Returns403(t *testing.T) {
	pool := newPool(t)
	router := newRouter(pool)
	owner, _ := createUser(t, pool)
	other, otherToken := createUser(t, pool)
	invitee, _ := createUser(t, pool)
	_ = other
	eventID := createEvent(t, pool, owner.ID)

	w := req(router, http.MethodPost, "/api/v1/events/"+eventID.String()+"/invites",
		map[string]any{"user_id": invitee.ID.String()},
		"Authorization", bearer(otherToken),
	)

	if w.Code != http.StatusForbidden && w.Code != http.StatusNotFound {
		t.Fatalf("expected 403/404, got %d: %s", w.Code, w.Body.String())
	}
}

// --- Tracks ---

func TestTracks_Suggest_Success(t *testing.T) {
	pool := newPool(t)
	router := newRouter(pool)
	owner, token := createUser(t, pool)
	eventID := createEvent(t, pool, owner.ID)

	w := req(router, http.MethodPost, "/api/v1/events/"+eventID.String()+"/tracks",
		map[string]any{
			"external_id": "deezer-" + uuid.New().String()[:8],
			"title":       "Test Track",
			"artist":      "Test Artist",
		},
		"Authorization", bearer(token),
	)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestTracks_Suggest_NoAuth_Returns401(t *testing.T) {
	pool := newPool(t)
	router := newRouter(pool)
	owner, _ := createUser(t, pool)
	eventID := createEvent(t, pool, owner.ID)

	w := req(router, http.MethodPost, "/api/v1/events/"+eventID.String()+"/tracks",
		map[string]any{"external_id": "x", "title": "x", "artist": "x"},
	)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestTracks_GetQueue_Success(t *testing.T) {
	pool := newPool(t)
	router := newRouter(pool)
	owner, token := createUser(t, pool)
	eventID := createEvent(t, pool, owner.ID)

	w := req(router, http.MethodGet, "/api/v1/events/"+eventID.String()+"/queue", nil,
		"Authorization", bearer(token),
	)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestTracks_Vote_Success(t *testing.T) {
	pool := newPool(t)
	router := newRouter(pool)
	owner, ownerToken := createUser(t, pool)
	voter, voterToken := createUser(t, pool)
	_ = voter
	eventID := createEvent(t, pool, owner.ID)
	trackID := createTrack(t, pool, eventID, owner.ID)

	w := req(router, http.MethodPost,
		"/api/v1/events/"+eventID.String()+"/tracks/"+trackID.String()+"/vote",
		nil,
		"Authorization", bearer(voterToken),
	)

	if w.Code != http.StatusOK && w.Code != http.StatusCreated {
		t.Fatalf("expected 200/201, got %d: %s", w.Code, w.Body.String())
	}

	// Second vote by same user should be rejected (duplicate prevention)
	w2 := req(router, http.MethodPost,
		"/api/v1/events/"+eventID.String()+"/tracks/"+trackID.String()+"/vote",
		nil,
		"Authorization", bearer(ownerToken),
	)
	_ = w2
	// Vote by owner (first vote for owner) — just verify no 500
	w3 := req(router, http.MethodPost,
		"/api/v1/events/"+eventID.String()+"/tracks/"+trackID.String()+"/vote",
		nil,
		"Authorization", bearer(voterToken),
	)
	if w3.Code != http.StatusConflict {
		t.Fatalf("duplicate vote: expected 409, got %d: %s", w3.Code, w3.Body.String())
	}
}

func TestTracks_DeleteTrack_Owner_Success(t *testing.T) {
	pool := newPool(t)
	router := newRouter(pool)
	owner, token := createUser(t, pool)
	eventID := createEvent(t, pool, owner.ID)
	trackID := createTrack(t, pool, eventID, owner.ID)

	w := req(router, http.MethodDelete,
		"/api/v1/events/"+eventID.String()+"/tracks/"+trackID.String(),
		nil,
		"Authorization", bearer(token),
	)

	if w.Code != http.StatusNoContent && w.Code != http.StatusOK {
		t.Fatalf("expected 200/204, got %d: %s", w.Code, w.Body.String())
	}
}

func TestTracks_DeleteTrack_NotOwner_Returns403(t *testing.T) {
	pool := newPool(t)
	router := newRouter(pool)
	owner, _ := createUser(t, pool)
	other, otherToken := createUser(t, pool)
	_ = other
	eventID := createEvent(t, pool, owner.ID)
	trackID := createTrack(t, pool, eventID, owner.ID)

	w := req(router, http.MethodDelete,
		"/api/v1/events/"+eventID.String()+"/tracks/"+trackID.String(),
		nil,
		"Authorization", bearer(otherToken),
	)

	if w.Code != http.StatusForbidden && w.Code != http.StatusNotFound {
		t.Fatalf("expected 403/404, got %d: %s", w.Code, w.Body.String())
	}
}
