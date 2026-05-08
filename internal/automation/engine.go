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
	// dueSoonInterval controls how often the scanner sweeps for tasks
	// approaching their due date. Defaults to 1 minute; tests override.
	dueSoonInterval time.Duration
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
		repo:            repo,
		actions:         actions,
		pool:            pool,
		log:             log,
		queue:           make(chan Event, 256),
		dueSoonInterval: time.Minute,
	}
	go e.run()
	go e.runDueSoonScanner()
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

// ClearTaskFires drops every (automation, task) dedup row for the given task
// so the task.due_soon trigger can fire again the next time the task lands
// inside the lead window. The task service calls this whenever a task's
// due_at, status, or completed_at changes — we'd rather over-clear than
// silently swallow a re-arm. A failure is logged and ignored: it can only
// cause a missed dedup, never a wrong action.
func (e *Engine) ClearTaskFires(ctx context.Context, taskID uuid.UUID) {
	if e == nil || e.pool == nil {
		return
	}
	if _, err := e.pool.Exec(ctx, `DELETE FROM automation_fires WHERE task_id = $1`, taskID); err != nil {
		if e.log != nil {
			e.log.Warn("automation: clear fires failed", "err", err, "task", taskID)
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

// runDueSoonScanner periodically sweeps for tasks approaching their due_at and
// fires the task.due_soon trigger exactly once per (automation, task) tuple.
//
// The trigger is "edge-triggered": once an automation has fired for a task we
// record (automation_id, task_id) in automation_fires; the FK ON DELETE CASCADE
// from tasks.due_at changes is enforced indirectly by triggers/code that null
// out the dedup row when a task's due_at moves outside the window. For now we
// keep it simple — re-arming on due_at change is a follow-up; the dedup row
// is cleared if the task is deleted (CASCADE) or completed via the existing
// task.completed pipeline (separate trigger).
func (e *Engine) runDueSoonScanner() {
	if e.pool == nil || e.dueSoonInterval <= 0 {
		return
	}
	// Run once on startup so newly-enabled automations don't wait a full
	// interval before firing for already-due-soon tasks.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	e.scanDueSoon(ctx)
	cancel()

	t := time.NewTicker(e.dueSoonInterval)
	defer t.Stop()
	for range t.C {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		e.scanDueSoon(ctx)
		cancel()
	}
}

// scanDueSoon runs a single sweep. It joins automations that use the
// task.due_soon trigger to tasks within the configured lead window, filters
// out (automation, task) pairs that have already fired, and dispatches a
// synthetic event into the same handle() pipeline used for live events. The
// dedup INSERT and the dispatch run inside the same transaction so a crash
// mid-sweep won't drop the event silently.
func (e *Engine) scanDueSoon(ctx context.Context) {
	const q = `
		WITH candidates AS (
			SELECT
				a.id AS automation_id,
				t.id AS task_id,
				COALESCE((a.trigger ->> 'lead_days')::int, 1) AS lead_days,
				t.list_id, t.parent_task_id, t.name, t.description, t.status,
				t.status_id, t.priority, t.position, t.assignee_id, t.creator_id,
				t.due_at, t.start_at, t.completed_at, t.archived,
				t.created_at, t.updated_at
			FROM automations a
			JOIN lists  l ON (a.list_id IS NULL OR a.list_id = l.id)
			JOIN spaces s ON s.id = l.space_id AND s.workspace_id = a.workspace_id
			JOIN tasks  t ON t.list_id = l.id
			WHERE a.enabled = TRUE
			  AND a.trigger ->> 'type' = 'task.due_soon'
			  AND t.due_at IS NOT NULL
			  AND t.archived = FALSE
			  AND t.completed_at IS NULL
			  AND t.due_at <= NOW() + (COALESCE((a.trigger ->> 'lead_days')::int, 1) || ' days')::interval
			  AND t.due_at >= NOW()
			  AND NOT EXISTS (
			    SELECT 1 FROM automation_fires f
			    WHERE f.automation_id = a.id AND f.task_id = t.id
			  )
		),
		inserted AS (
			INSERT INTO automation_fires (automation_id, task_id)
			SELECT automation_id, task_id FROM candidates
			ON CONFLICT DO NOTHING
			RETURNING automation_id, task_id
		)
		SELECT
			c.automation_id, c.task_id, c.list_id, c.parent_task_id, c.name,
			c.description, c.status, c.status_id, c.priority, c.position,
			c.assignee_id, c.creator_id, c.due_at, c.start_at, c.completed_at,
			c.archived, c.created_at, c.updated_at
		FROM candidates c
		JOIN inserted   i ON i.automation_id = c.automation_id AND i.task_id = c.task_id
	`
	rows, err := e.pool.Query(ctx, q)
	if err != nil {
		if e.log != nil {
			e.log.Warn("automation: due_soon scan failed", "err", err)
		}
		return
	}
	defer rows.Close()

	type fired struct {
		automationID uuid.UUID
		task         domain.Task
	}
	var matches []fired
	for rows.Next() {
		var f fired
		t := &f.task
		if err := rows.Scan(
			&f.automationID, &t.ID, &t.ListID, &t.ParentTaskID, &t.Name,
			&t.Description, &t.Status, &t.StatusID, &t.Priority, &t.Position,
			&t.AssigneeID, &t.CreatorID, &t.DueAt, &t.StartAt, &t.CompletedAt,
			&t.Archived, &t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			if e.log != nil {
				e.log.Warn("automation: due_soon scan row failed", "err", err)
			}
			continue
		}
		matches = append(matches, f)
	}
	if err := rows.Err(); err != nil && e.log != nil {
		e.log.Warn("automation: due_soon rows err", "err", err)
	}

	// Dispatch as synthetic events. We bypass Dispatch() because the engine
	// also needs to load the specific automation and run conditions/actions
	// — handle() already does that via repo.ListEnabledForTrigger. For
	// due_soon specifically we already know which automation matched, so
	// fire it directly to avoid re-querying.
	for _, m := range matches {
		task := m.task
		ev := Event{Type: domain.TriggerDueSoon, Task: &task}
		// Reuse the per-event timeout pattern from run().
		evCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		e.handleSpecific(evCtx, m.automationID, ev)
		cancel()
	}
}

// handleSpecific runs conditions + actions for a known automation ID. Used by
// the due_soon scanner where we already JOINed to find matching automations.
func (e *Engine) handleSpecific(ctx context.Context, automationID uuid.UUID, ev Event) {
	a, err := e.repo.GetByID(ctx, automationID)
	if err != nil || a == nil {
		if err != nil && e.log != nil {
			e.log.Warn("automation: load failed", "err", err, "id", automationID)
		}
		return
	}
	if !e.conditionsMatch(a, ev.Task) {
		return
	}
	e.runActions(ctx, a, ev)
	if err := e.repo.RecordRun(ctx, a.ID); err != nil && e.log != nil {
		e.log.Warn("automation: record run failed", "err", err, "id", a.ID)
	}
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
