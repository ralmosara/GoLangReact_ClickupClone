// Package automation is the M4 rule engine. It listens for task events
// (create/update/status-change/assignment/completion) and dispatches the
// matching automations' actions asynchronously so business transactions
// stay snappy.
//
// The engine is intentionally non-transactional: an action failure is
// logged, not retried. A proper outbox + retry worker lands in M8 once we
// have persistent queues.
package automation

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yourorg/clickup/internal/domain"
)

// Actions is the set of side effects an automation can perform. main.go wires
// these to concrete service methods so this package has no direct dependency
// on task/status/tag/comment services (avoiding import cycles).
type Actions struct {
	ChangeStatus func(ctx context.Context, actor, taskID uuid.UUID, statusID uuid.UUID) error
	AssignUser   func(ctx context.Context, actor, taskID, userID uuid.UUID) error
	AddTag       func(ctx context.Context, actor, taskID, tagID uuid.UUID) error
	AddComment   func(ctx context.Context, author, taskID uuid.UUID, body string) error
	Notify       func(ctx context.Context, actor, userID uuid.UUID, kind string, payload map[string]any)
}

type Engine struct {
	repo    domain.AutomationRepo
	actions Actions
	pool    *pgxpool.Pool
	log     *slog.Logger
	queue   chan Event
}

// Event is produced by the task service on any interesting mutation.
type Event struct {
	Type    string
	Task    *domain.Task
	Before  *domain.Task
	ActorID uuid.UUID
}

type triggerCfg struct {
	Type       string     `json:"type"`
	ToStatusID *uuid.UUID `json:"to_status_id,omitempty"`
	UserID     *uuid.UUID `json:"user_id,omitempty"`
	LeadDays   *int       `json:"lead_days,omitempty"`
}

type conditionCfg struct {
	Field string      `json:"field"` // priority | status_id | assignee_id
	Op    string      `json:"op"`    // eq | neq | gte | lte | in
	Value interface{} `json:"value"`
}

type actionCfg struct {
	Type     string          `json:"type"`
	StatusID *uuid.UUID      `json:"status_id,omitempty"`
	UserID   *uuid.UUID      `json:"user_id,omitempty"`
	TagID    *uuid.UUID      `json:"tag_id,omitempty"`
	Body     string          `json:"body,omitempty"`
	Kind     string          `json:"kind,omitempty"`
	Payload  json.RawMessage `json:"payload,omitempty"`
}

func New(repo domain.AutomationRepo, pool *pgxpool.Pool, actions Actions, log *slog.Logger) *Engine {
	e := &Engine{
		repo:    repo,
		actions: actions,
		pool:    pool,
		log:     log,
		queue:   make(chan Event, 256),
	}
	go e.run()
	return e
}

// Dispatch enqueues the event non-blocking. If the queue is full the event is
// dropped and logged — a signal that the worker goroutine is overwhelmed and
// we should scale out to a real queue.
func (e *Engine) Dispatch(ev Event) {
	if e == nil {
		return
	}
	select {
	case e.queue <- ev:
	default:
		if e.log != nil {
			e.log.Warn("automation queue full; dropping event", "type", ev.Type)
		}
	}
}

func (e *Engine) run() {
	for ev := range e.queue {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		e.handle(ctx, ev)
		cancel()
	}
}

func (e *Engine) handle(ctx context.Context, ev Event) {
	if ev.Task == nil {
		return
	}
	wsID, err := e.resolveWorkspace(ctx, ev.Task.ListID)
	if err != nil {
		if e.log != nil {
			e.log.Warn("automation: resolve workspace failed", "err", err)
		}
		return
	}

	automations, err := e.repo.ListEnabledForTrigger(ctx, &ev.Task.ListID, wsID, ev.Type)
	if err != nil {
		if e.log != nil {
			e.log.Warn("automation: list failed", "err", err)
		}
		return
	}

	for i := range automations {
		a := &automations[i]
		if !e.triggerMatches(a, ev) {
			continue
		}
		if !e.conditionsMatch(a, ev.Task) {
			continue
		}
		e.runActions(ctx, a, ev)
		if err := e.repo.RecordRun(ctx, a.ID); err != nil && e.log != nil {
			e.log.Warn("automation: record run failed", "err", err, "id", a.ID)
		}
	}
}

func (e *Engine) triggerMatches(a *domain.Automation, ev Event) bool {
	var t triggerCfg
	if err := json.Unmarshal(a.Trigger, &t); err != nil {
		return false
	}
	if t.Type != ev.Type {
		return false
	}
	switch ev.Type {
	case domain.TriggerStatusChanged:
		if t.ToStatusID != nil {
			if ev.Task.StatusID == nil || *ev.Task.StatusID != *t.ToStatusID {
				return false
			}
		}
	case domain.TriggerAssigned:
		if t.UserID != nil && ev.Task.AssigneeID != nil && *t.UserID != *ev.Task.AssigneeID {
			return false
		}
	}
	return true
}

func (e *Engine) conditionsMatch(a *domain.Automation, task *domain.Task) bool {
	if len(a.Conditions) == 0 || string(a.Conditions) == "null" {
		return true
	}
	var conds []conditionCfg
	if err := json.Unmarshal(a.Conditions, &conds); err != nil {
		return true // malformed — be permissive; surface errors in UI
	}
	for _, c := range conds {
		if !evalCondition(task, c) {
			return false
		}
	}
	return true
}

func evalCondition(t *domain.Task, c conditionCfg) bool {
	switch c.Field {
	case "priority":
		n, ok := toInt(c.Value)
		if !ok {
			return true
		}
		switch c.Op {
		case "eq":
			return t.Priority == n
		case "neq":
			return t.Priority != n
		case "gte":
			return t.Priority >= n
		case "lte":
			return t.Priority <= n
		}
	case "status_id":
		id, ok := toUUID(c.Value)
		if !ok {
			return true
		}
		if c.Op == "eq" {
			return t.StatusID != nil && *t.StatusID == id
		}
		if c.Op == "neq" {
			return t.StatusID == nil || *t.StatusID != id
		}
	case "assignee_id":
		id, ok := toUUID(c.Value)
		if !ok {
			return true
		}
		if c.Op == "eq" {
			return t.AssigneeID != nil && *t.AssigneeID == id
		}
	}
	return true
}

func (e *Engine) runActions(ctx context.Context, a *domain.Automation, ev Event) {
	var actions []actionCfg
	if err := json.Unmarshal(a.Actions, &actions); err != nil {
		if e.log != nil {
			e.log.Warn("automation: decode actions failed", "err", err, "id", a.ID)
		}
		return
	}
	for _, act := range actions {
		if err := e.runAction(ctx, act, ev); err != nil && e.log != nil {
			e.log.Warn("automation: action failed", "err", err, "action", act.Type, "id", a.ID)
		}
	}
}

func (e *Engine) runAction(ctx context.Context, act actionCfg, ev Event) error {
	switch act.Type {
	case domain.ActionChangeStatus:
		if act.StatusID == nil {
			return nil
		}
		return e.actions.ChangeStatus(ctx, ev.ActorID, ev.Task.ID, *act.StatusID)
	case domain.ActionAssignUser:
		if act.UserID == nil {
			return nil
		}
		return e.actions.AssignUser(ctx, ev.ActorID, ev.Task.ID, *act.UserID)
	case domain.ActionAddTag:
		if act.TagID == nil {
			return nil
		}
		return e.actions.AddTag(ctx, ev.ActorID, ev.Task.ID, *act.TagID)
	case domain.ActionAddComment:
		if act.Body == "" {
			return nil
		}
		return e.actions.AddComment(ctx, ev.ActorID, ev.Task.ID, act.Body)
	case domain.ActionNotify:
		if act.UserID == nil {
			return nil
		}
		payload := map[string]any{
			"task_id":   ev.Task.ID,
			"task_name": ev.Task.Name,
			"from":      "automation",
		}
		if len(act.Payload) > 0 {
			var extra map[string]any
			if err := json.Unmarshal(act.Payload, &extra); err == nil {
				for k, v := range extra {
					payload[k] = v
				}
			}
		}
		kind := act.Kind
		if kind == "" {
			kind = "automation.notice"
		}
		e.actions.Notify(ctx, ev.ActorID, *act.UserID, kind, payload)
	}
	return nil
}

func (e *Engine) resolveWorkspace(ctx context.Context, listID uuid.UUID) (uuid.UUID, error) {
	var ws uuid.UUID
	err := e.pool.QueryRow(ctx,
		`SELECT s.workspace_id FROM spaces s JOIN lists l ON l.space_id = s.id WHERE l.id = $1`,
		listID).Scan(&ws)
	return ws, err
}

func toInt(v any) (int, bool) {
	switch n := v.(type) {
	case float64:
		return int(n), true
	case int:
		return n, true
	case int64:
		return int(n), true
	}
	return 0, false
}

func toUUID(v any) (uuid.UUID, bool) {
	s, ok := v.(string)
	if !ok {
		return uuid.Nil, false
	}
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil, false
	}
	return id, true
}
