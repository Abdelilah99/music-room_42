package integration_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
)

func TestPlaylists_Create_Success(t *testing.T) {
	pool := newPool(t)
	router := newRouter(pool)
	_, token := createUser(t, pool)

	w := req(router, http.MethodPost, "/api/v1/playlists",
		map[string]any{"name": "My Playlist", "visibility": "public", "license": 0},
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

func TestPlaylists_Create_NoAuth_Returns401(t *testing.T) {
	pool := newPool(t)
	router := newRouter(pool)

	w := req(router, http.MethodPost, "/api/v1/playlists",
		map[string]any{"name": "My Playlist", "visibility": "public", "license": 0},
	)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPlaylists_List_Success(t *testing.T) {
	pool := newPool(t)
	router := newRouter(pool)
	_, token := createUser(t, pool)

	w := req(router, http.MethodGet, "/api/v1/playlists", nil,
		"Authorization", bearer(token),
	)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPlaylists_List_NoAuth_Returns401(t *testing.T) {
	pool := newPool(t)
	router := newRouter(pool)

	w := req(router, http.MethodGet, "/api/v1/playlists", nil)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPlaylists_Get_Success(t *testing.T) {
	pool := newPool(t)
	router := newRouter(pool)
	owner, token := createUser(t, pool)
	playlistID := createPlaylist(t, pool, owner.ID)

	w := req(router, http.MethodGet, "/api/v1/playlists/"+playlistID.String(), nil,
		"Authorization", bearer(token),
	)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPlaylists_Get_NotFound_Returns404(t *testing.T) {
	pool := newPool(t)
	router := newRouter(pool)
	_, token := createUser(t, pool)

	w := req(router, http.MethodGet, "/api/v1/playlists/"+uuid.New().String(), nil,
		"Authorization", bearer(token),
	)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPlaylists_Update_Owner_Success(t *testing.T) {
	pool := newPool(t)
	router := newRouter(pool)
	owner, token := createUser(t, pool)
	playlistID := createPlaylist(t, pool, owner.ID)

	w := req(router, http.MethodPut, "/api/v1/playlists/"+playlistID.String(),
		map[string]any{"name": "Updated Playlist", "visibility": "public", "license": 0},
		"Authorization", bearer(token),
	)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPlaylists_Update_NotOwner_Returns403(t *testing.T) {
	pool := newPool(t)
	router := newRouter(pool)
	owner, _ := createUser(t, pool)
	other, otherToken := createUser(t, pool)
	_ = other
	playlistID := createPlaylist(t, pool, owner.ID)

	w := req(router, http.MethodPut, "/api/v1/playlists/"+playlistID.String(),
		map[string]any{"name": "Hacked", "visibility": "public", "license": 0},
		"Authorization", bearer(otherToken),
	)

	if w.Code != http.StatusForbidden && w.Code != http.StatusNotFound {
		t.Fatalf("expected 403/404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPlaylists_Delete_Owner_Success(t *testing.T) {
	pool := newPool(t)
	router := newRouter(pool)
	owner, token := createUser(t, pool)
	playlistID := createPlaylist(t, pool, owner.ID)

	w := req(router, http.MethodDelete, "/api/v1/playlists/"+playlistID.String(), nil,
		"Authorization", bearer(token),
	)

	if w.Code != http.StatusNoContent && w.Code != http.StatusOK {
		t.Fatalf("expected 200/204, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPlaylists_Delete_NotOwner_Returns403(t *testing.T) {
	pool := newPool(t)
	router := newRouter(pool)
	owner, _ := createUser(t, pool)
	other, otherToken := createUser(t, pool)
	_ = other
	playlistID := createPlaylist(t, pool, owner.ID)

	w := req(router, http.MethodDelete, "/api/v1/playlists/"+playlistID.String(), nil,
		"Authorization", bearer(otherToken),
	)

	if w.Code != http.StatusForbidden && w.Code != http.StatusNotFound {
		t.Fatalf("expected 403/404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPlaylists_Invite_Owner_Success(t *testing.T) {
	pool := newPool(t)
	router := newRouter(pool)
	owner, token := createUser(t, pool)
	invitee, _ := createUser(t, pool)
	playlistID := createPlaylist(t, pool, owner.ID)

	w := req(router, http.MethodPost, "/api/v1/playlists/"+playlistID.String()+"/invites",
		map[string]any{"user_id": invitee.ID.String()},
		"Authorization", bearer(token),
	)

	if w.Code != http.StatusCreated && w.Code != http.StatusOK {
		t.Fatalf("expected 200/201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPlaylists_Invite_NotOwner_Returns403(t *testing.T) {
	pool := newPool(t)
	router := newRouter(pool)
	owner, _ := createUser(t, pool)
	other, otherToken := createUser(t, pool)
	invitee, _ := createUser(t, pool)
	_ = other
	playlistID := createPlaylist(t, pool, owner.ID)

	w := req(router, http.MethodPost, "/api/v1/playlists/"+playlistID.String()+"/invites",
		map[string]any{"user_id": invitee.ID.String()},
		"Authorization", bearer(otherToken),
	)

	if w.Code != http.StatusForbidden && w.Code != http.StatusNotFound {
		t.Fatalf("expected 403/404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPlaylists_AddTrack_Success(t *testing.T) {
	pool := newPool(t)
	router := newRouter(pool)
	owner, token := createUser(t, pool)
	playlistID := createPlaylist(t, pool, owner.ID)

	w := req(router, http.MethodPost, "/api/v1/playlists/"+playlistID.String()+"/tracks",
		map[string]any{
			"external_id": "deezer-" + uuid.New().String()[:8],
			"title":       "Test Track",
			"artist":      "Test Artist",
		},
		"Authorization", bearer(token),
	)

	if w.Code != http.StatusCreated && w.Code != http.StatusOK {
		t.Fatalf("expected 200/201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPlaylists_AddTrack_NoAuth_Returns401(t *testing.T) {
	pool := newPool(t)
	router := newRouter(pool)
	owner, _ := createUser(t, pool)
	playlistID := createPlaylist(t, pool, owner.ID)

	w := req(router, http.MethodPost, "/api/v1/playlists/"+playlistID.String()+"/tracks",
		map[string]any{"external_id": "x", "title": "x", "artist": "x"},
	)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPlaylists_RemoveTrack_Owner_Success(t *testing.T) {
	pool := newPool(t)
	router := newRouter(pool)
	owner, token := createUser(t, pool)
	playlistID := createPlaylist(t, pool, owner.ID)
	trackID := addPlaylistTrack(t, pool, playlistID, owner.ID)

	w := req(router, http.MethodDelete,
		"/api/v1/playlists/"+playlistID.String()+"/tracks/"+trackID.String(),
		nil,
		"Authorization", bearer(token),
	)

	if w.Code != http.StatusNoContent && w.Code != http.StatusOK {
		t.Fatalf("expected 200/204, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPlaylists_RemoveTrack_NotOwner_Returns403(t *testing.T) {
	pool := newPool(t)
	router := newRouter(pool)
	owner, _ := createUser(t, pool)
	other, otherToken := createUser(t, pool)
	_ = other

	// license=1 restricts editing to owner and invited users only
	var playlistID uuid.UUID
	err := pool.QueryRow(context.Background(),
		`INSERT INTO playlists (owner_id, name, visibility, license) VALUES ($1, $2, 'public', 1) RETURNING id`,
		owner.ID, "Restricted Playlist",
	).Scan(&playlistID)
	if err != nil {
		t.Fatalf("insert restricted playlist: %v", err)
	}
	t.Cleanup(func() {
		pool.Exec(context.Background(), `DELETE FROM playlists WHERE id = $1`, playlistID)
	})

	trackID := addPlaylistTrack(t, pool, playlistID, owner.ID)

	w := req(router, http.MethodDelete,
		"/api/v1/playlists/"+playlistID.String()+"/tracks/"+trackID.String(),
		nil,
		"Authorization", bearer(otherToken),
	)

	if w.Code != http.StatusForbidden && w.Code != http.StatusNotFound {
		t.Fatalf("expected 403/404, got %d: %s", w.Code, w.Body.String())
	}
}
