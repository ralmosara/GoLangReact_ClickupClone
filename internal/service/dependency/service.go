package dependency

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/yourorg/clickup/internal/audit"
	"github.com/yourorg/clickup/internal/domain"
	"github.com/yourorg/clickup/internal/ws"
)

var (
	ErrSelfDependency = errors.New("a task cannot depend on itself")
	ErrCycleDetected  = errors.New("adding this dependency would create a cycle")
)

type Service struct {
	repo  domain.DependencyRepo
	tasks domain.TaskRepo
	hub   *ws.Hub
	audit *audit.Recorder
}

func New(repo domain.DependencyRepo, tasks domain.TaskRepo, hub *ws.Hub, rec *audit.Recorder) *Service {
	return &Service{repo: repo, tasks: tasks, hub: hub, audit: rec}
}

type CreateInput struct {
	DependsOnID uuid.UUID `json:"depends_on_id"`
	Kind        string    `json:"kind,omitempty"`
}

func (s *Service) Create(ctx context.Context, actor, taskID uuid.UUID, in CreateInput) (*domain.TaskDependency, error) {
	if taskID == in.DependsOnID {
		return nil, ErrSelfDependency
	}
	cycle, err := s.repo.WouldCreateCycle(ctx, taskID, in.DependsOnID)
	if err != nil {
		return nil, err
	}
	if cycle {
		return nil, ErrCycleDetected
	}
	kind := in.Kind
	if kind == "" {
		kind = domain.DepKindWaitingOn
	}
	d := &domain.TaskDependency{
		TaskID:      taskID,
		DependsOnID: in.DependsOnID,
		Kind:        kind,
	}
	if err := s.repo.Create(ctx, d); err != nil {
		return nil, err
	}
	s.publish(d, "dependency.added", actor)
	if s.audit != nil {
		s.audit.Record(ctx, audit.Entry{
			ActorID:    &actor,
			EntityType: "task",
			EntityID:   &taskID,
			Verb:       "dependency.added",
			After:      d,
		})
	}
	return d, nil
}

func (s *Service) Delete(ctx context.Context, actor, taskID, dependsOnID uuid.UUID) error {
	if err := s.repo.Delete(ctx, taskID, dependsOnID); err != nil {
		return err
	}
	s.publish(&domain.TaskDependency{TaskID: taskID, DependsOnID: dependsOnID}, "dependency.removed", actor)
	if s.audit != nil {
		s.audit.Record(ctx, audit.Entry{
			ActorID:    &actor,
			EntityType: "task",
			EntityID:   &taskID,
			Verb:       "dependency.removed",
			Before:     map[string]uuid.UUID{"depends_on_id": dependsOnID},
		})
	}
	return nil
}

// Graph returns both "waiting on" (this task depends on X) and "blocking"
// (X depends on this task) edges so the UI can render both panels.
type Graph struct {
	WaitingOn []domain.TaskDependency `json:"waiting_on"`
	Blocking  []domain.TaskDependency `json:"blocking"`
}

func (s *Service) Graph(ctx context.Context, taskID uuid.UUID) (*Graph, error) {
	waiting, err := s.repo.ListByTask(ctx, taskID)
	if err != nil {
		return nil, err
	}
	blocking, err := s.repo.ListDependents(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if waiting == nil {
		waiting = []domain.TaskDependency{}
	}
	if blocking == nil {
		blocking = []domain.TaskDependency{}
	}
	return &Graph{WaitingOn: waiting, Blocking: blocking}, nil
}

func (s *Service) publish(d *domain.TaskDependency, kind string, actor uuid.UUID) {
	if s.hub == nil {
		return
	}
	payload, _ := json.Marshal(d)
	s.hub.Publish(ws.Event{
		V:        1,
		Room:     ws.RoomTask(d.TaskID.String()),
		Type:     kind,
		ActorID:  actor.String(),
		EntityID: d.TaskID.String(),
		TS:       time.Now().UTC(),
		Payload:  payload,
	})
}
