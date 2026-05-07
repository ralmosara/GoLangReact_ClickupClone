package task

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/yourorg/clickup/internal/domain"
)

type mockRepo struct {
	domain.TaskRepo
	tasks map[uuid.UUID]*domain.Task
}

func (m *mockRepo) Create(ctx context.Context, t *domain.Task) error {
	t.ID = uuid.New()
	m.tasks[t.ID] = t
	return nil
}

func (m *mockRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Task, error) {
	return m.tasks[id], nil
}

func TestCreateTask(t *testing.T) {
	repo := &mockRepo{tasks: make(map[uuid.UUID]*domain.Task)}
	svc := New(Deps{Tasks: repo})

	ctx := context.Background()
	creator := uuid.New()
	listID := uuid.New()

	input := CreateInput{
		ListID: listID,
		Name:   "Test Task",
	}

	task, err := svc.Create(ctx, creator, input)

	assert.NoError(t, err)
	assert.NotNil(t, task)
	assert.Equal(t, "Test Task", task.Name)
	assert.Equal(t, listID, task.ListID)
	assert.Equal(t, &creator, task.CreatorID)
}

func TestCreateTask_NoName(t *testing.T) {
	repo := &mockRepo{tasks: make(map[uuid.UUID]*domain.Task)}
	svc := New(Deps{Tasks: repo})

	ctx := context.Background()
	creator := uuid.New()

	input := CreateInput{
		Name: "",
	}

	task, err := svc.Create(ctx, creator, input)

	assert.Error(t, err)
	assert.Nil(t, task)
	assert.Contains(t, err.Error(), "name required")
}
