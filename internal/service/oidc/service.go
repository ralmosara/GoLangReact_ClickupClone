// Package oidc implements OpenID-Connect login. The current scope is one
// provider — Google — wired by env vars (GOOGLE_OIDC_CLIENT_ID,
// GOOGLE_OIDC_CLIENT_SECRET, GOOGLE_OIDC_REDIRECT_URL). The code is shaped
// so adding Microsoft Entra / Okta-OIDC is a config-only change: each new
// provider is one entry in providerSpecs with its discovery URL.
//
// Flow (as exposed by the handler):
//
//   GET  /api/v1/auth/oidc/{provider}/login    → 302 to IdP authorize URL
//                                                 with random state cookie
//   GET  /api/v1/auth/oidc/{provider}/callback → exchange code, fetch
//                                                 userinfo, find/create
//                                                 user, redirect to the
//                                                 frontend with the
//                                                 session JWT in the
//                                                 fragment ("#token=...")
package oidc

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	coreoidc "github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	"github.com/yourorg/clickup/internal/domain"
	"github.com/yourorg/clickup/internal/middleware"
)

// ProviderConfig is the per-provider env shape. Issuer is the discovery URL
// (e.g. https://accounts.google.com); the rest are standard OAuth2.
type ProviderConfig struct {
	Name         string
	Issuer       string
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string // defaults: openid email profile
}

// Service is the entry point for handlers. It memoises *coreoidc.Provider
// per-Issuer so we don't hammer the discovery endpoint on every login.
type Service struct {
	users     domain.UserRepo
	idents    domain.OAuthIdentityRepo
	jwtSecret string
	tokenTTL  time.Duration
	configs   map[string]ProviderConfig

	mu        sync.Mutex
	providers map[string]*coreoidc.Provider
}

func New(users domain.UserRepo, idents domain.OAuthIdentityRepo, jwtSecret string, providers []ProviderConfig) *Service {
	cfg := make(map[string]ProviderConfig, len(providers))
	for _, p := range providers {
		if p.Name == "" || p.Issuer == "" || p.ClientID == "" || p.RedirectURL == "" {
			continue
		}
		if len(p.Scopes) == 0 {
			p.Scopes = []string{coreoidc.ScopeOpenID, "email", "profile"}
		}
		cfg[strings.ToLower(p.Name)] = p
	}
	return &Service{
		users:     users,
		idents:    idents,
		jwtSecret: jwtSecret,
		tokenTTL:  7 * 24 * time.Hour,
		configs:   cfg,
		providers: make(map[string]*coreoidc.Provider),
	}
}

// Configured reports whether the named provider is enabled (env vars set).
// The login handler returns 404 for unconfigured providers so an attacker
// can't probe which IdPs we're set up against.
func (s *Service) Configured(name string) bool {
	_, ok := s.configs[strings.ToLower(name)]
	return ok
}

func (s *Service) ConfiguredProviders() []string {
	out := make([]string, 0, len(s.configs))
	for name := range s.configs {
		out = append(out, name)
	}
	return out
}

// AuthorizeURL builds the IdP redirect URL and returns the random `state`
// nonce. The handler stores `state` in a short-lived signed cookie and
// verifies it in the callback to defeat CSRF on the OIDC redirect.
func (s *Service) AuthorizeURL(ctx context.Context, providerName string) (url string, state string, err error) {
	cfg, ok := s.configs[strings.ToLower(providerName)]
	if !ok {
		return "", "", fmt.Errorf("oidc: provider %q not configured", providerName)
	}
	prov, err := s.provider(ctx, cfg)
	if err != nil {
		return "", "", err
	}
	state, err = randomHex(16)
	if err != nil {
		return "", "", err
	}
	oa := s.oauthConfig(prov, cfg)
	return oa.AuthCodeURL(state, oauth2.AccessTypeOnline), state, nil
}

type CallbackResult struct {
	Token string
	// JTI is the JWT ID stamped into Token. Surfaced so the handler
	// can record a session row keyed by it (so OIDC logins also show
	// up in the Active Sessions UI).
	JTI       string
	ExpiresAt time.Time
	User      *domain.User
}

// Callback exchanges the authorization code, validates the ID token, and
// resolves it to one of our users (creating one on first sight). It
// returns a session JWT that the handler hands to the browser.
func (s *Service) Callback(ctx context.Context, providerName, code string) (*CallbackResult, error) {
	cfg, ok := s.configs[strings.ToLower(providerName)]
	if !ok {
		return nil, fmt.Errorf("oidc: provider %q not configured", providerName)
	}
	prov, err := s.provider(ctx, cfg)
	if err != nil {
		return nil, err
	}
	oa := s.oauthConfig(prov, cfg)

	tok, err := oa.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("oidc: code exchange failed: %w", err)
	}
	rawIDToken, ok := tok.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		return nil, errors.New("oidc: id_token missing from token response")
	}
	verifier := prov.Verifier(&coreoidc.Config{ClientID: cfg.ClientID})
	idToken, err := verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, fmt.Errorf("oidc: id_token verify failed: %w", err)
	}
	var claims struct {
		Sub           string `json:"sub"`
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		Name          string `json:"name"`
		Picture       string `json:"picture"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return nil, fmt.Errorf("oidc: claims parse failed: %w", err)
	}
	if claims.Sub == "" {
		return nil, errors.New("oidc: id_token has no subject")
	}
	if claims.Email == "" {
		return nil, errors.New("oidc: id_token has no email")
	}
	// Refuse logins for unverified emails — otherwise an attacker who
	// controls foo@example.com via a misconfigured IdP could squat on
	// account-takeover by registering with that email server-side first.
	if !claims.EmailVerified {
		return nil, errors.New("oidc: email not verified by IdP")
	}

	// 1) If we already linked this (provider, sub) pair, just log in.
	existing, err := s.idents.GetByProviderSubject(ctx, cfg.Name, claims.Sub)
	if err != nil {
		return nil, err
	}
	var u *domain.User
	if existing != nil {
		u, err = s.users.GetByID(ctx, existing.UserID)
		if err != nil {
			return nil, err
		}
	}
	// 2) Otherwise look the user up by email — link the IdP to the
	//    existing local account (e.g. user originally signed up with
	//    password, now wants Google sign-in for the same account).
	if u == nil {
		u, err = s.users.GetByEmail(ctx, strings.ToLower(claims.Email))
		if err != nil {
			return nil, err
		}
	}
	// 3) Otherwise JIT-create a new user. Password hash is set to a
	//    random sentinel so password login can't be used to access an
	//    SSO-only account.
	if u == nil {
		u = &domain.User{
			Email:        strings.ToLower(claims.Email),
			Name:         claims.Name,
			PasswordHash: "!" + mustRandHex(32), // unmatchable bcrypt input
		}
		if err := s.users.Create(ctx, u); err != nil {
			return nil, err
		}
	}

	// 4) Persist (or refresh) the IdP linkage.
	rawProfile, _ := json.Marshal(map[string]any{
		"sub":     claims.Sub,
		"email":   claims.Email,
		"name":    claims.Name,
		"picture": claims.Picture,
	})
	emailCopy := claims.Email
	ident := &domain.OAuthIdentity{
		UserID:     u.ID,
		Provider:   cfg.Name,
		Subject:    claims.Sub,
		Email:      &emailCopy,
		RawProfile: rawProfile,
	}
	if err := s.idents.Upsert(ctx, ident); err != nil {
		return nil, err
	}

	jwtTok, jti, err := middleware.IssueToken(s.jwtSecret, u.ID, s.tokenTTL)
	if err != nil {
		return nil, err
	}
	return &CallbackResult{
		Token:     jwtTok,
		JTI:       jti.String(),
		ExpiresAt: time.Now().Add(s.tokenTTL),
		User:      u,
	}, nil
}

// provider memoises the discovery roundtrip. *coreoidc.Provider is safe for
// concurrent use after construction so we keep one per issuer for the
// process lifetime.
func (s *Service) provider(ctx context.Context, cfg ProviderConfig) (*coreoidc.Provider, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if p, ok := s.providers[cfg.Issuer]; ok {
		return p, nil
	}
	p, err := coreoidc.NewProvider(ctx, cfg.Issuer)
	if err != nil {
		return nil, err
	}
	s.providers[cfg.Issuer] = p
	return p, nil
}

func (s *Service) oauthConfig(prov *coreoidc.Provider, cfg ProviderConfig) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  cfg.RedirectURL,
		Endpoint:     prov.Endpoint(),
		Scopes:       cfg.Scopes,
	}
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func mustRandHex(n int) string {
	s, err := randomHex(n)
	if err != nil {
		// genuinely catastrophic — out-of-entropy on a kernel RNG read
		// is the kind of thing only a hard-broken host produces.
		panic(err)
	}
	return s
}
