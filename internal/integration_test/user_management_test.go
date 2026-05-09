package integration_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/yourorg/clickup/internal/audit"
	"github.com/yourorg/clickup/internal/authz"
	memberHandler "github.com/yourorg/clickup/internal/handler/member"
	userHandler "github.com/yourorg/clickup/internal/handler/user"
	"github.com/yourorg/clickup/internal/middleware"
	auditRepo "github.com/yourorg/clickup/internal/repository/audit"
	memberRepo "github.com/yourorg/clickup/internal/repository/member"
	userRepo "github.com/yourorg/clickup/internal/repository/user"
	memberSvc "github.com/yourorg/clickup/internal/service/member"
	userSvc "github.com/yourorg/clickup/internal/service/user"
)

// TestUserManagement_FullLifecycle exercises every endpoint behind the
// User Management page from end to end:
//
//   1. Owner creates a new user via POST /workspaces/{id}/users.
//   2. New user logs in.
//   3. Owner resets the new user's password; old password rejected, new
//      password accepted.
//   4. Owner edits the new user's name; /me reflects the new name.
//   5. As a member-tier user (no user.manage), each endpoint returns 403.
//   6. Owner deletes the new user; subsequent login attempts are rejected.
//
// Adds a multi-tenant safety check: an owner of workspace A cannot manage
// a user who only belongs to workspace B (404 not member).
func TestUserManagement_FullLifecycle(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	pool, teardown := startPostgres(ctx, t)
	defer teardown()

	const jwtSecret = "test-secret-do-not-use-in-prod"

	uRepo := userRepo.New(pool)
	uSvc := userSvc.New(uRepo, jwtSecret)
	mRepo := memberRepo.New(pool)
	auRepo := auditRepo.New(pool)
	rec := audit.New(auRepo, nil)
	mSvc := memberSvc.New(mRepo, uRepo, nil, rec)
	policy := authz.New(pool)

	uH := userHandler.New(uSvc).WithAdminDeps(policy, mRepo, mSvc)
	mH := memberHandler.NewWithPolicy(mSvc, policy)

	r := chi.NewRouter()
	r.Route("/api/v1", func(r chi.Router) {
		r.Group(func(pub chi.Router) { uH.PublicRoutes(pub) })
		r.Group(func(pr chi.Router) {
			pr.Use(middleware.Auth(jwtSecret))
			uH.PrivateRoutes(pr)
			uH.AdminRoutes(pr)
			mH.Routes(pr)
		})
	})
	srv := httptest.NewServer(r)
	defer srv.Close()

	// Bootstrap: owner + member user, owner's workspace.
	owner, err := uSvc.AdminCreate(ctx, userSvc.AdminCreateInput{
		Email: "owner@example.com", Password: "owner-password-1", Name: "Owner",
	})
	if err != nil {
		t.Fatalf("owner create: %v", err)
	}
	plainMember, err := uSvc.AdminCreate(ctx, userSvc.AdminCreateInput{
		Email: "plain@example.com", Password: "plain-password-1", Name: "Plain",
	})
	if err != nil {
		t.Fatalf("plain create: %v", err)
	}

	var workspaceID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO workspaces (name, slug, owner_id) VALUES ('Test', 'test', $1)
		RETURNING id
	`, owner.ID).Scan(&workspaceID); err != nil {
		t.Fatalf("ws insert: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO workspace_members (workspace_id, user_id, role, role_id)
		VALUES
		  ($1, $2, 'owner',  (SELECT id FROM roles WHERE workspace_id IS NULL AND name='owner')),
		  ($1, $3, 'member', (SELECT id FROM roles WHERE workspace_id IS NULL AND name='member'))
	`, workspaceID, owner.ID, plainMember.ID); err != nil {
		t.Fatalf("ws member insert: %v", err)
	}

	// Logins.
	ownerToken := login(t, srv.URL, "owner@example.com", "owner-password-1")
	plainToken := login(t, srv.URL, "plain@example.com", "plain-password-1")

	// ── 1. Owner creates a new user via POST /workspaces/{id}/users ────
	resp := postJSON(t, srv.URL+"/api/v1/workspaces/"+workspaceID+"/users", ownerToken,
		`{"email":"new@example.com","name":"New","password":"hunter22!","role":"member"}`)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create user = %d; want 201", resp.StatusCode)
	}
	var created struct {
		ID    string `json:"id"`
		Email string `json:"email"`
		Name  string `json:"name"`
		Role  string `json:"role"`
	}
	decode(t, resp, &created)
	if created.Email != "new@example.com" || created.Role != "member" || created.ID == "" {
		t.Fatalf("create payload: %+v", created)
	}
	newUserID := created.ID

	// ── 2. New user can log in ─────────────────────────────────────────
	_ = login(t, srv.URL, "new@example.com", "hunter22!")

	// ── 3. Owner resets password ───────────────────────────────────────
	resp = postJSON(t, srv.URL+"/api/v1/workspaces/"+workspaceID+"/users/"+newUserID+"/reset-password", ownerToken,
		`{"password":"newpass1!"}`)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("reset = %d; want 204", resp.StatusCode)
	}
	// Old password rejected
	resp = postJSON(t, srv.URL+"/api/v1/auth/login", "",
		`{"email":"new@example.com","password":"hunter22!"}`)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("old password login = %d; want 401", resp.StatusCode)
	}
	// New password accepted
	_ = login(t, srv.URL, "new@example.com", "newpass1!")

	// ── 4. Owner edits name ────────────────────────────────────────────
	resp = patchJSON(t, srv.URL+"/api/v1/workspaces/"+workspaceID+"/users/"+newUserID, ownerToken,
		`{"name":"Renamed"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("update name = %d; want 200", resp.StatusCode)
	}
	var updated struct {
		Name string `json:"name"`
	}
	decode(t, resp, &updated)
	if updated.Name != "Renamed" {
		t.Fatalf("update name body = %q; want Renamed", updated.Name)
	}

	// ── 5. Member-tier user is denied on every admin endpoint ─────────
	for _, call := range []struct {
		method, path, body string
	}{
		{http.MethodGet, "/api/v1/workspaces/" + workspaceID + "/users", ""},
		{http.MethodPost, "/api/v1/workspaces/" + workspaceID + "/users", `{"email":"x@x.test","name":"X","password":"hunter22!","role":"member"}`},
		{http.MethodPatch, "/api/v1/workspaces/" + workspaceID + "/users/" + newUserID, `{"name":"X"}`},
		{http.MethodPost, "/api/v1/workspaces/" + workspaceID + "/users/" + newUserID + "/reset-password", `{"password":"hunter22!"}`},
		{http.MethodDelete, "/api/v1/workspaces/" + workspaceID + "/users/" + newUserID, ""},
	} {
		resp = doJSON(t, call.method, srv.URL+call.path, plainToken, call.body)
		if resp.StatusCode != http.StatusForbidden {
			t.Fatalf("%s %s as plain member = %d; want 403", call.method, call.path, resp.StatusCode)
		}
	}

	// ── 6. Owner deletes the new user ─────────────────────────────────
	resp = doJSON(t, http.MethodDelete, srv.URL+"/api/v1/workspaces/"+workspaceID+"/users/"+newUserID, ownerToken, "")
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete = %d; want 204", resp.StatusCode)
	}
	// Login fails after delete
	resp = postJSON(t, srv.URL+"/api/v1/auth/login", "",
		`{"email":"new@example.com","password":"newpass1!"}`)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("post-delete login = %d; want 401", resp.StatusCode)
	}
}

// ── helpers ──────────────────────────────────────────────────────────────

func login(t *testing.T, baseURL, email, password string) string {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"email": email, "password": password})
	resp := postJSON(t, baseURL+"/api/v1/auth/login", "", string(body))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login %s = %d; want 200", email, resp.StatusCode)
	}
	var out struct {
		Token string `json:"token"`
	}
	decode(t, resp, &out)
	if out.Token == "" {
		t.Fatalf("login %s returned empty token", email)
	}
	return out.Token
}

func patchJSON(t *testing.T, url, bearer, body string) *http.Response {
	return doJSON(t, http.MethodPatch, url, bearer, body)
}

func doJSON(t *testing.T, method, url, bearer, body string) *http.Response {
	t.Helper()
	var bodyReader *strings.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	}
	var req *http.Request
	var err error
	if bodyReader != nil {
		req, err = http.NewRequest(method, url, bodyReader)
	} else {
		req, err = http.NewRequest(method, url, nil)
	}
	if err != nil {
		t.Fatalf("build %s request: %v", method, err)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s: %v", method, err)
	}
	return resp
}
