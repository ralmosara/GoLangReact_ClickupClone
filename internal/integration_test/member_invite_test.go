package integration_test

import (
	"context"
	"net/http"
	"net/http/httptest"
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

// TestMemberInvite_StrictExisting locks in the post-split contract for
// /workspaces/{id}/members:
//
//   * Existing user → added as member (201).
//   * Unknown email → 404 with reason="user_not_found" — admin must use
//     the User Management page first.
//
// User creation has moved to /workspaces/{id}/users (covered in
// user_management_test.go); this test ensures the member endpoint stays
// the lighter "invite-existing" surface.
func TestMemberInvite_StrictExisting(t *testing.T) {
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

	uH := userHandler.New(uSvc)
	mH := memberHandler.NewWithPolicy(mSvc, policy)

	r := chi.NewRouter()
	r.Route("/api/v1", func(r chi.Router) {
		r.Group(func(pub chi.Router) { uH.PublicRoutes(pub) })
		r.Group(func(pr chi.Router) {
			pr.Use(middleware.Auth(jwtSecret))
			uH.PrivateRoutes(pr)
			mH.Routes(pr)
		})
	})
	srv := httptest.NewServer(r)
	defer srv.Close()

	// Seed admin + workspace + an additional pre-existing user (the one
	// we'll legitimately invite into the workspace).
	admin, err := uSvc.AdminCreate(ctx, userSvc.AdminCreateInput{
		Email:    "owner@example.com",
		Password: "owner-password-1",
		Name:     "Owner",
	})
	if err != nil {
		t.Fatalf("admin create: %v", err)
	}
	other, err := uSvc.AdminCreate(ctx, userSvc.AdminCreateInput{
		Email:    "alice@example.com",
		Password: "alice-password-1",
		Name:     "Alice",
	})
	if err != nil {
		t.Fatalf("alice create: %v", err)
	}
	_ = other

	var workspaceID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO workspaces (name, slug, owner_id) VALUES ('Test', 'test', $1)
		RETURNING id
	`, admin.ID).Scan(&workspaceID); err != nil {
		t.Fatalf("ws insert: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO workspace_members (workspace_id, user_id, role, role_id)
		VALUES ($1, $2, 'owner',
		    (SELECT id FROM roles WHERE workspace_id IS NULL AND name = 'owner'))
	`, workspaceID, admin.ID); err != nil {
		t.Fatalf("ws member insert: %v", err)
	}

	// Owner logs in.
	resp := postJSON(t, srv.URL+"/api/v1/auth/login", "",
		`{"email":"owner@example.com","password":"owner-password-1"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("admin login = %d; want 200", resp.StatusCode)
	}
	var adminLogin struct {
		Token string `json:"token"`
	}
	decode(t, resp, &adminLogin)

	// 1. Existing user → 201
	resp = postJSON(t, srv.URL+"/api/v1/workspaces/"+workspaceID+"/members", adminLogin.Token,
		`{"email":"alice@example.com","role":"member"}`)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("existing invite = %d; want 201", resp.StatusCode)
	}
	var member struct {
		UserID string `json:"user_id"`
		Email  string `json:"email"`
		Role   string `json:"role"`
	}
	decode(t, resp, &member)
	if member.Email != "alice@example.com" || member.Role != "member" {
		t.Fatalf("invite payload: %+v", member)
	}

	// 2. Unknown email → 404 with reason="user_not_found"
	resp = postJSON(t, srv.URL+"/api/v1/workspaces/"+workspaceID+"/members", adminLogin.Token,
		`{"email":"nobody@example.com","role":"member"}`)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown invite = %d; want 404", resp.StatusCode)
	}
	var body struct {
		Error  string `json:"error"`
		Reason string `json:"reason"`
	}
	decode(t, resp, &body)
	if body.Reason != "user_not_found" {
		t.Fatalf("unknown invite reason = %q; want user_not_found", body.Reason)
	}

	// 3. Legacy fields (name/password) are silently ignored — the
	// endpoint just looks up by email and treats the rest as no-ops.
	resp = postJSON(t, srv.URL+"/api/v1/workspaces/"+workspaceID+"/members", adminLogin.Token,
		`{"email":"alice@example.com","role":"admin","name":"ignored","password":"ignored"}`)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("legacy-field invite = %d; want 201", resp.StatusCode)
	}
}
