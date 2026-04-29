package credential

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"

	cryptox "github.com/yourorg/clickup/internal/crypto"
	"github.com/yourorg/clickup/internal/domain"
)

const (
	maxNameLen     = 200
	maxURLLen      = 2000
	maxUsernameLen = 200
	maxPasswordLen = 1000
	maxNotesLen    = 5000
)

var (
	ErrNameRequired     = errors.New("name required")
	ErrPasswordRequired = errors.New("password required")
	ErrFieldTooLong     = errors.New("field exceeds max length")
	ErrNotFound         = errors.New("not found")
	ErrForbidden        = errors.New("forbidden")
)

// WorkspaceAuthorizer abstracts the workspace membership check so this
// service does not import the concrete workspace service package.
type WorkspaceAuthorizer interface {
	EnsureMember(ctx context.Context, workspaceID, userID uuid.UUID) error
}

type Service struct {
	repo domain.CredentialRepo
	key  []byte
	ws   WorkspaceAuthorizer
}

func New(repo domain.CredentialRepo, key []byte, ws WorkspaceAuthorizer) *Service {
	return &Service{repo: repo, key: key, ws: ws}
}

type CreateInput struct {
	Name     string `json:"name"`
	URL      string `json:"url"`
	Username string `json:"username"`
	Password string `json:"password"`
	Notes    string `json:"notes"`
}

// UpdateInput uses pointer fields so callers can patch a subset. A nil pointer
// means "leave unchanged"; a non-nil one (even pointing at "") replaces the
// stored value.
type UpdateInput struct {
	Name     *string `json:"name,omitempty"`
	URL      *string `json:"url,omitempty"`
	Username *string `json:"username,omitempty"`
	Password *string `json:"password,omitempty"`
	Notes    *string `json:"notes,omitempty"`
}

func (s *Service) Create(ctx context.Context, workspaceID, userID uuid.UUID, in CreateInput) (*domain.Credential, error) {
	if err := s.ws.EnsureMember(ctx, workspaceID, userID); err != nil {
		return nil, ErrForbidden
	}

	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		return nil, ErrNameRequired
	}
	if in.Password == "" {
		return nil, ErrPasswordRequired
	}
	if err := validateLengths(in.Name, in.URL, in.Username, in.Password, in.Notes); err != nil {
		return nil, err
	}

	ct, nonce, err := cryptox.Encrypt(s.key, []byte(in.Password))
	if err != nil {
		return nil, err
	}

	c := &domain.Credential{
		WorkspaceID:        workspaceID,
		UserID:             userID,
		Name:               in.Name,
		URL:                in.URL,
		Username:           in.Username,
		PasswordCiphertext: ct,
		PasswordNonce:      nonce,
		Notes:              in.Notes,
	}
	if err := s.repo.Create(ctx, c); err != nil {
		return nil, err
	}
	// Never echo the plaintext back.
	c.Password = ""
	return c, nil
}

// List returns metadata only — never decrypts. Password stays empty.
func (s *Service) List(ctx context.Context, workspaceID, userID uuid.UUID) ([]domain.Credential, error) {
	if err := s.ws.EnsureMember(ctx, workspaceID, userID); err != nil {
		return nil, ErrForbidden
	}
	return s.repo.ListByUserInWorkspace(ctx, workspaceID, userID)
}

// Get fetches one entry and decrypts the password into Password.
func (s *Service) Get(ctx context.Context, workspaceID, userID, id uuid.UUID) (*domain.Credential, error) {
	if err := s.ws.EnsureMember(ctx, workspaceID, userID); err != nil {
		return nil, ErrForbidden
	}
	c, err := s.repo.GetByID(ctx, workspaceID, userID, id)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, ErrNotFound
	}
	pt, err := cryptox.Decrypt(s.key, c.PasswordCiphertext, c.PasswordNonce)
	if err != nil {
		return nil, err
	}
	c.Password = string(pt)
	return c, nil
}

func (s *Service) Update(ctx context.Context, workspaceID, userID, id uuid.UUID, in UpdateInput) (*domain.Credential, error) {
	if err := s.ws.EnsureMember(ctx, workspaceID, userID); err != nil {
		return nil, ErrForbidden
	}
	c, err := s.repo.GetByID(ctx, workspaceID, userID, id)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, ErrNotFound
	}

	if in.Name != nil {
		n := strings.TrimSpace(*in.Name)
		if n == "" {
			return nil, ErrNameRequired
		}
		c.Name = n
	}
	if in.URL != nil {
		c.URL = *in.URL
	}
	if in.Username != nil {
		c.Username = *in.Username
	}
	if in.Notes != nil {
		c.Notes = *in.Notes
	}
	if in.Password != nil {
		if *in.Password == "" {
			return nil, ErrPasswordRequired
		}
		ct, nonce, err := cryptox.Encrypt(s.key, []byte(*in.Password))
		if err != nil {
			return nil, err
		}
		c.PasswordCiphertext = ct
		c.PasswordNonce = nonce
	}

	if err := validateLengths(c.Name, c.URL, c.Username, "", c.Notes); err != nil {
		return nil, err
	}
	if in.Password != nil && len(*in.Password) > maxPasswordLen {
		return nil, ErrFieldTooLong
	}

	if err := s.repo.Update(ctx, c); err != nil {
		return nil, err
	}
	c.Password = ""
	return c, nil
}

func (s *Service) Delete(ctx context.Context, workspaceID, userID, id uuid.UUID) error {
	if err := s.ws.EnsureMember(ctx, workspaceID, userID); err != nil {
		return ErrForbidden
	}
	return s.repo.Delete(ctx, workspaceID, userID, id)
}

func validateLengths(name, url, username, password, notes string) error {
	if len(name) > maxNameLen ||
		len(url) > maxURLLen ||
		len(username) > maxUsernameLen ||
		len(password) > maxPasswordLen ||
		len(notes) > maxNotesLen {
		return ErrFieldTooLong
	}
	return nil
}
