package member

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

type Service struct {
	repo  domain.MemberRepo
	users domain.UserRepo
	hub   *ws.Hub
	audit *audit.Recorder
}

func New(repo domain.MemberRepo, users domain.UserRepo, hub *ws.Hub, rec *audit.Recorder) *Service {
	return &Service{repo: repo, users: users, hub: hub, audit: rec}
}

type AddInput struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

// --- workspace ---------------------------------------------------------------

func (s *Service) AddWorkspaceMember(ctx context.Context, actor, wsID uuid.UUID, in AddInput) (*domain.Member, error) {
	u, err := s.users.GetByEmail(ctx, in.Email)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, errors.New("no user with that email — invite flow pending")
	}
	if err := s.repo.AddWorkspaceMember(ctx, wsID, u.ID, in.Role); err != nil {
		return nil, err
	}
	m := &domain.Member{
		WorkspaceID: &wsID,
		UserID:      u.ID,
		Email:       u.Email,
		Name:        u.Name,
		AvatarURL:   u.AvatarURL,
		Role:        in.Role,
		CreatedAt:   time.Now(),
	}
	s.publishWorkspace(wsID, ws.EventMemberAdded, actor, m)
	if s.audit != nil {
		s.audit.Record(ctx, audit.Entry{
			WorkspaceID: &wsID,
			ActorID:     &actor,
			EntityType:  "workspace_member",
			EntityID:    &u.ID,
			Verb:        "added",
			After:       m,
		})
	}
	return m, nil
}

func (s *Service) RemoveWorkspaceMember(ctx context.Context, actor, wsID, userID uuid.UUID) error {
	if err := s.repo.RemoveWorkspaceMember(ctx, wsID, userID); err != nil {
		return err
	}
	s.publishWorkspace(wsID, ws.EventMemberRemoved, actor, map[string]uuid.UUID{"user_id": userID})
	if s.audit != nil {
		s.audit.Record(ctx, audit.Entry{
			WorkspaceID: &wsID,
			ActorID:     &actor,
			EntityType:  "workspace_member",
			EntityID:    &userID,
			Verb:        "removed",
		})
	}
	return nil
}

func (s *Service) ListWorkspaceMembers(ctx context.Context, wsID uuid.UUID) ([]domain.Member, error) {
	return s.repo.ListWorkspaceMembers(ctx, wsID)
}

// --- space -------------------------------------------------------------------

func (s *Service) AddSpaceMember(ctx context.Context, actor, spaceID uuid.UUID, in AddInput) (*domain.Member, error) {
	u, err := s.users.GetByEmail(ctx, in.Email)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, errors.New("no user with that email")
	}
	if err := s.repo.AddSpaceMember(ctx, spaceID, u.ID, in.Role); err != nil {
		return nil, err
	}
	m := &domain.Member{
		SpaceID:   &spaceID,
		UserID:    u.ID,
		Email:     u.Email,
		Name:      u.Name,
		AvatarURL: u.AvatarURL,
		Role:      in.Role,
		CreatedAt: time.Now(),
	}
	if s.audit != nil {
		s.audit.Record(ctx, audit.Entry{
			ActorID:    &actor,
			EntityType: "space_member",
			EntityID:   &u.ID,
			Verb:       "added",
			After:      m,
		})
	}
	return m, nil
}

func (s *Service) RemoveSpaceMember(ctx context.Context, actor, spaceID, userID uuid.UUID) error {
	if err := s.repo.RemoveSpaceMember(ctx, spaceID, userID); err != nil {
		return err
	}
	if s.audit != nil {
		s.audit.Record(ctx, audit.Entry{
			ActorID:    &actor,
			EntityType: "space_member",
			EntityID:   &userID,
			Verb:       "removed",
		})
	}
	return nil
}

func (s *Service) ListSpaceMembers(ctx context.Context, spaceID uuid.UUID) ([]domain.Member, error) {
	return s.repo.ListSpaceMembers(ctx, spaceID)
}

func (s *Service) publishWorkspace(wsID uuid.UUID, kind string, actor uuid.UUID, payload any) {
	if s.hub == nil {
		return
	}
	p, _ := json.Marshal(payload)
	s.hub.Publish(ws.Event{
		V:           1,
		Room:        ws.RoomWorkspace(wsID.String()),
		Type:        kind,
		WorkspaceID: wsID.String(),
		ActorID:     actor.String(),
		TS:          time.Now().UTC(),
		Payload:     p,
	})
}
