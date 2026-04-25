package attachment

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"

	"github.com/yourorg/clickup/internal/audit"
	"github.com/yourorg/clickup/internal/domain"
	"github.com/yourorg/clickup/internal/storage"
	"github.com/yourorg/clickup/internal/ws"
)

type Service struct {
	repo  domain.AttachmentRepo
	store storage.Store
	hub   *ws.Hub
	audit *audit.Recorder
}

func New(repo domain.AttachmentRepo, store storage.Store, hub *ws.Hub, rec *audit.Recorder) *Service {
	return &Service{repo: repo, store: store, hub: hub, audit: rec}
}

type UploadInput struct {
	TaskID   uuid.UUID
	Uploader uuid.UUID
	Filename string
	MimeType string
	Body     io.Reader
}

func (s *Service) Upload(ctx context.Context, in UploadInput) (*domain.Attachment, error) {
	if in.Filename == "" {
		return nil, errors.New("filename required")
	}
	key := fmt.Sprintf("tasks/%s/%s-%s", in.TaskID, uuid.NewString(), in.Filename)
	meta, err := s.store.Put(ctx, key, in.Body)
	if err != nil {
		return nil, err
	}
	mime := in.MimeType
	if mime == "" {
		mime = "application/octet-stream"
	}
	uid := in.Uploader
	att := &domain.Attachment{
		TaskID:     in.TaskID,
		UploaderID: &uid,
		Filename:   in.Filename,
		Path:       key,
		SizeBytes:  meta.Size,
		MimeType:   mime,
	}
	if err := s.repo.Create(ctx, att); err != nil {
		_ = s.store.Delete(ctx, key)
		return nil, err
	}
	s.publish(att, ws.EventAttachmentCreated, in.Uploader)
	if s.audit != nil {
		s.audit.Record(ctx, audit.Entry{
			ActorID:    &uid,
			EntityType: "attachment",
			EntityID:   &att.ID,
			Verb:       "created",
			After:      att,
		})
	}
	return att, nil
}

func (s *Service) ListByTask(ctx context.Context, taskID uuid.UUID) ([]domain.Attachment, error) {
	return s.repo.ListByTask(ctx, taskID)
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*domain.Attachment, error) {
	return s.repo.GetByID(ctx, id)
}

// Open returns a reader for the blob plus its metadata; caller is responsible for closing.
func (s *Service) Open(ctx context.Context, id uuid.UUID) (*domain.Attachment, io.ReadCloser, error) {
	a, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	if a == nil {
		return nil, nil, errors.New("not found")
	}
	rc, err := s.store.Open(ctx, a.Path)
	if err != nil {
		return nil, nil, err
	}
	return a, rc, nil
}

func (s *Service) Delete(ctx context.Context, actor, id uuid.UUID) error {
	a, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if a == nil {
		return nil
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	_ = s.store.Delete(ctx, a.Path)
	s.publish(a, ws.EventAttachmentDeleted, actor)
	if s.audit != nil {
		s.audit.Record(ctx, audit.Entry{
			ActorID:    &actor,
			EntityType: "attachment",
			EntityID:   &a.ID,
			Verb:       "deleted",
			Before:     a,
		})
	}
	return nil
}

func (s *Service) publish(a *domain.Attachment, kind string, actor uuid.UUID) {
	if s.hub == nil {
		return
	}
	payload, _ := json.Marshal(a)
	s.hub.Publish(ws.Event{
		V:        1,
		Room:     ws.RoomTask(a.TaskID.String()),
		Type:     kind,
		ActorID:  actor.String(),
		EntityID: a.ID.String(),
		TS:       time.Now().UTC(),
		Payload:  payload,
	})
}
