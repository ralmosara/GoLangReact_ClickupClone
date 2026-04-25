package audit

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/google/uuid"

	"github.com/yourorg/clickup/internal/domain"
)

// Recorder persists audit entries. Callers pass `before` and `after` (nil allowed)
// and the recorder will JSON-encode them lazily. Failures are logged, not propagated,
// so a DB hiccup in audit doesn't fail the caller's business transaction.
type Recorder struct {
	repo domain.AuditRepo
	log  *slog.Logger
}

func New(repo domain.AuditRepo, log *slog.Logger) *Recorder {
	return &Recorder{repo: repo, log: log}
}

type Entry struct {
	WorkspaceID *uuid.UUID
	ActorID     *uuid.UUID
	EntityType  string
	EntityID    *uuid.UUID
	Verb        string
	Before      any
	After       any
}

func (r *Recorder) Record(ctx context.Context, e Entry) {
	if r == nil || r.repo == nil {
		return
	}
	var beforeJSON, afterJSON []byte
	if e.Before != nil {
		beforeJSON, _ = json.Marshal(e.Before)
	}
	if e.After != nil {
		afterJSON, _ = json.Marshal(e.After)
	}
	entry := &domain.AuditEntry{
		WorkspaceID: e.WorkspaceID,
		ActorID:     e.ActorID,
		EntityType:  e.EntityType,
		EntityID:    e.EntityID,
		Verb:        e.Verb,
		BeforeJSON:  beforeJSON,
		AfterJSON:   afterJSON,
	}
	if err := r.repo.Create(ctx, entry); err != nil && r.log != nil {
		r.log.Warn("audit write failed", "err", err, "verb", e.Verb, "entity", e.EntityType)
	}
}
