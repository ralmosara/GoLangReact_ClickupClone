//go:build ignore

// scripts/seed.go — populates a fresh database with a realistic demo
// workspace so newcomers can log in and immediately see the system in
// action.
//
// IMPORTANT: this is now ALSO the production-bootstrap path. Self-
// registration was removed; the seeder is the only way to create the
// very first owner without an existing admin to invite them. After the
// initial run, every additional account is created by an existing
// workspace admin via POST /api/v1/workspaces/{id}/members.
//
// What it creates:
//
//   * 4 users with predictable credentials (see banner at the bottom for
//     emails and passwords).
//   * 1 workspace, with each user holding a different built-in role
//     (owner / admin / member / guest) so RBAC paths are exercised.
//   * 1 custom role "Project Manager" with a curated permission set, plus
//     one user re-roled into it.
//   * 2 spaces, several folders and lists, statuses, tags, custom fields.
//   * ~30 tasks distributed across the spaces, with assignees, statuses,
//     priorities, due dates, comments, time entries, and tag attachments.
//   * 1 saved view, 1 dashboard, 1 doc, 1 channel with a few messages.
//
// Idempotent: ON CONFLICT clauses everywhere, so you can re-run safely.
//
// Usage:
//
//	go run ./scripts/seed.go
//
// Optional env:
//
//	SEED_PASSWORD  override the default password ("demo1234") for every
//	               seeded user. Useful when seeding shared environments.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"github.com/yourorg/clickup/internal/config"
	"github.com/yourorg/clickup/internal/db"
	"github.com/yourorg/clickup/internal/migrate"
)

func main() {
	cfg := config.Load()
	if cfg.DBDSN == "" {
		log.Fatal("DB_DSN not set (copy .env.example to .env and edit)")
	}

	pool, err := db.NewPostgres(cfg.DBDSN)
	if err != nil {
		log.Fatalf("postgres: %v", err)
	}
	defer pool.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if err := migrate.Up(ctx, pool, cfg.MigrationsDir); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	password := getenv("SEED_PASSWORD", "demo1234")
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("bcrypt: %v", err)
	}

	s := &seeder{pool: pool, ctx: ctx, password: password, hashedPwd: string(hash)}

	// 1. Users — fixed slugs so tests / docs can reference them.
	demoID := s.upsertUser("demo@demo.test", "Demo Owner")
	aliceID := s.upsertUser("alice@demo.test", "Alice Admin")
	bobID := s.upsertUser("bob@demo.test", "Bob Member")
	charlieID := s.upsertUser("charlie@demo.test", "Charlie Guest")

	// 2. Workspace — owner is demo@.
	wsID := s.upsertWorkspace("Demo Workspace", "demo", demoID)

	// 3. Memberships — one user per built-in role so RBAC paths are
	// covered end-to-end. Each membership gets role_id set via the
	// workspace_members table; the migration's back-fill set role_id for
	// the owner row already, but the seeder is idempotent so we re-assign.
	s.assignBuiltinMembership(wsID, demoID, "owner")
	s.assignBuiltinMembership(wsID, aliceID, "admin")
	s.assignBuiltinMembership(wsID, bobID, "member")
	s.assignBuiltinMembership(wsID, charlieID, "guest")

	// 4. Custom role — "Project Manager". Demonstrates per-workspace
	// role creation. Bob is re-roled into it so the resolved permission
	// set differs from the built-in member role.
	pmRoleID := s.upsertCustomRole(wsID, "Project Manager", "Lead a project: assign tasks, manage statuses & sprints, no member-management.", 3, []string{
		"workspace.read",
		"space.read", "space.create", "space.update",
		"folder.create", "folder.update",
		"list.create", "list.read", "list.update", "list.delete",
		"task.create", "task.read", "task.update", "task.delete",
		"task.assign", "task.change_status",
		"comment.create", "comment.update_own", "comment.delete_any",
		"attachment.upload", "attachment.delete",
		"tag.manage", "status.manage", "view.manage",
		"dashboard.create", "dashboard.update",
		"automation.manage", "goal.manage", "sprint.manage",
		"doc.create", "doc.update",
		"chat.send", "chat.manage_channels",
		"time_entry.create", "time_entry.update_own", "time_entry.update_any",
		"report.read",
	})
	s.assignRoleByID(wsID, bobID, pmRoleID)

	// 5. Two spaces with realistic structure.
	productSpaceID := s.upsertSpace(wsID, "Product")
	opsSpaceID := s.upsertSpace(wsID, "Operations")

	q1FolderID := s.upsertFolder(productSpaceID, "Q1 Roadmap")
	backlogListID := s.upsertList(productSpaceID, &q1FolderID, "Backlog")
	currentSprintListID := s.upsertList(productSpaceID, &q1FolderID, "Current Sprint")
	bugsListID := s.upsertList(productSpaceID, nil, "Bugs")

	hiringListID := s.upsertList(opsSpaceID, nil, "Hiring")
	purchasingListID := s.upsertList(opsSpaceID, nil, "Purchasing")

	// 6. Statuses — three per task list. ON CONFLICT keeps re-runs idempotent.
	for _, lid := range []uuid.UUID{backlogListID, currentSprintListID, bugsListID, hiringListID, purchasingListID} {
		s.upsertStatus(lid, "To Do", "#94a3b8", "active", 0)
		s.upsertStatus(lid, "In Progress", "#3b82f6", "active", 1)
		s.upsertStatus(lid, "Done", "#10b981", "closed", 2)
	}

	// 7. Tags — workspace-scoped.
	urgentTag := s.upsertTag(wsID, "urgent", "#ef4444")
	bugTag := s.upsertTag(wsID, "bug", "#f97316")
	enhancementTag := s.upsertTag(wsID, "enhancement", "#10b981")

	// 8. Tasks. Status IDs are looked up after they're created so the
	// "To Do" status we just inserted is the one referenced.
	statusFor := func(listID uuid.UUID, name string) uuid.UUID {
		return s.statusID(listID, name)
	}

	taskSeed := []taskInput{
		{list: backlogListID, name: "Draft Q2 product brief", status: "To Do", priority: 2, assignee: aliceID, due: 7 * 24 * time.Hour, tags: []uuid.UUID{enhancementTag}},
		{list: backlogListID, name: "Customer interview round-up", status: "To Do", priority: 3, assignee: bobID, due: 14 * 24 * time.Hour},
		{list: backlogListID, name: "Pricing tier proposal", status: "In Progress", priority: 2, assignee: aliceID, due: 5 * 24 * time.Hour, tags: []uuid.UUID{enhancementTag}},
		{list: backlogListID, name: "Onboarding redesign", status: "To Do", priority: 1, assignee: bobID, due: 30 * 24 * time.Hour},

		{list: currentSprintListID, name: "Implement OIDC login", status: "In Progress", priority: 3, assignee: aliceID, due: 3 * 24 * time.Hour, tags: []uuid.UUID{enhancementTag}},
		{list: currentSprintListID, name: "Add MFA recovery codes", status: "Done", priority: 3, assignee: aliceID, due: -2 * 24 * time.Hour, tags: []uuid.UUID{enhancementTag}},
		{list: currentSprintListID, name: "Custom roles UI", status: "To Do", priority: 2, assignee: bobID, due: 10 * 24 * time.Hour, tags: []uuid.UUID{enhancementTag}},
		{list: currentSprintListID, name: "Audit log streaming export", status: "Done", priority: 2, assignee: aliceID, due: -1 * 24 * time.Hour},
		{list: currentSprintListID, name: "GDPR workspace export", status: "Done", priority: 2, assignee: aliceID, due: -1 * 24 * time.Hour},
		{list: currentSprintListID, name: "Prometheus metrics", status: "Done", priority: 3, assignee: aliceID, due: -3 * 24 * time.Hour},

		{list: bugsListID, name: "Drag-drop drops on Safari", status: "To Do", priority: 3, assignee: bobID, due: 2 * 24 * time.Hour, tags: []uuid.UUID{bugTag, urgentTag}},
		{list: bugsListID, name: "WebSocket reconnect storm", status: "In Progress", priority: 2, assignee: aliceID, due: 4 * 24 * time.Hour, tags: []uuid.UUID{bugTag}},
		{list: bugsListID, name: "Search returns archived tasks", status: "To Do", priority: 1, assignee: bobID, due: 14 * 24 * time.Hour, tags: []uuid.UUID{bugTag}},

		{list: hiringListID, name: "Backend engineer JD", status: "Done", priority: 2, assignee: demoID, due: -7 * 24 * time.Hour},
		{list: hiringListID, name: "Phone screen Alice's referral", status: "In Progress", priority: 2, assignee: aliceID, due: 1 * 24 * time.Hour},
		{list: hiringListID, name: "Onsite loop for senior FE", status: "To Do", priority: 3, assignee: demoID, due: 7 * 24 * time.Hour, tags: []uuid.UUID{urgentTag}},

		{list: purchasingListID, name: "Datadog renewal", status: "To Do", priority: 1, assignee: demoID, due: 21 * 24 * time.Hour},
		{list: purchasingListID, name: "GitHub seats audit", status: "Done", priority: 1, assignee: aliceID, due: -10 * 24 * time.Hour},
	}

	type createdTask struct {
		id, listID, assigneeID uuid.UUID
		name                   string
	}
	var createdTasks []createdTask
	for i, t := range taskSeed {
		statusID := statusFor(t.list, t.status)
		dueAt := time.Now().Add(t.due)
		taskID := s.upsertTask(t.list, t.name, t.priority, demoID, t.assignee, statusID, dueAt, i, t.status == "Done")
		createdTasks = append(createdTasks, createdTask{id: taskID, listID: t.list, assigneeID: t.assignee, name: t.name})
		for _, tagID := range t.tags {
			s.attachTag(taskID, tagID)
		}
	}

	// 9. Comments — a couple per recently-active task so the UI's
	// activity feeds aren't empty.
	for _, t := range createdTasks[:6] {
		s.upsertComment(t.id, t.assigneeID, "Started looking into this — will share an outline tomorrow.")
		s.upsertComment(t.id, demoID, "Thanks! Tag me when you have a draft.")
	}

	// 10. Time entries — randomized but reproducible.
	rng := rand.New(rand.NewSource(42))
	for _, t := range createdTasks[:8] {
		minutes := 30 + rng.Intn(120)
		started := time.Now().Add(-time.Duration(rng.Intn(72)) * time.Hour)
		s.upsertTimeEntry(t.id, t.assigneeID, started, started.Add(time.Duration(minutes)*time.Minute), "Focus block on "+t.name)
	}

	// 11. One saved view, dashboard, doc, channel — small but enough
	// that every top-level nav page renders something.
	s.upsertView(currentSprintListID, "Sprint board", "board", demoID, json.RawMessage(`{"groupBy":"status"}`))
	dashID := s.upsertDashboard(wsID, "Engineering dashboard", demoID)
	s.upsertWidget(dashID, "task_count", json.RawMessage(fmt.Sprintf(`{"list_id":"%s"}`, currentSprintListID)))

	docID := s.upsertDoc(wsID, demoID, "Welcome", `[{"type":"paragraph","content":[{"type":"text","text":"Welcome to the Demo Workspace! Log in as the four seeded users to see how RBAC behaves."}]}]`)
	_ = docID

	chID := s.upsertChannel(wsID, "general", demoID)
	s.upsertMessage(chID, demoID, "Welcome to general 👋")
	s.upsertMessage(chID, aliceID, "Hi! Just deployed the new RBAC migration.")
	s.upsertMessage(chID, bobID, "Nice — I can see Project Manager in the role picker now.")

	fmt.Println()
	fmt.Println("──────────────────────────────────────────────────────────────")
	fmt.Println(" Seed complete.")
	fmt.Println("──────────────────────────────────────────────────────────────")
	fmt.Println(" Workspace: Demo Workspace (slug: demo)")
	fmt.Println()
	fmt.Println(" Users (password:", password+"):")
	fmt.Println("   demo@demo.test     — owner    (every permission)")
	fmt.Println("   alice@demo.test    — admin    (every permission except workspace.delete)")
	fmt.Println("   bob@demo.test      — Project Manager  (custom role; assigned to Bob)")
	fmt.Println("   charlie@demo.test  — guest    (read-only + comment)")
	fmt.Println()
	fmt.Println(" Try the RBAC endpoints with each token:")
	fmt.Println("   GET /api/v1/permissions")
	fmt.Println("   GET /api/v1/workspaces/{id}/roles")
	fmt.Println("   GET /api/v1/workspaces/{id}/roles/effective")
	fmt.Println("──────────────────────────────────────────────────────────────")
}

// ── helpers ──────────────────────────────────────────────────────────────

type taskInput struct {
	list      uuid.UUID
	name      string
	status    string
	priority  int
	assignee  uuid.UUID
	due       time.Duration
	tags      []uuid.UUID
}

type seeder struct {
	pool      *pgxpool.Pool
	ctx       context.Context
	password  string
	hashedPwd string
}

func (s *seeder) upsertUser(email, name string) uuid.UUID {
	var id uuid.UUID
	err := s.pool.QueryRow(s.ctx, `
		INSERT INTO users (email, password_hash, name)
		VALUES ($1, $2, $3)
		ON CONFLICT (email) DO UPDATE SET name = EXCLUDED.name, password_hash = EXCLUDED.password_hash
		RETURNING id
	`, email, s.hashedPwd, name).Scan(&id)
	if err != nil {
		log.Fatalf("user %s: %v", email, err)
	}
	return id
}

func (s *seeder) upsertWorkspace(name, slug string, ownerID uuid.UUID) uuid.UUID {
	var id uuid.UUID
	err := s.pool.QueryRow(s.ctx, `
		INSERT INTO workspaces (name, slug, owner_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (slug) DO UPDATE SET name = EXCLUDED.name
		RETURNING id
	`, name, slug, ownerID).Scan(&id)
	if err != nil {
		log.Fatalf("workspace %s: %v", slug, err)
	}
	return id
}

// assignBuiltinMembership creates (or updates) a workspace_members row
// pointing at a built-in role by name. The legacy `role` text column is
// kept in sync so callers that still SELECT it directly read the right
// value, even though authorization checks now go through role_id.
func (s *seeder) assignBuiltinMembership(wsID, userID uuid.UUID, builtinRole string) {
	var roleID uuid.UUID
	err := s.pool.QueryRow(s.ctx, `
		SELECT id FROM roles WHERE workspace_id IS NULL AND name = $1
	`, builtinRole).Scan(&roleID)
	if err != nil {
		log.Fatalf("lookup builtin %s: %v", builtinRole, err)
	}
	_, err = s.pool.Exec(s.ctx, `
		INSERT INTO workspace_members (workspace_id, user_id, role, role_id)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (workspace_id, user_id) DO UPDATE
		   SET role    = EXCLUDED.role,
		       role_id = EXCLUDED.role_id
	`, wsID, userID, builtinRole, roleID)
	if err != nil {
		log.Fatalf("membership: %v", err)
	}
}

// upsertCustomRole creates a workspace-scoped custom role with the given
// permission grants. Re-runs of the seeder update the grants in place.
func (s *seeder) upsertCustomRole(wsID uuid.UUID, name, description string, rank int, permKeys []string) uuid.UUID {
	var roleID uuid.UUID
	err := s.pool.QueryRow(s.ctx, `
		INSERT INTO roles (workspace_id, name, description, is_builtin, rank)
		VALUES ($1, $2, $3, FALSE, $4)
		ON CONFLICT (workspace_id, name) WHERE workspace_id IS NOT NULL
		DO UPDATE SET description = EXCLUDED.description, rank = EXCLUDED.rank, updated_at = NOW()
		RETURNING id
	`, wsID, name, description, rank).Scan(&roleID)
	if err != nil {
		log.Fatalf("custom role %s: %v", name, err)
	}
	// Replace grants atomically so removing a key from the seeder script
	// removes it from the DB on next run.
	tx, err := s.pool.Begin(s.ctx)
	if err != nil {
		log.Fatalf("tx: %v", err)
	}
	defer tx.Rollback(s.ctx)
	if _, err := tx.Exec(s.ctx, `DELETE FROM role_permissions WHERE role_id = $1`, roleID); err != nil {
		log.Fatalf("clear role grants: %v", err)
	}
	if _, err := tx.Exec(s.ctx, `
		INSERT INTO role_permissions (role_id, permission_key)
		SELECT $1, p.key
		  FROM permissions p
		 WHERE p.key = ANY($2::TEXT[])
		ON CONFLICT DO NOTHING
	`, roleID, permKeys); err != nil {
		log.Fatalf("set role grants: %v", err)
	}
	if err := tx.Commit(s.ctx); err != nil {
		log.Fatalf("commit grants: %v", err)
	}
	return roleID
}

func (s *seeder) assignRoleByID(wsID, userID, roleID uuid.UUID) {
	_, err := s.pool.Exec(s.ctx, `
		UPDATE workspace_members
		   SET role_id = $3,
		       role    = COALESCE((SELECT name FROM roles WHERE id = $3), role)
		 WHERE workspace_id = $1 AND user_id = $2
	`, wsID, userID, roleID)
	if err != nil {
		log.Fatalf("assign role: %v", err)
	}
}

func (s *seeder) upsertSpace(wsID uuid.UUID, name string) uuid.UUID {
	var id uuid.UUID
	err := s.pool.QueryRow(s.ctx, `
		WITH ins AS (
			SELECT id FROM spaces WHERE workspace_id = $1 AND name = $2
		),
		new_row AS (
			INSERT INTO spaces (workspace_id, name)
			SELECT $1, $2 WHERE NOT EXISTS (SELECT 1 FROM ins)
			RETURNING id
		)
		SELECT id FROM ins
		UNION ALL
		SELECT id FROM new_row
		LIMIT 1
	`, wsID, name).Scan(&id)
	if err != nil {
		log.Fatalf("space %s: %v", name, err)
	}
	return id
}

func (s *seeder) upsertFolder(spaceID uuid.UUID, name string) uuid.UUID {
	var id uuid.UUID
	err := s.pool.QueryRow(s.ctx, `
		WITH ins AS (
			SELECT id FROM folders WHERE space_id = $1 AND name = $2
		),
		new_row AS (
			INSERT INTO folders (space_id, name)
			SELECT $1, $2 WHERE NOT EXISTS (SELECT 1 FROM ins)
			RETURNING id
		)
		SELECT id FROM ins UNION ALL SELECT id FROM new_row LIMIT 1
	`, spaceID, name).Scan(&id)
	if err != nil {
		log.Fatalf("folder %s: %v", name, err)
	}
	return id
}

func (s *seeder) upsertList(spaceID uuid.UUID, folderID *uuid.UUID, name string) uuid.UUID {
	var id uuid.UUID
	err := s.pool.QueryRow(s.ctx, `
		WITH ins AS (
			SELECT id FROM lists WHERE space_id = $1 AND COALESCE(folder_id, '00000000-0000-0000-0000-000000000000'::uuid) = COALESCE($2, '00000000-0000-0000-0000-000000000000'::uuid) AND name = $3
		),
		new_row AS (
			INSERT INTO lists (space_id, folder_id, name)
			SELECT $1, $2, $3 WHERE NOT EXISTS (SELECT 1 FROM ins)
			RETURNING id
		)
		SELECT id FROM ins UNION ALL SELECT id FROM new_row LIMIT 1
	`, spaceID, folderID, name).Scan(&id)
	if err != nil {
		log.Fatalf("list %s: %v", name, err)
	}
	return id
}

func (s *seeder) upsertStatus(listID uuid.UUID, name, color, category string, order int) uuid.UUID {
	var id uuid.UUID
	err := s.pool.QueryRow(s.ctx, `
		INSERT INTO statuses (list_id, name, color, category, order_index)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (list_id, name) DO UPDATE
		   SET color = EXCLUDED.color, category = EXCLUDED.category, order_index = EXCLUDED.order_index
		RETURNING id
	`, listID, name, color, category, order).Scan(&id)
	if err != nil {
		log.Fatalf("status %s: %v", name, err)
	}
	return id
}

func (s *seeder) statusID(listID uuid.UUID, name string) uuid.UUID {
	var id uuid.UUID
	err := s.pool.QueryRow(s.ctx,
		`SELECT id FROM statuses WHERE list_id=$1 AND name=$2`, listID, name).Scan(&id)
	if err != nil {
		log.Fatalf("status lookup %s: %v", name, err)
	}
	return id
}

func (s *seeder) upsertTag(wsID uuid.UUID, name, color string) uuid.UUID {
	var id uuid.UUID
	err := s.pool.QueryRow(s.ctx, `
		WITH ins AS (
			SELECT id FROM tags WHERE workspace_id = $1 AND name = $2
		),
		new_row AS (
			INSERT INTO tags (workspace_id, name, color)
			SELECT $1, $2, $3 WHERE NOT EXISTS (SELECT 1 FROM ins)
			RETURNING id
		)
		SELECT id FROM ins UNION ALL SELECT id FROM new_row LIMIT 1
	`, wsID, name, color).Scan(&id)
	if err != nil {
		log.Fatalf("tag %s: %v", name, err)
	}
	return id
}

func (s *seeder) upsertTask(listID uuid.UUID, name string, priority int, creator, assignee, statusID uuid.UUID, dueAt time.Time, position int, completed bool) uuid.UUID {
	var id uuid.UUID
	var completedAt *time.Time
	if completed {
		t := time.Now().Add(-1 * time.Hour)
		completedAt = &t
	}
	err := s.pool.QueryRow(s.ctx, `
		WITH ins AS (
			SELECT id FROM tasks WHERE list_id = $1 AND name = $2
		),
		new_row AS (
			INSERT INTO tasks (list_id, name, status, status_id, priority, creator_id, assignee_id, due_at, position, completed_at)
			SELECT $1, $2, 'open', $3, $4, $5, $6, $7, $8, $9
			WHERE NOT EXISTS (SELECT 1 FROM ins)
			RETURNING id
		)
		SELECT id FROM ins UNION ALL SELECT id FROM new_row LIMIT 1
	`, listID, name, statusID, priority, creator, assignee, dueAt, position, completedAt).Scan(&id)
	if err != nil {
		log.Fatalf("task %s: %v", name, err)
	}
	// Make sure the multi-assignee join row exists too — the modern code
	// reads task_assignees, not the legacy single column.
	_, err = s.pool.Exec(s.ctx, `
		INSERT INTO task_assignees (task_id, user_id) VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`, id, assignee)
	if err != nil {
		log.Fatalf("task_assignees: %v", err)
	}
	return id
}

func (s *seeder) attachTag(taskID, tagID uuid.UUID) {
	_, err := s.pool.Exec(s.ctx, `
		INSERT INTO task_tags (task_id, tag_id) VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`, taskID, tagID)
	if err != nil {
		log.Fatalf("task_tags: %v", err)
	}
}

func (s *seeder) upsertComment(taskID, authorID uuid.UUID, body string) {
	// No natural unique key on comments; we just dedupe by (task_id, author_id, body).
	_, err := s.pool.Exec(s.ctx, `
		INSERT INTO comments (task_id, author_id, body)
		SELECT $1, $2, $3
		WHERE NOT EXISTS (
			SELECT 1 FROM comments WHERE task_id = $1 AND author_id = $2 AND body = $3
		)
	`, taskID, authorID, body)
	if err != nil {
		log.Fatalf("comment: %v", err)
	}
}

func (s *seeder) upsertTimeEntry(taskID, userID uuid.UUID, started, stopped time.Time, note string) {
	// In an INSERT ... SELECT $params form Postgres can't infer parameter
	// types from the target columns, so the bare `$4 - $3` subtraction
	// resolves as `unknown - unknown` and fails with SQLSTATE 42725.
	// Explicit ::timestamptz casts pin the types so the operator resolves
	// to timestamptz subtraction. duration_s is an INT in the schema.
	_, err := s.pool.Exec(s.ctx, `
		INSERT INTO time_entries (task_id, user_id, started_at, stopped_at, duration_s, note, billable)
		SELECT $1::uuid, $2::uuid, $3::timestamptz, $4::timestamptz,
		       EXTRACT(EPOCH FROM ($4::timestamptz - $3::timestamptz))::INT,
		       $5::text, TRUE
		WHERE NOT EXISTS (
			SELECT 1 FROM time_entries
			 WHERE task_id    = $1::uuid
			   AND user_id    = $2::uuid
			   AND started_at = $3::timestamptz
		)
	`, taskID, userID, started, stopped, note)
	if err != nil {
		log.Fatalf("time entry: %v", err)
	}
}

func (s *seeder) upsertView(listID uuid.UUID, name, kind string, ownerID uuid.UUID, config json.RawMessage) {
	// `views` has no creator_id column in the current schema — ownerID is
	// kept on the function signature so the caller's intent is documented
	// and so adding the column later is a one-line change.
	_ = ownerID
	_, err := s.pool.Exec(s.ctx, `
		INSERT INTO views (list_id, name, kind, config)
		SELECT $1, $2, $3, $4
		WHERE NOT EXISTS (SELECT 1 FROM views WHERE list_id = $1 AND name = $2)
	`, listID, name, kind, config)
	if err != nil {
		log.Fatalf("view: %v", err)
	}
}

func (s *seeder) upsertDashboard(wsID uuid.UUID, name string, ownerID uuid.UUID) uuid.UUID {
	var id uuid.UUID
	err := s.pool.QueryRow(s.ctx, `
		WITH ins AS (
			SELECT id FROM dashboards WHERE workspace_id = $1 AND name = $2
		),
		new_row AS (
			INSERT INTO dashboards (workspace_id, name, creator_id)
			SELECT $1, $2, $3 WHERE NOT EXISTS (SELECT 1 FROM ins)
			RETURNING id
		)
		SELECT id FROM ins UNION ALL SELECT id FROM new_row LIMIT 1
	`, wsID, name, ownerID).Scan(&id)
	if err != nil {
		log.Fatalf("dashboard: %v", err)
	}
	return id
}

func (s *seeder) upsertWidget(dashID uuid.UUID, kind string, config json.RawMessage) {
	// Real table name is `dashboard_widgets`; the domain layer aliases it
	// as Widget which is why the codebase is sprinkled with both names.
	_, err := s.pool.Exec(s.ctx, `
		INSERT INTO dashboard_widgets (dashboard_id, kind, config)
		SELECT $1, $2, $3
		WHERE NOT EXISTS (SELECT 1 FROM dashboard_widgets WHERE dashboard_id = $1 AND kind = $2)
	`, dashID, kind, config)
	if err != nil {
		log.Fatalf("widget: %v", err)
	}
}

func (s *seeder) upsertDoc(wsID, ownerID uuid.UUID, title, contentJSON string) uuid.UUID {
	var id uuid.UUID
	err := s.pool.QueryRow(s.ctx, `
		WITH ins AS (
			SELECT id FROM docs WHERE workspace_id = $1 AND title = $2
		),
		new_row AS (
			INSERT INTO docs (workspace_id, creator_id, title, content, content_text)
			SELECT $1, $3, $2, $4::JSONB, $2
			WHERE NOT EXISTS (SELECT 1 FROM ins)
			RETURNING id
		)
		SELECT id FROM ins UNION ALL SELECT id FROM new_row LIMIT 1
	`, wsID, title, ownerID, contentJSON).Scan(&id)
	if err != nil {
		log.Fatalf("doc: %v", err)
	}
	return id
}

func (s *seeder) upsertChannel(wsID uuid.UUID, name string, ownerID uuid.UUID) uuid.UUID {
	var id uuid.UUID
	err := s.pool.QueryRow(s.ctx, `
		WITH ins AS (
			SELECT id FROM channels WHERE workspace_id = $1 AND name = $2
		),
		new_row AS (
			INSERT INTO channels (workspace_id, name, kind, creator_id)
			SELECT $1, $2, 'channel', $3 WHERE NOT EXISTS (SELECT 1 FROM ins)
			RETURNING id
		)
		SELECT id FROM ins UNION ALL SELECT id FROM new_row LIMIT 1
	`, wsID, name, ownerID).Scan(&id)
	if err != nil {
		log.Fatalf("channel: %v", err)
	}
	return id
}

func (s *seeder) upsertMessage(channelID, authorID uuid.UUID, body string) {
	_, err := s.pool.Exec(s.ctx, `
		INSERT INTO messages (channel_id, author_id, body)
		SELECT $1, $2, $3
		WHERE NOT EXISTS (SELECT 1 FROM messages WHERE channel_id = $1 AND author_id = $2 AND body = $3)
	`, channelID, authorID, body)
	if err != nil {
		log.Fatalf("message: %v", err)
	}
}

func getenv(k, fallback string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return fallback
}
