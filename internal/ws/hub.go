package ws

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// TokenValidator is set by main.go. Given an HTTP request, it returns the
// authenticated user id or an error. When nil, ServeWS rejects every upgrade.
type TokenValidator func(r *http.Request) (uuid.UUID, error)

// AllowedOrigins is the set of values the upgrader's CheckOrigin accepts
// verbatim. An empty set falls back to same-host matching.
type AllowedOrigins map[string]bool

type Hub struct {
	mu       sync.RWMutex
	rooms    map[string]map[*Client]bool
	ops      chan roomOp
	publish  chan Event
	validate TokenValidator
	origins  AllowedOrigins
	log      *slog.Logger
}

// roomOp discriminates join/leave so both share a single FIFO channel — that's
// the only way to guarantee order preservation for interleaved subscribe events
// from the same client. When join/leave lived on separate channels, Go select
// could reorder them and strand a client outside a room (dev-mode StrictMode
// double-mount was the reliable repro).
type roomOp struct {
	kind   byte // 'J' | 'L'
	room   string
	client *Client
}

func NewHub() *Hub {
	return &Hub{
		rooms:   make(map[string]map[*Client]bool),
		ops:     make(chan roomOp, 256),
		publish: make(chan Event, 256),
	}
}

// SetAuth wires the JWT validator and the allow-list of origins. Call once
// during bootstrap; the guard applies to every subsequent Upgrade.
func (h *Hub) SetAuth(v TokenValidator, origins AllowedOrigins) {
	h.validate = v
	h.origins = origins
}

// SetLogger installs the slog logger used for diagnostic lines. Nil is safe —
// all hub logging calls fan through logIf which no-ops on nil.
func (h *Hub) SetLogger(l *slog.Logger) { h.log = l }

func (h *Hub) logIf(level slog.Level, msg string, args ...any) {
	if h.log == nil {
		return
	}
	h.log.Log(nil, level, msg, args...)
}

func (h *Hub) Run() {
	for {
		select {
		case op := <-h.ops:
			h.mu.Lock()
			switch op.kind {
			case 'J':
				if h.rooms[op.room] == nil {
					h.rooms[op.room] = make(map[*Client]bool)
				}
				h.rooms[op.room][op.client] = true
				n := len(h.rooms[op.room])
				h.mu.Unlock()
				h.logIf(slog.LevelDebug, "ws.join", "room", op.room, "uid", op.client.userID, "size", n)
			case 'L':
				delete(h.rooms[op.room], op.client)
				n := len(h.rooms[op.room])
				h.mu.Unlock()
				h.logIf(slog.LevelDebug, "ws.leave", "room", op.room, "uid", op.client.userID, "size", n)
			default:
				h.mu.Unlock()
			}
		case e := <-h.publish:
			h.mu.RLock()
			subs := h.rooms[e.Room]
			delivered, dropped := 0, 0
			for c := range subs {
				select {
				case c.send <- e:
					delivered++
				default:
					dropped++
				}
			}
			h.mu.RUnlock()
			if delivered+dropped == 0 {
				h.logIf(slog.LevelInfo, "ws.publish.no_subscribers", "room", e.Room, "type", e.Type)
			} else {
				h.logIf(slog.LevelDebug, "ws.publish", "room", e.Room, "type", e.Type, "delivered", delivered, "dropped", dropped)
			}
		}
	}
}

// join / leave are the internal API clients use to manipulate room membership.
// Both enqueue onto the same channel so ordering is preserved.
func (h *Hub) join(room string, c *Client)  { h.ops <- roomOp{kind: 'J', room: room, client: c} }
func (h *Hub) leave(room string, c *Client) { h.ops <- roomOp{kind: 'L', room: room, client: c} }

func (h *Hub) checkOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true // non-browser clients
	}
	if len(h.origins) > 0 && h.origins[origin] {
		return true
	}
	if i := indexOf(origin, "://"); i >= 0 && origin[i+3:] == r.Host {
		return true
	}
	return false
}

func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	if h.validate == nil {
		http.Error(w, "ws auth not configured", http.StatusServiceUnavailable)
		return
	}
	uid, err := h.validate(r)
	if err != nil {
		h.logIf(slog.LevelInfo, "ws.auth.reject", "origin", r.Header.Get("Origin"), "err", err.Error())
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	upgrader := websocket.Upgrader{CheckOrigin: h.checkOrigin}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logIf(slog.LevelWarn, "ws.upgrade.failed", "origin", r.Header.Get("Origin"), "err", err.Error())
		return
	}
	h.logIf(slog.LevelInfo, "ws.connect", "uid", uid, "origin", r.Header.Get("Origin"))
	client := &Client{
		hub:    h,
		conn:   conn,
		send:   make(chan Event, 32),
		userID: uid,
		rooms:  map[string]bool{},
	}
	go client.writePump()
	go client.readPump()

	// Auto-subscribe to the user's personal room so server-initiated toast
	// notifications reach them without an explicit `join` handshake.
	personal := RoomUser(uid.String())
	client.rooms[personal] = true
	h.join(personal, client)

	// Hello frame so the client can confirm auth + its server-known id.
	payload, _ := json.Marshal(map[string]string{"user_id": uid.String()})
	select {
	case client.send <- Event{
		V:       1,
		Type:    "ws.hello",
		ActorID: uid.String(),
		TS:      time.Now().UTC(),
		Payload: payload,
	}:
	default:
	}
}

func (h *Hub) Publish(e Event) { h.publish <- e }

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
