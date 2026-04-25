//go:build ignore

package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/yourorg/clickup/internal/config"
	"github.com/yourorg/clickup/internal/db"
	"github.com/yourorg/clickup/internal/migrate"
)

// Usage: go run ./scripts/seed.go
// Creates a demo user (demo@demo.test / demo1234), a workspace, a space, a list,
// and a few tasks so the UI has something to show on first boot.
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

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := migrate.Up(ctx, pool, cfg.MigrationsDir); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	hash, _ := bcrypt.GenerateFromPassword([]byte("demo1234"), bcrypt.DefaultCost)

	var userID string
	err = pool.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, name)
		VALUES ('demo@demo.test', $1, 'Demo')
		ON CONFLICT (email) DO UPDATE SET name = EXCLUDED.name
		RETURNING id
	`, string(hash)).Scan(&userID)
	if err != nil {
		log.Fatalf("user: %v", err)
	}

	var workspaceID string
	err = pool.QueryRow(ctx, `
		INSERT INTO workspaces (name, slug, owner_id)
		VALUES ('Demo Workspace', 'demo', $1)
		ON CONFLICT (slug) DO UPDATE SET name = EXCLUDED.name
		RETURNING id
	`, userID).Scan(&workspaceID)
	if err != nil {
		log.Fatalf("workspace: %v", err)
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO workspace_members (workspace_id, user_id, role)
		VALUES ($1, $2, 'owner') ON CONFLICT DO NOTHING
	`, workspaceID, userID)
	if err != nil {
		log.Fatalf("membership: %v", err)
	}

	var spaceID string
	err = pool.QueryRow(ctx, `
		INSERT INTO spaces (workspace_id, name) VALUES ($1, 'General')
		RETURNING id
	`, workspaceID).Scan(&spaceID)
	if err != nil {
		log.Fatalf("space: %v", err)
	}

	var listID string
	err = pool.QueryRow(ctx, `
		INSERT INTO lists (space_id, name) VALUES ($1, 'Inbox')
		RETURNING id
	`, spaceID).Scan(&listID)
	if err != nil {
		log.Fatalf("list: %v", err)
	}

	for i, name := range []string{"Draft project brief", "Kickoff meeting", "Ship MVP"} {
		_, err = pool.Exec(ctx, `
			INSERT INTO tasks (list_id, name, status, priority, creator_id, position)
			VALUES ($1, $2, 'open', $3, $4, $5)
		`, listID, name, i+1, userID, i)
		if err != nil {
			log.Fatalf("task: %v", err)
		}
	}

	fmt.Println("Seed complete.")
	fmt.Println("Login: demo@demo.test / demo1234")
}
