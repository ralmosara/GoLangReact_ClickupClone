package task

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	rrulego "github.com/teambition/rrule-go"

	"github.com/yourorg/clickup/internal/audit"
	"github.com/yourorg/clickup/internal/automation"
	"github.com/yourorg/clickup/internal/domain"
	"github.com/yourorg/clickup/internal/notify"
	"github.com/yourorg/clickup/internal/ws"
)

type Service struct {
	repo      domain.TaskRepo
	statuses  domain.StatusRepo
	assignees domain.AssigneeRepo
	hub       *ws.Hub
	notify    *notify.Dispatcher
	audit     *audit.Recorder
	engine    *automation.Engine
}

type Deps struct {
	Tasks     domain.TaskRepo
	Statuses  domain.StatusRepo // optional — required for done-category detection on reorder
	Assignees domain.AssigneeRepo
	Hub       *ws.Hub
	Notify    *notify.Dispatcher
	Audit     *audit.Recorder
	Engine    *automation.Engine // optional; nil during bootstrapping
}

func New(d Deps) *Service {
	return &Service{
		repo:      d.Tasks,
		statuses:  d.Statuses,
		assignees: d.Assignees,
		hub:       d.Hub,
		notify:    d.Notify,
		audit:     d.Audit,
		engine:    d.Engine,
	}
}

// SetEngine lets main.go break the initialization cycle (engine depends on
// task actions, which depend on the task service).
func (s *Service) SetEngine(e *automation.Engine) { s.engine = e }

type CreateInput struct {
	ListID         uuid.UUID   `json:"list_id"`
	ParentTaskID   *uuid.UUID  `json:"parent_task_id,omitempty"`
	Name           string      `json:"name"`
	Description    string      `json:"description"`
	Status         string      `json:"status"`
	StatusID       *uuid.UUID  `json:"status_id,omitempty"`
	Priority       int         `json:"priority"`
	AssigneeIDs    []uuid.UUID `json:"assignee_ids,omitempty"`
	DueAt          *time.Time  `json:"due_at,omitempty"`
	StartAt        *time.Time  `json:"start_at,omitempty"`
	RecurringRule  *string     `json:"recurring_rule,omitempty"`
	SprintID       *uuid.UUID  `json:"sprint_id,omitempty"`
	Points         *int        `json:"points,omitempty"`
}

type UpdateInput struct {
	Name          *string    `json:"name,omitempty"`
	Description   *string    `json:"description,omitempty"`
	Status        *string    `json:"status,omitempty"`
	StatusID      *uuid.UUID `json:"status_id,omitempty"`
	Priority      *int       `json:"priority,omitempty"`
	Position      *float64   `json:"position,omitempty"`
	DueAt         *time.Time `json:"due_at,omitempty"`
	StartAt       *time.Time `json:"start_at,omitempty"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
	Archived      *bool      `json:"archived,omitempty"`
	ListID        *uuid.UUID `json:"list_id,omitempty"`
	RecurringRule *string    `json:"recurring_rule,omitempty"`
	ClearRecur    *bool      `json:"clear_recurring,omitempty"`
	SprintID      *uuid.UUID `json:"sprint_id,omitempty"`
	ClearSprint   *bool      `json:"clear_sprint,omitempty"`
	Points        *int       `json:"points,omitempty"`
	ClearPoints   *bool      `json:"clear_points,omitempty"`
}

type ReorderInput struct {
	StatusID *uuid.UUID `json:"status_id,omitempty"`
	Position *float64   `json:"position,omitempty"`
	Prev     *float64   `json:"prev,omitempty"`
	Next     *float64   `json:"next,omitempty"`
}

func (s *Service) Create(ctx context.Context, creator uuid.UUID, in CreateInput) (*domain.Task, error) {
	if strings.TrimSpace(in.Name) == "" {
		return nil, errors.New("name required")
	}
	status := in.Status
	if status == "" {
		status = "open"
	}
	if in.RecurringRule != nil && strings.TrimSpace(*in.RecurringRule) != "" {
		if _, err := rrulego.StrToRRule(*in.RecurringRule); err != nil {
			return nil, errors.New("invalid recurring_rule (RRULE): " + err.Error())
		}
	}
	t := &domain.Task{
		ListID:        in.ListID,
		ParentTaskID:  in.ParentTaskID,
		Name:          in.Name,
		Description:   in.Description,
		Status:        status,
		StatusID:      in.StatusID,
		Priority:      in.Priority,
		CreatorID:     &creator,
		DueAt:         in.DueAt,
		StartAt:       in.StartAt,
		RecurringRule: trimNonEmpty(in.RecurringRule),
		SprintID:      in.SprintID,
		Points:        in.Points,
	}
	if err := s.repo.Create(ctx, t); err != nil {
		return nil, err
	}
	if s.assignees != nil {
		for _, uid := range in.AssigneeIDs {
			if err := s.assignees.Add(ctx, t.ID, uid); err == nil && uid != creator {
				s.dispatchAssignNotification(t, creator, uid)
			}
		}
	}
	s.publish(t, ws.EventTaskCreated, creator)
	if t.ParentTaskID != nil {
		s.publish(t, ws.EventSubtaskCreated, creator)
	}
	if s.audit != nil {
		s.audit.Record(ctx, audit.Entry{
			ActorID:    &creator,
			EntityType: "task",
			EntityID:   &t.ID,
			Verb:       "created",
			After:      t,
		})
	}
	s.dispatchAutomation(automation.Event{Type: domain.TriggerCreated, Task: t, ActorID: creator})
	return t, nil
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*domain.Task, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) List(ctx context.Context, f domain.TaskFilter) ([]domain.Task, error) {
	return s.repo.List(ctx, f)
}

func (s *Service) ListSubtasks(ctx context.Context, parentID uuid.UUID) ([]domain.Task, error) {
	return s.repo.ListSubtasks(ctx, parentID)
}

func (s *Service) Update(ctx context.Context, actor uuid.UUID, id uuid.UUID, in UpdateInput) (*domain.Task, error) {
	t, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, errors.New("not found")
	}
	before := *t
	if in.Name != nil {
		t.Name = *in.Name
	}
	if in.Description != nil {
		t.Description = *in.Description
	}
	statusChanged := false
	if in.Status != nil {
		if t.Status != *in.Status {
			statusChanged = true
		}
		t.Status = *in.Status
		if *in.Status == "completed" && t.CompletedAt == nil {
			now := time.Now()
			t.CompletedAt = &now
		}
		if *in.Status != "completed" {
			t.CompletedAt = nil
		}
	}
	if in.StatusID != nil {
		if t.StatusID == nil || *t.StatusID != *in.StatusID {
			statusChanged = true
		}
		t.StatusID = in.StatusID
		// If the new status belongs to a done category, mark completion.
		if s.statuses != nil {
			if st, _ := s.statuses.GetByID(ctx, *in.StatusID); st != nil {
				if st.Category == "done" || st.Category == "closed" {
					if t.CompletedAt == nil {
						now := time.Now()
						t.CompletedAt = &now
					}
				} else if t.CompletedAt != nil {
					t.CompletedAt = nil
				}
			}
		}
	}
	if in.Priority != nil {
		t.Priority = *in.Priority
	}
	if in.Position != nil {
		t.Position = *in.Position
	}
	if in.DueAt != nil {
		t.DueAt = in.DueAt
	}
	if in.StartAt != nil {
		t.StartAt = in.StartAt
	}
	if in.CompletedAt != nil {
		t.CompletedAt = in.CompletedAt
	}
	if in.Archived != nil {
		t.Archived = *in.Archived
	}
	if in.ListID != nil {
		t.ListID = *in.ListID
	}
	if in.RecurringRule != nil {
		trimmed := strings.TrimSpace(*in.RecurringRule)
		if trimmed == "" {
			t.RecurringRule = nil
		} else {
			if _, err := rrulego.StrToRRule(trimmed); err != nil {
				return nil, errors.New("invalid recurring_rule (RRULE): " + err.Error())
			}
			t.RecurringRule = &trimmed
		}
	}
	if in.ClearRecur != nil && *in.ClearRecur {
		t.RecurringRule = nil
	}
	if in.SprintID != nil {
		t.SprintID = in.SprintID
	}
	if in.ClearSprint != nil && *in.ClearSprint {
		t.SprintID = nil
	}
	if in.Points != nil {
		t.Points = in.Points
	}
	if in.ClearPoints != nil && *in.ClearPoints {
		t.Points = nil
	}
	if err := s.repo.Update(ctx, t); err != nil {
		return nil, err
	}
	s.publish(t, ws.EventTaskUpdated, actor)
	if s.audit != nil {
		s.audit.Record(ctx, audit.Entry{
			ActorID:    &actor,
			EntityType: "task",
			EntityID:   &t.ID,
			Verb:       "updated",
			Before:     before,
			After:      t,
		})
	}

	// Fire events (status change, completion) and spawn recurring copy.
	if statusChanged {
		s.dispatchAutomation(automation.Event{Type: domain.TriggerStatusChanged, Task: t, Before: &before, ActorID: actor})
	}
	if before.CompletedAt == nil && t.CompletedAt != nil {
		s.dispatchAutomation(automation.Event{Type: domain.TriggerCompleted, Task: t, Before: &before, ActorID: actor})
		s.maybeSpawnRecurring(ctx, t, actor)
	}
	return t, nil
}

func (s *Service) Reorder(ctx context.Context, actor, id uuid.UUID, in ReorderInput) (*domain.Task, error) {
	t, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, errors.New("not found")
	}
	before := *t

	pos := t.Position
	switch {
	case in.Position != nil:
		pos = *in.Position
	case in.Prev != nil && in.Next != nil:
		pos = (*in.Prev + *in.Next) / 2
	case in.Prev != nil:
		pos = *in.Prev + 1000
	case in.Next != nil:
		pos = *in.Next - 1000
	default:
		max, _ := s.repo.MaxPosition(ctx, t.ListID, in.StatusID)
		pos = max + 1000
	}

	newStatusID := t.StatusID
	if in.StatusID != nil {
		newStatusID = in.StatusID
	}

	if err := s.repo.Reorder(ctx, id, newStatusID, pos); err != nil {
		return nil, err
	}
	statusChanged := (t.StatusID == nil) != (newStatusID == nil) ||
		(t.StatusID != nil && newStatusID != nil && *t.StatusID != *newStatusID)
	t.StatusID = newStatusID
	t.Position = pos

	// Moving to a done-category column should also mark completion (ClickUp behaviour).
	if statusChanged && newStatusID != nil && s.statuses != nil {
		if st, _ := s.statuses.GetByID(ctx, *newStatusID); st != nil {
			if st.Category == "done" || st.Category == "closed" {
				if t.CompletedAt == nil {
					now := time.Now()
					t.CompletedAt = &now
					_ = s.repo.Update(ctx, t)
				}
			}
		}
	}

	s.publish(t, ws.EventTaskMoved, actor)
	if s.audit != nil {
		s.audit.Record(ctx, audit.Entry{
			ActorID:    &actor,
			EntityType: "task",
			EntityID:   &t.ID,
			Verb:       "moved",
			After: map[string]any{
				"status_id": newStatusID,
				"position":  pos,
			},
		})
	}
	if statusChanged {
		s.dispatchAutomation(automation.Event{Type: domain.TriggerStatusChanged, Task: t, Before: &before, ActorID: actor})
	}
	if before.CompletedAt == nil && t.CompletedAt != nil {
		s.dispatchAutomation(automation.Event{Type: domain.TriggerCompleted, Task: t, Before: &before, ActorID: actor})
		s.maybeSpawnRecurring(ctx, t, actor)
	}
	return t, nil
}

func (s *Service) Delete(ctx context.Context, actor uuid.UUID, id uuid.UUID) error {
	t, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	if t != nil {
		s.publish(t, ws.EventTaskDeleted, actor)
		if s.audit != nil {
			s.audit.Record(ctx, audit.Entry{
				ActorID:    &actor,
				EntityType: "task",
				EntityID:   &t.ID,
				Verb:       "deleted",
				Before:     t,
			})
		}
	}
	return nil
}

// ChangeStatus is a thin wrapper used by the automation engine's
// ChangeStatus action. It exists so engine callbacks don't need to
// construct an UpdateInput.
func (s *Service) ChangeStatus(ctx context.Context, actor, taskID, statusID uuid.UUID) error {
	_, err := s.Update(ctx, actor, taskID, UpdateInput{StatusID: &statusID})
	return err
}

// --- assignee helpers -------------------------------------------------------

func (s *Service) AddAssignee(ctx context.Context, actor, taskID, userID uuid.UUID) error {
	if s.assignees == nil {
		return errors.New("assignees unsupported")
	}
	if err := s.assignees.Add(ctx, taskID, userID); err != nil {
		return err
	}
	t, _ := s.repo.GetByID(ctx, taskID)
	if t != nil {
		s.publishAssignee(t, userID, ws.EventAssigneeAdded, actor)
		if actor != userID {
			s.dispatchAssignNotification(t, actor, userID)
		}
		if s.audit != nil {
			s.audit.Record(ctx, audit.Entry{
				ActorID:    &actor,
				EntityType: "task",
				EntityID:   &taskID,
				Verb:       "assignee.added",
				After:      map[string]uuid.UUID{"user_id": userID},
			})
		}
		assignee := userID
		copy := *t
		copy.AssigneeID = &assignee
		s.dispatchAutomation(automation.Event{Type: domain.TriggerAssigned, Task: &copy, ActorID: actor})
	}
	return nil
}

func (s *Service) RemoveAssignee(ctx context.Context, actor, taskID, userID uuid.UUID) error {
	if s.assignees == nil {
		return errors.New("assignees unsupported")
	}
	if err := s.assignees.Remove(ctx, taskID, userID); err != nil {
		return err
	}
	t, _ := s.repo.GetByID(ctx, taskID)
	if t != nil {
		s.publishAssignee(t, userID, ws.EventAssigneeRemoved, actor)
		if s.audit != nil {
			s.audit.Record(ctx, audit.Entry{
				ActorID:    &actor,
				EntityType: "task",
				EntityID:   &taskID,
				Verb:       "assignee.removed",
				Before:     map[string]uuid.UUID{"user_id": userID},
			})
		}
	}
	return nil
}

func (s *Service) ListAssignees(ctx context.Context, taskID uuid.UUID) ([]uuid.UUID, error) {
	if s.assignees == nil {
		return nil, nil
	}
	return s.assignees.ListByTask(ctx, taskID)
}

// --- recurring ---------------------------------------------------------------

// maybeSpawnRecurring creates the next occurrence of a recurring task when it
// completes. The new task inherits name/description/priority/assignees and has
// its start/due shifted by the delta returned by RRULE.
func (s *Service) maybeSpawnRecurring(ctx context.Context, t *domain.Task, actor uuid.UUID) {
	if t.RecurringRule == nil || *t.RecurringRule == "" {
		return
	}
	rule, err := rrulego.StrToRRule(*t.RecurringRule)
	if err != nil {
		return
	}
	base := time.Now()
	if t.DueAt != nil {
		base = *t.DueAt
	}
	next := rule.After(base, false)
	if next.IsZero() {
		return
	}

	dueAt := next
	var startAt *time.Time
	if t.DueAt != nil && t.StartAt != nil {
		delta := next.Sub(*t.DueAt)
		nv := t.StartAt.Add(delta)
		startAt = &nv
	}

	parent := t.ID
	if t.RecurringParentID != nil {
		parent = *t.RecurringParentID
	}

	newTask := &domain.Task{
		ListID:            t.ListID,
		ParentTaskID:      t.ParentTaskID,
		Name:              t.Name,
		Description:       t.Description,
		Status:            "open",
		Priority:          t.Priority,
		CreatorID:         t.CreatorID,
		DueAt:             &dueAt,
		StartAt:           startAt,
		RecurringRule:     t.RecurringRule,
		RecurringParentID: &parent,
	}
	// Leave StatusID nil so the board shows it in the first active column; users
	// can customise by creating an automation that maps "task.created" → change_status.

	if err := s.repo.Create(ctx, newTask); err != nil {
		return
	}
	// Inherit assignees.
	if s.assignees != nil {
		existing, _ := s.assignees.ListByTask(ctx, t.ID)
		for _, uid := range existing {
			_ = s.assignees.Add(ctx, newTask.ID, uid)
		}
	}
	s.publish(newTask, ws.EventTaskCreated, actor)
	s.dispatchAutomation(automation.Event{Type: domain.TriggerCreated, Task: newTask, ActorID: actor})
	if s.audit != nil {
		s.audit.Record(ctx, audit.Entry{
			ActorID:    &actor,
			EntityType: "task",
			EntityID:   &newTask.ID,
			Verb:       "recurring.spawned",
			After:      newTask,
		})
	}
}

// --- publish / engine helpers ----------------------------------------------

func (s *Service) publish(t *domain.Task, kind string, actor uuid.UUID) {
	if s.hub == nil {
		return
	}
	payload, _ := json.Marshal(t)
	s.hub.Publish(ws.Event{
		V:        1,
		Room:     ws.RoomList(t.ListID.String()),
		Type:     kind,
		ActorID:  actor.String(),
		EntityID: t.ID.String(),
		TS:       time.Now().UTC(),
		Payload:  payload,
	})
}

func (s *Service) publishAssignee(t *domain.Task, userID uuid.UUID, kind string, actor uuid.UUID) {
	if s.hub == nil {
		return
	}
	payload, _ := json.Marshal(map[string]any{
		"task_id": t.ID,
		"user_id": userID,
	})
	s.hub.Publish(ws.Event{
		V:        1,
		Room:     ws.RoomList(t.ListID.String()),
		Type:     kind,
		ActorID:  actor.String(),
		EntityID: t.ID.String(),
		TS:       time.Now().UTC(),
		Payload:  payload,
	})
}

func (s *Service) dispatchAssignNotification(t *domain.Task, actor, recipient uuid.UUID) {
	if s.notify == nil {
		return
	}
	taskID := t.ID
	s.notify.Enqueue(notify.Notice{
		Recipient:  recipient,
		Actor:      &actor,
		Kind:       "task.assigned",
		EntityType: "task",
		EntityID:   &taskID,
		Payload: map[string]any{
			"task_id":   taskID,
			"task_name": t.Name,
			"list_id":   t.ListID,
		},
	})
}

func (s *Service) dispatchAutomation(ev automation.Event) {
	if s.engine == nil {
		return
	}
	s.engine.Dispatch(ev)
}

func trimNonEmpty(s *string) *string {
	if s == nil {
		return nil
	}
	t := strings.TrimSpace(*s)
	if t == "" {
		return nil
	}
	return &t
}
