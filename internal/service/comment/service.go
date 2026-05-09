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
	// ParentCommentID — when set, this comment is a reply. The service
	// enforces single-level threading: replying to a reply pulls the
	// grandparent up so the thread stays one deep (Slack-style).
	ParentCommentID *uuid.UUID  `json:"parent_comment_id,omitempty"`
	Body            string      `json:"body"`
	MentionIDs      []uuid.UUID `json:"mention_ids,omitempty"`
}

// ErrNotAuthor is returned by Update when the caller didn't author
// the comment they're trying to edit. Edits are author-only;
// admins go through SoftDelete to remove (replace with placeholder)
// rather than rewrite history.
var ErrNotAuthor = errors.New("only the author can edit this comment")

// ErrCommentDeleted blocks edits/reactions on a soft-deleted comment.
var ErrCommentDeleted = errors.New("comment is deleted")

func (s *Service) Create(ctx context.Context, authorID uuid.UUID, in CreateInput) (*domain.Comment, error) {
	if strings.TrimSpace(in.Body) == "" {
		return nil, errors.New("body required")
	}
	// Single-level threading enforcement: if the parent is itself a
	// reply, target its grandparent instead. Keeps the UI tree shallow.
	if in.ParentCommentID != nil {
		parent, err := s.repo.GetByID(ctx, *in.ParentCommentID)
		if err != nil {
			return nil, err
		}
		if parent == nil {
			return nil, errors.New("parent comment not found")
		}
		if parent.ParentCommentID != nil {
			in.ParentCommentID = parent.ParentCommentID
		}
		// And: replies inherit the parent's task — keeps Drag-To-Move
		// invariants intact.
		in.TaskID = parent.TaskID
	}
	c := &domain.Comment{
		TaskID:          in.TaskID,
		AuthorID:        &authorID,
		ParentCommentID: in.ParentCommentID,
		Body:            in.Body,
	}
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

// ListByTask returns top-level comments + their reply counts, with
// reaction summaries populated for the calling viewer (Reacted = true
// on emojis the caller has used). Replies are loaded on-demand via
// ListReplies so the initial render isn't blocked by deeply-discussed
// threads.
func (s *Service) ListByTask(ctx context.Context, viewer uuid.UUID, taskID uuid.UUID) ([]domain.Comment, error) {
	tops, err := s.repo.ListByTask(ctx, taskID)
	if err != nil {
		return nil, err
	}
	return s.attachReactions(ctx, viewer, tops)
}

// ListReplies returns the children of a parent comment with reactions
// populated for the viewer.
func (s *Service) ListReplies(ctx context.Context, viewer, parentID uuid.UUID) ([]domain.Comment, error) {
	children, err := s.repo.ListReplies(ctx, parentID)
	if err != nil {
		return nil, err
	}
	return s.attachReactions(ctx, viewer, children)
}

// Update edits a comment body. Author-only; admins should use SoftDelete
// instead. Sets edited_at = now() — the UI surfaces "(edited)" off it.
func (s *Service) Update(ctx context.Context, actor, id uuid.UUID, body string) (*domain.Comment, error) {
	body = strings.TrimSpace(body)
	if body == "" {
		return nil, errors.New("body required")
	}
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, errors.New("not found")
	}
	if existing.DeletedAt != nil {
		return nil, ErrCommentDeleted
	}
	if existing.AuthorID == nil || *existing.AuthorID != actor {
		return nil, ErrNotAuthor
	}
	updated, err := s.repo.UpdateBody(ctx, id, body)
	if err != nil {
		return nil, err
	}
	if s.hub != nil {
		payload, _ := json.Marshal(updated)
		s.hub.Publish(ws.Event{
			V:        1,
			Room:     ws.RoomTask(updated.TaskID.String()),
			Type:     ws.EventCommentUpdated,
			ActorID:  actor.String(),
			EntityID: updated.ID.String(),
			TS:       time.Now().UTC(),
			Payload:  payload,
		})
	}
	if s.audit != nil {
		s.audit.Record(ctx, audit.Entry{
			ActorID:    &actor,
			EntityType: "comment",
			EntityID:   &id,
			Verb:       "updated",
			Before:     existing,
			After:      updated,
		})
	}
	return updated, nil
}

// SoftDelete marks a comment as removed but keeps the row so threading
// stays intact. Caller authorisation (author OR comment.delete_any) is
// enforced at the handler.
func (s *Service) SoftDelete(ctx context.Context, actor, id uuid.UUID) error {
	if err := s.repo.SoftDelete(ctx, id, actor); err != nil {
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

// Delete is the legacy hard-delete kept for callers that genuinely want
// the row gone. Prefer SoftDelete for user-initiated removals.
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

// React adds an emoji reaction. Idempotent: re-reacting with the same
// emoji is a no-op.
func (s *Service) React(ctx context.Context, actor, commentID uuid.UUID, emoji string) error {
	if emoji == "" {
		return errors.New("emoji required")
	}
	c, err := s.repo.GetByID(ctx, commentID)
	if err != nil {
		return err
	}
	if c == nil {
		return errors.New("comment not found")
	}
	if c.DeletedAt != nil {
		return ErrCommentDeleted
	}
	return s.repo.AddReaction(ctx, commentID, actor, emoji)
}

func (s *Service) Unreact(ctx context.Context, actor, commentID uuid.UUID, emoji string) error {
	return s.repo.RemoveReaction(ctx, commentID, actor, emoji)
}

func (s *Service) attachReactions(ctx context.Context, viewer uuid.UUID, list []domain.Comment) ([]domain.Comment, error) {
	if len(list) == 0 {
		return list, nil
	}
	ids := make([]uuid.UUID, 0, len(list))
	for _, c := range list {
		ids = append(ids, c.ID)
	}
	summaries, err := s.repo.ListReactionSummaries(ctx, ids, viewer)
	if err != nil {
		// Reactions are decorative — failing the whole listing because
		// the aggregate query erred out is hostile UX. Log via the
		// caller's slog and ship the bare list.
		return list, nil
	}
	for i := range list {
		if r, ok := summaries[list[i].ID]; ok {
			list[i].Reactions = r
		}
	}
	return list, nil
}

func snippet(s string, max int) string {
	s = strings.TrimSpace(s)
	if len([]rune(s)) <= max {
		return s
	}
	runes := []rune(s)
	return string(runes[:max]) + "…"
}
