package domain

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ViewKind enumerates the supported view types. The frontend picks the renderer.
const (
	ViewKindBoard    = "board"
	ViewKindList     = "list"
	ViewKindCalendar = "calendar"
	ViewKindGantt    = "gantt"
	ViewKindTable    = "table"
	ViewKindTimeline = "timeline"
)

// Filter is one clause of a view's filter expression. Frontend applies them;
// the backend just stores the JSON so views are portable across view types.
type Filter struct {
	Field string      `json:"field"` // status_id | assignee_id | priority | tag_id | due_at | custom:<field_id>
	Op    string      `json:"op"`    // eq | neq | in | nin | gt | lt | between | empty | notempty
	Value interface{} `json:"value,omitempty"`
}

type SortField struct {
	Field string `json:"field"`
	Desc  bool   `json:"desc,omitempty"`
}

// ViewConfig is the typed structure serialized into the views.config JSONB column.
type ViewConfig struct {
	Filters     []Filter    `json:"filters,omitempty"`
	Sort        []SortField `json:"sort,omitempty"`
	GroupBy     string      `json:"group_by,omitempty"`
	VisibleCols []string    `json:"visible_cols,omitempty"`
	IsShared    bool        `json:"is_shared"`
	CreatorID   *uuid.UUID  `json:"creator_id,omitempty"`
	// Free-form bucket for view-type-specific knobs (calendar start day, gantt zoom, etc.)
	Options map[string]any `json:"options,omitempty"`
}

type View struct {
	ID        uuid.UUID       `json:"id"`
	ListID    *uuid.UUID      `json:"list_id,omitempty"`
	SpaceID   *uuid.UUID      `json:"space_id,omitempty"`
	Name      string          `json:"name"`
	Kind      string          `json:"kind"`
	Config    json.RawMessage `json:"config"`
	CreatedAt time.Time       `json:"created_at"`
}

// DecodeConfig is a convenience used by services wanting to read/validate config.
func (v *View) DecodeConfig() (ViewConfig, error) {
	var c ViewConfig
	if len(v.Config) == 0 || string(v.Config) == "null" {
		return c, nil
	}
	err := json.Unmarshal(v.Config, &c)
	return c, err
}

type ViewRepo interface {
	Create(ctx context.Context, v *View) error
	GetByID(ctx context.Context, id uuid.UUID) (*View, error)
	ListByList(ctx context.Context, listID uuid.UUID) ([]View, error)
	ListBySpace(ctx context.Context, spaceID uuid.UUID) ([]View, error)
	Update(ctx context.Context, v *View) error
	Delete(ctx context.Context, id uuid.UUID) error
}
