package hub

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// CheckOrigin allows all origins. Production services should restrict this.
	CheckOrigin: func(r *http.Request) bool { return true },
}

// ServeWS upgrades the HTTP connection to WebSocket and registers the client
// with the hub identified by entityID. It retries once if the hub was cleaned
// up between lookup and registration.
func ServeWS(manager *HubManager, entityID string, c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		slog.Error("websocket upgrade error", "error", err)
		return
	}

	client := &Client{
		conn: conn,
		send: make(chan []byte, sendBufferSize),
	}

	// Retry once in case the hub was cleaned up between GetOrCreate and tryRegister.
	for range 2 {
		h := manager.GetOrCreate(entityID)
		client.hub = h
		if h.tryRegister(client) {
			go client.writePump()
			go client.readPump()
			return
		}
	}

	// Both attempts failed — hub keeps stopping before we can join.
	conn.Close()
}
