// Package integration_test runs end-to-end tests against an ephemeral
// PostgreSQL container. The suite is opt-out: if Docker is unreachable
// (typical on a developer laptop with Docker Desktop off, or a CI runner
// without the docker-in-docker service) every test in the suite calls
// t.Skip and exits 0 — so the rest of the unit-test suite still runs.
//
// Run locally:
//
//	go test ./internal/integration_test/...
//
// CI:
//
//	go test -count=1 ./internal/integration_test/...
package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	userHandler "github.com/yourorg/clickup/internal/handler/user"
	"github.com/yourorg/clickup/internal/middleware"
	"github.com/yourorg/clickup/internal/migrate"
	userRepo "github.com/yourorg/clickup/internal/repository/user"
	userSvc "github.com/yourorg/clickup/internal/service/user"
)

// startPostgres returns a connected pgxpool against an ephemeral container
// with all migrations applied, plus a teardown function. If the host has no
// usable Docker daemon, the test calls t.Skip and the suite exits cleanly —
// it must not fail the build for missing infra.
func startPostgres(ctx context.Context, t *testing.T) (*pgxpool.Pool, func()) {
	t.Helper()

	c, err := tcpostgres.Run(ctx,
		"postgres:16-alpine",
		tcpostgres.WithDatabase("clickup_test"),
		tcpostgres.WithUsername("postgres"),
		tcpostgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		// testcontainers-go raises a recognizable error when Docker is
		// unavailable or unsupported; treat any of these as a skip rather
		// than a hard failure so the suite is friendly on Docker-less
		// laptops, rootless-Docker-on-Windows boxes, and CI runners that
		// don't ship Docker.
		msg := strings.ToLower(err.Error())
		switch {
		case strings.Contains(msg, "cannot connect to the docker daemon"),
			strings.Contains(msg, "docker daemon"),
			strings.Contains(msg, "rootless docker"),
			strings.Contains(msg, "failed to create docker provider"),
			strings.Contains(msg, "no such host"),
			strings.Contains(msg, "permission denied while trying to connect"),
			strings.Contains(msg, "is the docker daemon running"),
			errors.Is(err, context.DeadlineExceeded):
			t.Skipf("docker not available, skipping integration test: %v", err)
		}
		t.Fatalf("start postgres: %v", err)
	}

	dsn, err := c.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		_ = c.Terminate(ctx)
		t.Fatalf("dsn: %v", err)
	}

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		_ = c.Terminate(ctx)
		t.Fatalf("parse dsn: %v", err)
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		_ = c.Terminate(ctx)
		t.Fatalf("connect: %v", err)
	}

	if err := migrate.Up(ctx, pool, "../../migrations"); err != nil {
		pool.Close()
		_ = c.Terminate(ctx)
		t.Fatalf("migrate: %v", err)
	}

	return pool, func() {
		pool.Close()
		_ = c.Terminate(ctx)
	}
}

// TestAuthLoginMe is the smoke test of the auth surface: a user created
// via the admin-side `AdminCreate` path can log in via the public
// /auth/login endpoint, hit /me with the resulting JWT, and is rejected
// when the credentials are wrong.
//
// Self-registration (`POST /auth/register`) was removed — admins create
// users either through the seeder (production bootstrap) or through the
// member-invite flow once the first owner exists. This test mirrors that
// by calling `usersvc.AdminCreate` directly to set up the fixture user.
func TestAuthLoginMe(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	pool, teardown := startPostgres(ctx, t)
	defer teardown()

	const jwtSecret = "test-secret-do-not-use-in-prod"

	uRepo := userRepo.New(pool)
	svc := userSvc.New(uRepo, jwtSecret)
	h := userHandler.New(svc)

	// Admin-side fixture: the very same code path the seeder uses.
	if _, err := svc.AdminCreate(ctx, userSvc.AdminCreateInput{
		Email:    "e2e@example.com",
		Password: "correct-horse-battery",
		Name:     "E2E",
	}); err != nil {
		t.Fatalf("admin create: %v", err)
	}

	r := chi.NewRouter()
	r.Route("/api/v1", func(r chi.Router) {
		r.Group(func(pub chi.Router) { h.PublicRoutes(pub) })
		r.Group(func(pr chi.Router) {
			pr.Use(middleware.Auth(jwtSecret))
			h.PrivateRoutes(pr)
		})
	})
	srv := httptest.NewServer(r)
	defer srv.Close()

	// 1. /auth/register is gone — confirm it's a 404.
	resp := postJSON(t, srv.URL+"/api/v1/auth/register", "",
		`{"email":"x@x","password":"hunter22!","name":"X"}`)
	if resp.StatusCode != http.StatusNotFound && resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("register expected 404/405; got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// 2. Login with the seeded credentials
	resp = postJSON(t, srv.URL+"/api/v1/auth/login", "",
		`{"email":"e2e@example.com","password":"correct-horse-battery"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login status = %d; want 200", resp.StatusCode)
	}
	var login struct {
		Token string `json:"token"`
	}
	decode(t, resp, &login)
	if login.Token == "" {
		t.Fatalf("login token empty")
	}

	// 3. /me with the login token returns the user
	resp = getJSON(t, srv.URL+"/api/v1/me", login.Token)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("me status = %d; want 200", resp.StatusCode)
	}
	var me struct {
		Email string `json:"email"`
	}
	decode(t, resp, &me)
	if me.Email != "e2e@example.com" {
		t.Fatalf("me email = %q; want e2e@example.com", me.Email)
	}

	// 4. Wrong password is rejected
	resp = postJSON(t, srv.URL+"/api/v1/auth/login", "",
		`{"email":"e2e@example.com","password":"WRONG"}`)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("bad-password status = %d; want 401", resp.StatusCode)
	}

	// 5. /me without token is rejected
	resp = getJSON(t, srv.URL+"/api/v1/me", "")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("anon me status = %d; want 401", resp.StatusCode)
	}
}

func postJSON(t *testing.T, url, bearer, body string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	return resp
}

func getJSON(t *testing.T, url, bearer string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	return resp
}

func decode(t *testing.T, resp *http.Response, into any) {
	t.Helper()
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()
	if err := json.NewDecoder(resp.Body).Decode(into); err != nil {
		t.Fatalf("decode body: %v", err)
	}
}
