package hub

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const sendBufferSize = 256

const (
	// writeWait is the maximum time allowed to write a message to a client.
	writeWait = 10 * time.Second
	// pongWait is how long we wait for a pong reply before treating the
	// connection as dead.
	pongWait = 60 * time.Second
	// pingPeriod is how often we send a ping. Must be shorter than pongWait.
	pingPeriod = (pongWait * 9) / 10
)

// Client is a single WebSocket connection attached to a Hub.
type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte
}

// Hub maintains a set of active clients and routes messages safely using
// channels so no goroutine ever touches the clients map directly.
type Hub struct {
	clients    map[*Client]struct{}
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	stopped    chan struct{} // closed when the last client leaves
	stopOnce   sync.Once
}

func newHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]struct{}),
		broadcast:  make(chan []byte, sendBufferSize),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		stopped:    make(chan struct{}),
	}
}

// Broadcast enqueues msg for delivery to every connected client.
// Safe to call from any goroutine.
func (h *Hub) Broadcast(msg []byte) {
	select {
	case h.broadcast <- msg:
	case <-h.stopped:
	}
}

// tryRegister sends the client to the hub's register channel.
// Returns false if the hub has already stopped (all clients left before
// this one could connect), so the caller can retry with a fresh hub.
func (h *Hub) tryRegister(c *Client) bool {
	select {
	case h.register <- c:
		return true
	case <-h.stopped:
		return false
	}
}

// run is the single goroutine that owns the clients map.
// onEmpty is called (from this goroutine) when the last client leaves.
func (h *Hub) run(onEmpty func()) {
	hadClients := false
	for {
		select {
		case c := <-h.register:
			hadClients = true
			h.clients[c] = struct{}{}

		case c := <-h.unregister:
			if _, ok := h.clients[c]; ok {
				delete(h.clients, c)
				close(c.send)
			}
			if hadClients && len(h.clients) == 0 {
				h.stopOnce.Do(func() { close(h.stopped) })
				if onEmpty != nil {
					onEmpty()
				}
				return
			}

		case msg := <-h.broadcast:
			for c := range h.clients {
				select {
				case c.send <- msg:
				default:
					// Slow client: drop and disconnect.
					close(c.send)
					delete(h.clients, c)
				}
			}
			// A broadcast may have evicted slow clients, check for empty hub.
			if hadClients && len(h.clients) == 0 {
				h.stopOnce.Do(func() { close(h.stopped) })
				if onEmpty != nil {
					onEmpty()
				}
				return
			}
		}
	}
}

// writePump pumps messages from the client's send channel to the WebSocket
// and sends periodic pings to keep idle connections alive.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case msg, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub closed the channel: tell the client we are done.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// readPump reads from the WebSocket and deregisters the client on close.
// Incoming messages are forwarded to the hub for broadcast.
func (c *Client) readPump() {
	defer func() {
		// Use select so we don't block if the hub already stopped (e.g. it
		// removed this client via the slow-client path and then shut down).
		select {
		case c.hub.unregister <- c:
		case <-c.hub.stopped:
		}
		c.conn.Close()
	}()
	// Reset the read deadline on every pong so a live connection stays open.
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	for {
		_, msg, err := c.conn.ReadMessage()
		if err != nil {
			return
		}
		c.hub.Broadcast(msg)
	}
}
