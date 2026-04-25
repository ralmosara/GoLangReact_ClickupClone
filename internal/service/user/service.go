package user

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/yourorg/clickup/internal/domain"
	"github.com/yourorg/clickup/internal/middleware"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrEmailTaken         = errors.New("email already registered")
	ErrWeakPassword       = errors.New("password must be at least 8 characters")
)

type Service struct {
	users     domain.UserRepo
	jwtSecret string
	tokenTTL  time.Duration
}

func New(users domain.UserRepo, jwtSecret string) *Service {
	return &Service{users: users, jwtSecret: jwtSecret, tokenTTL: 7 * 24 * time.Hour}
}

type RegisterInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthResult struct {
	Token string       `json:"token"`
	User  *domain.User `json:"user"`
}

func (s *Service) Register(ctx context.Context, in RegisterInput) (*AuthResult, error) {
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	if len(in.Password) < 8 {
		return nil, ErrWeakPassword
	}
	existing, err := s.users.GetByEmail(ctx, in.Email)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrEmailTaken
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	u := &domain.User{Email: in.Email, PasswordHash: string(hash), Name: in.Name}
	if err := s.users.Create(ctx, u); err != nil {
		return nil, err
	}
	token, err := middleware.IssueToken(s.jwtSecret, u.ID, s.tokenTTL)
	if err != nil {
		return nil, err
	}
	return &AuthResult{Token: token, User: u}, nil
}

func (s *Service) Login(ctx context.Context, in LoginInput) (*AuthResult, error) {
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	u, err := s.users.GetByEmail(ctx, in.Email)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(in.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}
	token, err := middleware.IssueToken(s.jwtSecret, u.ID, s.tokenTTL)
	if err != nil {
		return nil, err
	}
	return &AuthResult{Token: token, User: u}, nil
}

func (s *Service) Me(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	return s.users.GetByID(ctx, id)
}
