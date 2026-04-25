package template

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/google/uuid"

	"github.com/yourorg/clickup/internal/audit"
	"github.com/yourorg/clickup/internal/domain"
)

// Appliers decouple the template service from the concrete services that
// actually create tasks / lists / docs. main.go wires these with closures.
type Appliers struct {
	CreateList   func(ctx context.Context, actor, spaceID uuid.UUID, name string) (*domain.List, error)
	CreateStatus func(ctx context.Context, actor, listID uuid.UUID, name, color, category string, order int) error
	CreateTask   func(ctx context.Context, actor, listID uuid.UUID, parentTaskID *uuid.UUID, name, description string) (*domain.Task, error)
	CreateDoc    func(ctx context.Context, actor, workspaceID uuid.UUID, parentID *uuid.UUID, title string, content json.RawMessage) (*domain.Doc, error)
}

type Service struct {
	repo     domain.TemplateRepo
	appliers Appliers
	audit    *audit.Recorder
}

func New(repo domain.TemplateRepo, appliers Appliers, rec *audit.Recorder) *Service {
	return &Service{repo: repo, appliers: appliers, audit: rec}
}

type CreateInput struct {
	WorkspaceID uuid.UUID       `json:"workspace_id"`
	Kind        string          `json:"kind"`
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Snapshot    json.RawMessage `json:"snapshot"`
}

type UpdateInput struct {
	Name        *string         `json:"name,omitempty"`
	Description *string         `json:"description,omitempty"`
	Snapshot    json.RawMessage `json:"snapshot,omitempty"`
}

var validKinds = map[string]bool{
	domain.TemplateKindList: true,
	domain.TemplateKindDoc:  true,
	domain.TemplateKindTask: true,
}

func (s *Service) Create(ctx context.Context, actor uuid.UUID, in CreateInput) (*domain.Template, error) {
	if !validKinds[in.Kind] {
		return nil, errors.New("invalid template kind")
	}
	if strings.TrimSpace(in.Name) == "" {
		return nil, errors.New("name required")
	}
	t := &domain.Template{
		WorkspaceID: in.WorkspaceID,
		Kind:        in.Kind,
		Name:        strings.TrimSpace(in.Name),
		Description: in.Description,
		Snapshot:    in.Snapshot,
		CreatorID:   &actor,
	}
	if err := s.repo.Create(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*domain.Template, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) ListByWorkspace(ctx context.Context, wsID uuid.UUID, kind string) ([]domain.Template, error) {
	return s.repo.ListByWorkspace(ctx, wsID, kind)
}

func (s *Service) Update(ctx context.Context, actor, id uuid.UUID, in UpdateInput) (*domain.Template, error) {
	t, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, errors.New("not found")
	}
	if in.Name != nil {
		t.Name = *in.Name
	}
	if in.Description != nil {
		t.Description = *in.Description
	}
	if len(in.Snapshot) > 0 {
		t.Snapshot = in.Snapshot
	}
	if err := s.repo.Update(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

func (s *Service) Delete(ctx context.Context, actor, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

/* --- apply --------------------------------------------------------------- */

// ApplyListTemplate creates a new list in `spaceID` from a `list` template,
// then inflates its statuses + seed tasks.
type listSnapshot struct {
	Name     string `json:"name"`
	Statuses []struct {
		Name     string `json:"name"`
		Color    string `json:"color"`
		Category string `json:"category"`
	} `json:"statuses,omitempty"`
	Tasks []struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	} `json:"tasks,omitempty"`
}

type taskSnapshot struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Subtasks    []taskSnapshot `json:"subtasks,omitempty"`
}

type docSnapshot struct {
	Title   string          `json:"title"`
	Content json.RawMessage `json:"content,omitempty"`
	Pages   []docSnapshot   `json:"pages,omitempty"`
}

type ApplyTarget struct {
	SpaceID      *uuid.UUID `json:"space_id,omitempty"`       // list template
	ListID       *uuid.UUID `json:"list_id,omitempty"`        // task template
	ParentTaskID *uuid.UUID `json:"parent_task_id,omitempty"` // task template → subtask
	WorkspaceID  *uuid.UUID `json:"workspace_id,omitempty"`   // doc template
	ParentDocID  *uuid.UUID `json:"parent_doc_id,omitempty"`  // doc template → child
}

// Apply inflates the template into real entities. Returns a summary map of
// newly created IDs keyed by entity kind.
func (s *Service) Apply(ctx context.Context, actor, templateID uuid.UUID, target ApplyTarget) (map[string]any, error) {
	t, err := s.repo.GetByID(ctx, templateID)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, errors.New("template not found")
	}
	switch t.Kind {
	case domain.TemplateKindList:
		return s.applyListTemplate(ctx, actor, t, target)
	case domain.TemplateKindTask:
		return s.applyTaskTemplate(ctx, actor, t, target)
	case domain.TemplateKindDoc:
		return s.applyDocTemplate(ctx, actor, t, target)
	}
	return nil, errors.New("unknown template kind")
}

func (s *Service) applyListTemplate(ctx context.Context, actor uuid.UUID, t *domain.Template, target ApplyTarget) (map[string]any, error) {
	if target.SpaceID == nil {
		return nil, errors.New("space_id required")
	}
	if s.appliers.CreateList == nil || s.appliers.CreateStatus == nil || s.appliers.CreateTask == nil {
		return nil, errors.New("appliers not wired")
	}
	var snap listSnapshot
	_ = json.Unmarshal(t.Snapshot, &snap)
	name := snap.Name
	if name == "" {
		name = t.Name
	}
	list, err := s.appliers.CreateList(ctx, actor, *target.SpaceID, name)
	if err != nil {
		return nil, err
	}
	for i, st := range snap.Statuses {
		_ = s.appliers.CreateStatus(ctx, actor, list.ID, st.Name, st.Color, st.Category, i*10)
	}
	taskIDs := make([]uuid.UUID, 0, len(snap.Tasks))
	for _, tt := range snap.Tasks {
		task, err := s.appliers.CreateTask(ctx, actor, list.ID, nil, tt.Name, tt.Description)
		if err == nil && task != nil {
			taskIDs = append(taskIDs, task.ID)
		}
	}
	return map[string]any{"kind": "list", "list_id": list.ID, "task_ids": taskIDs}, nil
}

func (s *Service) applyTaskTemplate(ctx context.Context, actor uuid.UUID, t *domain.Template, target ApplyTarget) (map[string]any, error) {
	if target.ListID == nil {
		return nil, errors.New("list_id required")
	}
	if s.appliers.CreateTask == nil {
		return nil, errors.New("appliers not wired")
	}
	var snap taskSnapshot
	_ = json.Unmarshal(t.Snapshot, &snap)
	name := snap.Name
	if name == "" {
		name = t.Name
	}
	root, err := s.appliers.CreateTask(ctx, actor, *target.ListID, target.ParentTaskID, name, snap.Description)
	if err != nil {
		return nil, err
	}
	var createSubs func(parent *domain.Task, subs []taskSnapshot)
	createSubs = func(parent *domain.Task, subs []taskSnapshot) {
		for _, s2 := range subs {
			child, err := s.appliers.CreateTask(ctx, actor, parent.ListID, &parent.ID, s2.Name, s2.Description)
			if err == nil && child != nil {
				createSubs(child, s2.Subtasks)
			}
		}
	}
	createSubs(root, snap.Subtasks)
	return map[string]any{"kind": "task", "task_id": root.ID}, nil
}

func (s *Service) applyDocTemplate(ctx context.Context, actor uuid.UUID, t *domain.Template, target ApplyTarget) (map[string]any, error) {
	if target.WorkspaceID == nil {
		return nil, errors.New("workspace_id required")
	}
	if s.appliers.CreateDoc == nil {
		return nil, errors.New("appliers not wired")
	}
	var snap docSnapshot
	_ = json.Unmarshal(t.Snapshot, &snap)
	title := snap.Title
	if title == "" {
		title = t.Name
	}
	root, err := s.appliers.CreateDoc(ctx, actor, *target.WorkspaceID, target.ParentDocID, title, snap.Content)
	if err != nil {
		return nil, err
	}
	var createPages func(parent *domain.Doc, pages []docSnapshot)
	createPages = func(parent *domain.Doc, pages []docSnapshot) {
		for _, p := range pages {
			child, err := s.appliers.CreateDoc(ctx, actor, parent.WorkspaceID, &parent.ID, p.Title, p.Content)
			if err == nil && child != nil {
				createPages(child, p.Pages)
			}
		}
	}
	createPages(root, snap.Pages)
	return map[string]any{"kind": "doc", "doc_id": root.ID}, nil
}
