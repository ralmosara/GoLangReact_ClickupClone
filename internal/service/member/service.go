package member

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/yourorg/clickup/internal/audit"
	"github.com/yourorg/clickup/internal/domain"
	"github.com/yourorg/clickup/internal/ws"
)

// ErrUserNotFound is returned by AddWorkspaceMember when the supplied
// email doesn't map to a registered user. The handler translates this to
// a 404 with a body that points the admin at the User Management page,
// which is the only path to create new accounts now that self-registration
// and member-create-on-missing are both gone.
var ErrUserNotFound = errors.New("no user with that email — create them in User Management first")

type Service struct {
	repo  domain.MemberRepo
	users domain.UserRepo
	hub   *ws.Hub
	audit *audit.Recorder
}

func New(repo domain.MemberRepo, users domain.UserRepo, hub *ws.Hub, rec *audit.Recorder) *Service {
	return &Service{repo: repo, users: users, hub: hub, audit: rec}
}

// InviteLookup is the response shape for the pre-invite lookup endpoint.
// It tells the inviting admin whether the email maps to an existing local
// account, and if so whether that user is already a member of the
// workspace they're inviting into. The frontend uses it to adapt the
// invite form: hide the password field for existing users, show a
// "already a member" hint when applicable.
type InviteLookup struct {
	Exists        bool   `json:"exists"`
	Name          string `json:"name,omitempty"`
	AlreadyMember bool   `json:"already_member"`
}

// LookupForInvite resolves an email to (account_exists, is_already_member)
// in the context of a workspace. The auth gate (member.invite on the
// workspace) is enforced at the handler layer; an admin can already learn
// the same information by trying to invite, so this endpoint only makes
// the existing oracle explicit.
func (s *Service) LookupForInvite(ctx context.Context, wsID uuid.UUID, email string) (*InviteLookup, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return nil, errors.New("email is required")
	}
	u, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return &InviteLookup{Exists: false}, nil
	}
	isMember, err := s.repo.IsWorkspaceMember(ctx, wsID, u.ID)
	if err != nil {
		return nil, err
	}
	return &InviteLookup{Exists: true, Name: u.Name, AlreadyMember: isMember}, nil
}

// AddInput is the request body for the workspace invite endpoint.
// Only Email and Role are honoured — user creation has moved to the User
// Management page, so this endpoint is strictly invite-existing.
type AddInput struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

// --- workspace ---------------------------------------------------------------

func (s *Service) AddWorkspaceMember(ctx context.Context, actor, wsID uuid.UUID, in AddInput) (*domain.Member, error) {
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	if in.Email == "" {
		return nil, errors.New("email is required")
	}
	u, err := s.users.GetByEmail(ctx, in.Email)
	if err != nil {
		return nil, err
	}
	if u == nil {
		// Strict invite-existing only. Account creation has moved to the
		// User Management page; the handler turns this into a 404 with a
		// body that points the admin there.
		return nil, ErrUserNotFound
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

// AddSpaceMember keeps the existing "user must already exist" semantics —
// space-level invites are within an existing workspace, so the user has
// already been created by the workspace-level admin.
func (s *Service) AddSpaceMember(ctx context.Context, actor, spaceID uuid.UUID, in AddInput) (*domain.Member, error) {
	u, err := s.users.GetByEmail(ctx, in.Email)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, errors.New("no user with that email — invite them to the workspace first")
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
