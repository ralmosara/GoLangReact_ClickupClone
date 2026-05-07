package comment

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/yourorg/clickup/internal/audit"
	"github.com/yourorg/clickup/internal/domain"
	"github.com/yourorg/clickup/internal/notify"
	"github.com/yourorg/clickup/internal/ws"
)

type Service struct {
	repo   domain.CommentRepo
	tasks  domain.TaskRepo
	hub    *ws.Hub
	notify *notify.Dispatcher
	audit  *audit.Recorder
}

type Deps struct {
	Comments domain.CommentRepo
	Tasks    domain.TaskRepo
	Hub      *ws.Hub
	Notify   *notify.Dispatcher
	Audit    *audit.Recorder
}

func New(d Deps) *Service {
	return &Service{
		repo:   d.Comments,
		tasks:  d.Tasks,
		hub:    d.Hub,
		notify: d.Notify,
		audit:  d.Audit,
	}
}

type CreateInput struct {
	TaskID     uuid.UUID   `json:"task_id"`
	Body       string      `json:"body"`
	MentionIDs []uuid.UUID `json:"mention_ids,omitempty"`
}

func (s *Service) Create(ctx context.Context, authorID uuid.UUID, in CreateInput) (*domain.Comment, error) {
	if strings.TrimSpace(in.Body) == "" {
		return nil, errors.New("body required")
	}
	c := &domain.Comment{TaskID: in.TaskID, AuthorID: &authorID, Body: in.Body}
	if err := s.repo.Create(ctx, c); err != nil {
		return nil, err
	}
	if s.hub != nil {
		payload, _ := json.Marshal(c)
		s.hub.Publish(ws.Event{
			V:        1,
			Room:     ws.RoomTask(c.TaskID.String()),
			Type:     ws.EventCommentCreated,
			ActorID:  authorID.String(),
			EntityID: c.ID.String(),
			TS:       time.Now().UTC(),
			Payload:  payload,
		})
	}
	// Fan out @mention notifications (deduped, excluding author).
	if s.notify != nil && len(in.MentionIDs) > 0 {
		seen := map[uuid.UUID]bool{authorID: true}
		var task *domain.Task
		if s.tasks != nil {
			task, _ = s.tasks.GetByID(ctx, in.TaskID)
		}
		for _, uid := range in.MentionIDs {
			if seen[uid] {
				continue
			}
			seen[uid] = true
			taskID := in.TaskID
			payload := map[string]any{
				"task_id":    taskID,
				"comment_id": c.ID,
				"excerpt":    snippet(in.Body, 120),
			}
			if task != nil {
				payload["task_name"] = task.Name
				payload["list_id"] = task.ListID
			}
			s.notify.Enqueue(notify.Notice{
				Recipient:  uid,
				Actor:      &authorID,
				Kind:       "comment.mention",
				EntityType: "comment",
				EntityID:   &c.ID,
				Payload:    payload,
			})
		}
	}
	if s.audit != nil {
		s.audit.Record(ctx, audit.Entry{
			ActorID:    &authorID,
			EntityType: "comment",
			EntityID:   &c.ID,
			Verb:       "created",
			After:      c,
		})
	}
	return c, nil
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*domain.Comment, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) ListByTask(ctx context.Context, taskID uuid.UUID) ([]domain.Comment, error) {
	return s.repo.ListByTask(ctx, taskID)
}

func (s *Service) Delete(ctx context.Context, actor, id uuid.UUID) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	if s.audit != nil {
		s.audit.Record(ctx, audit.Entry{
			ActorID:    &actor,
			EntityType: "comment",
			EntityID:   &id,
			Verb:       "deleted",
		})
	}
	return nil
}

func snippet(s string, max int) string {
	s = strings.TrimSpace(s)
	if len([]rune(s)) <= max {
		return s
	}
	runes := []rune(s)
	return string(runes[:max]) + "…"
}
