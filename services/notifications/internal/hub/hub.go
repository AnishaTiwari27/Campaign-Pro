// Package hub holds a small in-memory WebSocket fan-out: broadcast a
// message to every currently-connected client. Each client gets its own
// buffered outbound channel and writer goroutine, so one slow or dead
// connection can never block a broadcast to the others — it just drops
// that one client's message and moves on.
package hub

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/coder/websocket"
)

const clientSendBuffer = 8 // messages queued per client before we start dropping for it

type client struct {
	conn *websocket.Conn
	send chan []byte
}

type Hub struct {
	mu      sync.Mutex
	clients map[*client]bool
}

func New() *Hub {
	return &Hub{clients: map[*client]bool{}}
}

// Connections reports how many clients are currently connected — surfaced
// on /healthz so it's visible whether the dashboard is actually listening.
func (h *Hub) Connections() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.clients)
}

// Serve registers conn, relays outbound messages to it until it
// disconnects, and unregisters it on the way out. Blocks for the
// connection's lifetime — call it in its own goroutine per connection.
func (h *Hub) Serve(conn *websocket.Conn) {
	c := &client{conn: conn, send: make(chan []byte, clientSendBuffer)}

	h.mu.Lock()
	h.clients[c] = true
	h.mu.Unlock()

	defer func() {
		h.mu.Lock()
		delete(h.clients, c)
		h.mu.Unlock()
	}()

	// A read loop that discards anything the client sends — this hub is
	// server-push only — but is the only way to notice the client closed
	// the connection (Read returns an error) so we can stop writing to it.
	closed := make(chan struct{})
	go func() {
		defer close(closed)
		for {
			if _, _, err := conn.Read(context.Background()); err != nil {
				return
			}
		}
	}()

	for {
		select {
		case msg := <-c.send:
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			err := conn.Write(ctx, websocket.MessageText, msg)
			cancel()
			if err != nil {
				return
			}
		case <-closed:
			return
		}
	}
}

// Broadcast fans payload out to every connected client, non-blocking —
// a client whose send buffer is already full has its message dropped
// rather than stalling the broadcast for everyone else.
func (h *Hub) Broadcast(payload []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for c := range h.clients {
		select {
		case c.send <- payload:
		default:
			slog.Warn("hub: client send buffer full, dropping message")
		}
	}
}
