package hub

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// dialTestServer spins up a test HTTP server using ServeWS and returns a
// connected WebSocket client.
func dialTestServer(t *testing.T, manager *HubManager, entityID string) *websocket.Conn {
	t.Helper()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/ws/:id", func(c *gin.Context) {
		ServeWS(manager, c.Param("id"), c)
	})

	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws/" + entityID
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	return conn
}

func recv(t *testing.T, conn *websocket.Conn, timeout time.Duration) []byte {
	t.Helper()
	conn.SetReadDeadline(time.Now().Add(timeout))
	_, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	return msg
}

// TestBroadcastReachesAllClients verifies that a message sent to the hub
// arrives at every connected client.
func TestBroadcastReachesAllClients(t *testing.T) {
	manager := NewHubManager()

	c1 := dialTestServer(t, manager, "room-1")
	c2 := dialTestServer(t, manager, "room-1")

	// Let registrations propagate.
	time.Sleep(50 * time.Millisecond)

	want := []byte("hello everyone")

	// Trigger broadcast by having c1 send a message (readPump forwards it).
	if err := c1.WriteMessage(websocket.TextMessage, want); err != nil {
		t.Fatalf("write: %v", err)
	}

	// Both c1 and c2 should receive the broadcast.
	got1 := recv(t, c1, time.Second)
	got2 := recv(t, c2, time.Second)

	if string(got1) != string(want) {
		t.Errorf("c1 got %q, want %q", got1, want)
	}
	if string(got2) != string(want) {
		t.Errorf("c2 got %q, want %q", got2, want)
	}
}

// TestDisconnectDoesNotAffectOtherClients verifies that closing one client
// does not crash the hub and other clients keep receiving messages.
func TestDisconnectDoesNotAffectOtherClients(t *testing.T) {
	manager := NewHubManager()

	c1 := dialTestServer(t, manager, "room-2")
	c2 := dialTestServer(t, manager, "room-2")

	time.Sleep(50 * time.Millisecond)

	// Disconnect c1 abruptly.
	c1.Close()
	time.Sleep(50 * time.Millisecond)

	// c2 should still receive broadcasts.
	want := []byte("still here")
	if err := c2.WriteMessage(websocket.TextMessage, want); err != nil {
		t.Fatalf("write: %v", err)
	}

	got := recv(t, c2, time.Second)
	if string(got) != string(want) {
		t.Errorf("c2 got %q, want %q", got, want)
	}
}

// TestHubCleansUpWhenEmpty verifies that the HubManager removes the hub entry
// once all clients disconnect, preventing a memory leak.
func TestHubCleansUpWhenEmpty(t *testing.T) {
	manager := NewHubManager()

	c1 := dialTestServer(t, manager, "room-3")

	time.Sleep(50 * time.Millisecond)

	if manager.Len() != 1 {
		t.Fatalf("expected 1 hub, got %d", manager.Len())
	}

	c1.Close()

	// Give the hub time to detect the disconnect and remove itself.
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if manager.Len() == 0 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("hub was not cleaned up after all clients disconnected; manager.Len()=%d", manager.Len())
}

// TestSlowClientEvictionCleansUpHub verifies that when the hub drops a slow
// client (full send buffer), the hub and manager still clean up correctly and
// no goroutine hangs.
//
// We test this at the hub level directly (no real WebSocket) because a real
// writePump drains c.send into the OS TCP buffer, making the channel appear
// never-full from the hub's perspective.
func TestSlowClientEvictionCleansUpHub(t *testing.T) {
	manager := NewHubManager()
	h := manager.GetOrCreate("slow-room")

	// Unbuffered send channel: any broadcast immediately triggers the
	// slow-client default branch and evicts this client.
	slowClient := &Client{
		hub:  h,
		send: make(chan []byte), // capacity 0
	}

	// Register directly via the hub's channel (same package access).
	h.register <- slowClient
	time.Sleep(10 * time.Millisecond)

	// One broadcast is enough to evict the slow client and empty the hub.
	h.Broadcast([]byte("trigger"))

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if manager.Len() == 0 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("hub not cleaned up after slow-client eviction; manager.Len()=%d", manager.Len())
}

// TestClientsInDifferentHubsAreIsolated verifies that two entity IDs produce
// separate hubs and messages do not leak between rooms.
func TestClientsInDifferentHubsAreIsolated(t *testing.T) {
	manager := NewHubManager()

	c1 := dialTestServer(t, manager, "room-a")
	c2 := dialTestServer(t, manager, "room-b")

	time.Sleep(50 * time.Millisecond)

	if manager.Len() != 2 {
		t.Fatalf("expected 2 hubs, got %d", manager.Len())
	}

	// c1 sends a message — only c1's hub broadcasts it.
	if err := c1.WriteMessage(websocket.TextMessage, []byte("room-a only")); err != nil {
		t.Fatalf("write: %v", err)
	}

	// c1 should receive its own broadcast.
	got := recv(t, c1, time.Second)
	if string(got) != "room-a only" {
		t.Errorf("c1 got %q", got)
	}

	// c2 must NOT receive the message — set a short deadline.
	c2.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
	_, _, err := c2.ReadMessage()
	if err == nil {
		t.Error("c2 received a message from a different hub")
	}
}
