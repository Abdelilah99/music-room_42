package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"music-room/internal/hub"
	"music-room/internal/model"
	"music-room/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// --- mock playlist service ---

type mockPlaylistService struct {
	getFn         func(ctx context.Context, playlistID, callerID uuid.UUID) (*model.PlaylistWithTracks, error)
	addTrackFn    func(ctx context.Context, playlistID, callerID uuid.UUID, req model.AddTrackRequest) (*model.PlaylistTrack, error)
	removeTrackFn func(ctx context.Context, playlistID, callerID, trackID uuid.UUID) error
	moveTrackFn   func(ctx context.Context, playlistID, callerID, trackID uuid.UUID, newPos int) error
}

func (m *mockPlaylistService) Create(ctx context.Context, ownerID uuid.UUID, req model.CreatePlaylistRequest) (*model.Playlist, error) {
	return nil, nil
}
func (m *mockPlaylistService) List(ctx context.Context, callerID uuid.UUID, f model.PlaylistListFilter) ([]model.Playlist, error) {
	return nil, nil
}
func (m *mockPlaylistService) Get(ctx context.Context, playlistID, callerID uuid.UUID) (*model.PlaylistWithTracks, error) {
	if m.getFn != nil {
		return m.getFn(ctx, playlistID, callerID)
	}
	return &model.PlaylistWithTracks{Playlist: model.Playlist{ID: playlistID}, Tracks: []model.PlaylistTrack{}}, nil
}
func (m *mockPlaylistService) Update(ctx context.Context, playlistID, callerID uuid.UUID, req model.UpdatePlaylistRequest) (*model.Playlist, error) {
	return nil, nil
}
func (m *mockPlaylistService) Delete(ctx context.Context, playlistID, callerID uuid.UUID) error {
	return nil
}
func (m *mockPlaylistService) Invite(ctx context.Context, playlistID, callerID, targetUserID uuid.UUID) error {
	return nil
}
func (m *mockPlaylistService) AddTrack(ctx context.Context, playlistID, callerID uuid.UUID, req model.AddTrackRequest) (*model.PlaylistTrack, error) {
	return m.addTrackFn(ctx, playlistID, callerID, req)
}
func (m *mockPlaylistService) RemoveTrack(ctx context.Context, playlistID, callerID, trackID uuid.UUID) error {
	return m.removeTrackFn(ctx, playlistID, callerID, trackID)
}
func (m *mockPlaylistService) MoveTrack(ctx context.Context, playlistID, callerID, trackID uuid.UUID, newPos int) error {
	return m.moveTrackFn(ctx, playlistID, callerID, trackID, newPos)
}

// ensure mockPlaylistService satisfies the interface at compile time
var _ service.PlaylistService = (*mockPlaylistService)(nil)

// --- test helpers ---

func setupPlaylistRouter(svc service.PlaylistService, manager *hub.HubManager) (*gin.Engine, uuid.UUID) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewPlaylistHandler(svc, manager)

	callerID := uuid.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", callerID.String())
		c.Next()
	})

	r.POST("/api/v1/playlists/:id/tracks", h.AddTrack)
	r.DELETE("/api/v1/playlists/:id/tracks/:trackId", h.RemoveTrack)
	r.PATCH("/api/v1/playlists/:id/tracks/:trackId/position", h.MoveTrack)
	r.GET("/api/v1/playlists/:id/ws", h.ServeWS)
	return r, callerID
}

// dialPlaylistWS connects a WebSocket client to the playlist hub via the test server.
func dialPlaylistWS(t *testing.T, srv *httptest.Server, playlistID uuid.UUID) *websocket.Conn {
	t.Helper()
	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/api/v1/playlists/" + playlistID.String() + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("dial ws: %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	return conn
}

func recvPlaylistMsg(t *testing.T, conn *websocket.Conn) map[string]any {
	t.Helper()
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, raw, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read ws message: %v", err)
	}
	var msg map[string]any
	if err := json.Unmarshal(raw, &msg); err != nil {
		t.Fatalf("unmarshal ws message: %v", err)
	}
	return msg
}

// --- broadcast tests ---

func TestPlaylistHandler_AddTrack_BroadcastsTrackAdded(t *testing.T) {
	playlistID := uuid.New()
	trackID := uuid.New()

	svc := &mockPlaylistService{
		addTrackFn: func(_ context.Context, _ uuid.UUID, _ uuid.UUID, req model.AddTrackRequest) (*model.PlaylistTrack, error) {
			return &model.PlaylistTrack{
				ID:         trackID,
				PlaylistID: playlistID,
				Title:      req.Title,
				Artist:     req.Artist,
				Position:   1,
			}, nil
		},
	}

	manager := hub.NewHubManager()
	r, _ := setupPlaylistRouter(svc, manager)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	ws := dialPlaylistWS(t, srv, playlistID)

	body := `{"external_id":"1","title":"Creep","artist":"Radiohead"}`
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/v1/playlists/"+playlistID.String()+"/tracks", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST tracks: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}

	msg := recvPlaylistMsg(t, ws)
	if msg["type"] != "track_added" {
		t.Errorf("expected type=track_added, got %v", msg["type"])
	}
	track, ok := msg["track"].(map[string]any)
	if !ok {
		t.Fatalf("expected track object, got %T", msg["track"])
	}
	if track["id"] != trackID.String() {
		t.Errorf("expected track id %s, got %v", trackID, track["id"])
	}
}

func TestPlaylistHandler_RemoveTrack_BroadcastsTrackRemoved(t *testing.T) {
	playlistID := uuid.New()
	trackID := uuid.New()

	svc := &mockPlaylistService{
		removeTrackFn: func(_ context.Context, _, _, _ uuid.UUID) error { return nil },
	}

	manager := hub.NewHubManager()
	r, _ := setupPlaylistRouter(svc, manager)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	ws := dialPlaylistWS(t, srv, playlistID)

	req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/api/v1/playlists/"+playlistID.String()+"/tracks/"+trackID.String(), nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("DELETE track: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	msg := recvPlaylistMsg(t, ws)
	if msg["type"] != "track_removed" {
		t.Errorf("expected type=track_removed, got %v", msg["type"])
	}
	if msg["track_id"] != trackID.String() {
		t.Errorf("expected track_id %s, got %v", trackID, msg["track_id"])
	}
}

func TestPlaylistHandler_MoveTrack_BroadcastsTrackMoved(t *testing.T) {
	playlistID := uuid.New()
	trackID := uuid.New()

	svc := &mockPlaylistService{
		moveTrackFn: func(_ context.Context, _, _, _ uuid.UUID, _ int) error { return nil },
	}

	manager := hub.NewHubManager()
	r, _ := setupPlaylistRouter(svc, manager)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	ws := dialPlaylistWS(t, srv, playlistID)

	body := `{"position":3}`
	req, _ := http.NewRequest(http.MethodPatch, srv.URL+"/api/v1/playlists/"+playlistID.String()+"/tracks/"+trackID.String()+"/position", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("PATCH position: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	msg := recvPlaylistMsg(t, ws)
	if msg["type"] != "track_moved" {
		t.Errorf("expected type=track_moved, got %v", msg["type"])
	}
	if msg["track_id"] != trackID.String() {
		t.Errorf("expected track_id %s, got %v", trackID, msg["track_id"])
	}
	if int(msg["position"].(float64)) != 3 {
		t.Errorf("expected position=3, got %v", msg["position"])
	}
}

func TestPlaylistHandler_NoBroadcast_WhenNoClientsConnected(t *testing.T) {
	playlistID := uuid.New()
	broadcastCalled := false

	svc := &mockPlaylistService{
		removeTrackFn: func(_ context.Context, _, _, _ uuid.UUID) error { return nil },
	}

	manager := hub.NewHubManager()
	r, _ := setupPlaylistRouter(svc, manager)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	// No WebSocket client connected - hub should not exist.
	req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/api/v1/playlists/"+playlistID.String()+"/tracks/"+uuid.New().String(), nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("DELETE track: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	// Verify no hub was created for this playlist.
	if manager.HasHub(playlistID.String()) {
		broadcastCalled = true
	}
	if broadcastCalled {
		t.Error("expected no hub to exist when no clients are connected")
	}
}

func TestPlaylistHandler_ServeWS_InaccessiblePlaylist_Rejects(t *testing.T) {
	playlistID := uuid.New()

	svc := &mockPlaylistService{
		getFn: func(_ context.Context, _, _ uuid.UUID) (*model.PlaylistWithTracks, error) {
			return nil, service.ErrPlaylistNotFound
		},
	}

	manager := hub.NewHubManager()
	r, _ := setupPlaylistRouter(svc, manager)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/api/v1/playlists/" + playlistID.String() + "/ws"
	_, resp, err := websocket.DefaultDialer.Dial(url, nil)
	if err == nil {
		t.Fatal("expected dial to fail for inaccessible playlist")
	}
	if resp != nil && resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}
