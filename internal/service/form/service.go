package form

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/yourorg/clickup/internal/audit"
	"github.com/yourorg/clickup/internal/domain"
)

// TaskCreator lets the public-submit path create a task without tying this
// package to task service. main.go wires this via a closure.
type TaskCreator func(ctx context.Context, listID uuid.UUID, name, description string) (*domain.Task, error)

type Service struct {
	repo   domain.FormRepo
	create TaskCreator
	audit  *audit.Recorder
}

func New(repo domain.FormRepo, create TaskCreator, rec *audit.Recorder) *Service {
	return &Service{repo: repo, create: create, audit: rec}
}

type CreateInput struct {
	ListID      uuid.UUID       `json:"list_id"`
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Fields      json.RawMessage `json:"fields,omitempty"`
	IsPublic    bool            `json:"is_public,omitempty"`
}

type UpdateInput struct {
	Name        *string         `json:"name,omitempty"`
	Description *string         `json:"description,omitempty"`
	Fields      json.RawMessage `json:"fields,omitempty"`
	IsPublic    *bool           `json:"is_public,omitempty"`
}

func (s *Service) Create(ctx context.Context, actor uuid.UUID, in CreateInput) (*domain.Form, error) {
	if strings.TrimSpace(in.Name) == "" {
		return nil, errors.New("name required")
	}
	f := &domain.Form{
		ListID:      in.ListID,
		Name:        strings.TrimSpace(in.Name),
		Description: in.Description,
		Fields:      in.Fields,
		IsPublic:    in.IsPublic,
		CreatorID:   &actor,
	}
	if err := s.repo.Create(ctx, f); err != nil {
		return nil, err
	}
	return f, nil
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*domain.Form, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) ListByList(ctx context.Context, listID uuid.UUID) ([]domain.Form, error) {
	return s.repo.ListByList(ctx, listID)
}

func (s *Service) Update(ctx context.Context, actor, id uuid.UUID, in UpdateInput) (*domain.Form, error) {
	f, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if f == nil {
		return nil, errors.New("not found")
	}
	if in.Name != nil {
		f.Name = *in.Name
	}
	if in.Description != nil {
		f.Description = *in.Description
	}
	if len(in.Fields) > 0 {
		f.Fields = in.Fields
	}
	if in.IsPublic != nil {
		f.IsPublic = *in.IsPublic
	}
	if err := s.repo.Update(ctx, f); err != nil {
		return nil, err
	}
	return f, nil
}

func (s *Service) Delete(ctx context.Context, actor, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

func (s *Service) ListSubmissions(ctx context.Context, id uuid.UUID, limit int) ([]domain.FormSubmission, error) {
	return s.repo.ListSubmissions(ctx, id, limit)
}

/* --- public intake ------------------------------------------------------- */

type fieldDef struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Kind     string `json:"kind"`
	Required bool   `json:"required"`
}

// Submit is the public endpoint. Validates required fields, renders the task
// title+description from the submission, creates a task in the linked list,
// and records the submission row.
func (s *Service) Submit(ctx context.Context, formID uuid.UUID, payload map[string]any, submitterIP string) (*domain.FormSubmission, error) {
	f, err := s.repo.GetByID(ctx, formID)
	if err != nil {
		return nil, err
	}
	if f == nil {
		return nil, errors.New("form not found")
	}
	if !f.IsPublic {
		return nil, errors.New("form is not public")
	}

	var fields []fieldDef
	if err := json.Unmarshal(f.Fields, &fields); err != nil {
		return nil, errors.New("form fields malformed")
	}
	// Required-field validation.
	for _, fd := range fields {
		if !fd.Required {
			continue
		}
		v, ok := payload[fd.ID]
		if !ok || isEmpty(v) {
			return nil, fmt.Errorf("required field missing: %s", fd.Label)
		}
	}

	title, desc := renderTask(f, fields, payload)
	var taskID *uuid.UUID
	if s.create != nil {
		t, err := s.create(ctx, f.ListID, title, desc)
		if err == nil && t != nil {
			taskID = &t.ID
		}
	}

	pb, _ := json.Marshal(payload)
	sub := &domain.FormSubmission{
		FormID:      formID,
		TaskID:      taskID,
		Payload:     pb,
		SubmitterIP: submitterIP,
	}
	if err := s.repo.RecordSubmission(ctx, sub); err != nil {
		return nil, err
	}
	if s.audit != nil {
		// FormSubmission has no workspace — audit at the form entity.
		s.audit.Record(ctx, audit.Entry{
			EntityType: "form",
			EntityID:   &formID,
			Verb:       "submitted",
			After:      map[string]any{"task_id": taskID},
		})
	}
	return sub, nil
}

// renderTask produces the new task's title + description from a submission.
// The first text-ish field becomes the title; remaining fields are listed in
// the description as a "label: value" block.
func renderTask(f *domain.Form, fields []fieldDef, payload map[string]any) (string, string) {
	title := f.Name + " submission"
	if len(fields) > 0 {
		if v, ok := payload[fields[0].ID]; ok && !isEmpty(v) {
			title = stringify(v)
		}
	}
	var sb strings.Builder
	if f.Description != "" {
		sb.WriteString(f.Description)
		sb.WriteString("\n\n")
	}
	for _, fd := range fields {
		v, ok := payload[fd.ID]
		if !ok {
			continue
		}
		sb.WriteString(fd.Label)
		sb.WriteString(": ")
		sb.WriteString(stringify(v))
		sb.WriteString("\n")
	}
	return title, sb.String()
}

func isEmpty(v any) bool {
	if v == nil {
		return true
	}
	switch x := v.(type) {
	case string:
		return strings.TrimSpace(x) == ""
	case []any:
		return len(x) == 0
	}
	return false
}

func stringify(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case []any:
		parts := make([]string, 0, len(x))
		for _, item := range x {
			parts = append(parts, stringify(item))
		}
		return strings.Join(parts, ", ")
	default:
		b, _ := json.Marshal(v)
		return string(b)
	}
}
