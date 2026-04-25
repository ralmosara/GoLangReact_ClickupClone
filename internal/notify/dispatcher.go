package notify

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/yourorg/clickup/internal/domain"
	"github.com/yourorg/clickup/internal/ws"
)

// Dispatcher persists in-app notifications and fans them out over WS.
// M1 is in-app only; email/push plug in at M8 by wrapping Enqueue.
type Dispatcher struct {
	repo domain.NotificationRepo
	hub  *ws.Hub
	log  *slog.Logger
	in   chan envelope
}

type envelope struct {
	n       *domain.Notification
	actorID *uuid.UUID
}

type Notice struct {
	Recipient  uuid.UUID
	Actor      *uuid.UUID
	Kind       string
	EntityType string
	EntityID   *uuid.UUID
	Payload    any
}

func New(repo domain.NotificationRepo, hub *ws.Hub, log *slog.Logger) *Dispatcher {
	d := &Dispatcher{
		repo: repo,
		hub:  hub,
		log:  log,
		in:   make(chan envelope, 256),
	}
	go d.run()
	return d
}

func (d *Dispatcher) Enqueue(n Notice) {
	payload, _ := json.Marshal(n.Payload)
	note := &domain.Notification{
		UserID:     n.Recipient,
		Kind:       n.Kind,
		EntityType: n.EntityType,
		EntityID:   n.EntityID,
		ActorID:    n.Actor,
		Payload:    payload,
	}
	select {
	case d.in <- envelope{n: note, actorID: n.Actor}:
	default:
		// Channel is full — persist synchronously rather than drop.
		d.persistAndPublish(context.Background(), envelope{n: note, actorID: n.Actor})
	}
}

func (d *Dispatcher) run() {
	for env := range d.in {
		d.persistAndPublish(context.Background(), env)
	}
}

func (d *Dispatcher) persistAndPublish(ctx context.Context, env envelope) {
	if err := d.repo.Create(ctx, env.n); err != nil {
		if d.log != nil {
			d.log.Error("notify persist failed", "err", err, "kind", env.n.Kind)
		}
		return
	}
	if d.hub == nil {
		return
	}
	payload, _ := json.Marshal(env.n)
	e := ws.Event{
		V:        1,
		Room:     ws.RoomUser(env.n.UserID.String()),
		Type:     ws.EventNotificationCreated,
		EntityID: env.n.ID.String(),
		TS:       time.Now().UTC(),
		Payload:  payload,
	}
	if env.actorID != nil {
		e.ActorID = env.actorID.String()
	}
	d.hub.Publish(e)
}
