package main

import (
	"context"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"encoding/json"

	"github.com/getsentry/sentry-go"

	"github.com/yourorg/clickup/internal/audit"
	"github.com/yourorg/clickup/internal/authz"
	autoeng "github.com/yourorg/clickup/internal/automation"
	"github.com/yourorg/clickup/internal/config"
	cryptox "github.com/yourorg/clickup/internal/crypto"
	"github.com/yourorg/clickup/internal/db"
	"github.com/yourorg/clickup/internal/domain"
	"github.com/yourorg/clickup/internal/middleware"
	"github.com/yourorg/clickup/internal/migrate"
	"github.com/yourorg/clickup/internal/notify"
	"github.com/yourorg/clickup/internal/observability"
	oidcsvc "github.com/yourorg/clickup/internal/service/oidc"
	"github.com/yourorg/clickup/internal/storage"
	"github.com/yourorg/clickup/internal/ws"

	"github.com/google/uuid"

	assigneeRepo "github.com/yourorg/clickup/internal/repository/assignee"
	attachmentRepo "github.com/yourorg/clickup/internal/repository/attachment"
	auditRepo "github.com/yourorg/clickup/internal/repository/audit"
	automationRepo "github.com/yourorg/clickup/internal/repository/automation"
	channelRepo "github.com/yourorg/clickup/internal/repository/channel"
	commentRepo "github.com/yourorg/clickup/internal/repository/comment"
	credentialRepo "github.com/yourorg/clickup/internal/repository/credential"
	customfieldRepo "github.com/yourorg/clickup/internal/repository/customfield"
	customvalueRepo "github.com/yourorg/clickup/internal/repository/customvalue"
	dashboardRepo "github.com/yourorg/clickup/internal/repository/dashboard"
	dependencyRepo "github.com/yourorg/clickup/internal/repository/dependency"
	docRepo "github.com/yourorg/clickup/internal/repository/doc"
	folderRepo "github.com/yourorg/clickup/internal/repository/folder"
	formRepo "github.com/yourorg/clickup/internal/repository/form"
	goalRepo "github.com/yourorg/clickup/internal/repository/goal"
	goaltargetRepo "github.com/yourorg/clickup/internal/repository/goaltarget"
	listRepo "github.com/yourorg/clickup/internal/repository/list"
	memberRepo "github.com/yourorg/clickup/internal/repository/member"
	messageRepo "github.com/yourorg/clickup/internal/repository/message"
	mfaRepo "github.com/yourorg/clickup/internal/repository/mfa"
	milestoneRepo "github.com/yourorg/clickup/internal/repository/milestone"
	portfolioRepo "github.com/yourorg/clickup/internal/repository/portfolio"
	notificationRepo "github.com/yourorg/clickup/internal/repository/notification"
	oidcRepo "github.com/yourorg/clickup/internal/repository/oidc"
	reportRepo "github.com/yourorg/clickup/internal/repository/report"
	roleRepo "github.com/yourorg/clickup/internal/repository/role"
	savedsearchRepo "github.com/yourorg/clickup/internal/repository/savedsearch"
	searchRepo "github.com/yourorg/clickup/internal/repository/search"
	sessionRepo "github.com/yourorg/clickup/internal/repository/session"
	spaceRepo "github.com/yourorg/clickup/internal/repository/space"
	sprintRepo "github.com/yourorg/clickup/internal/repository/sprint"
	statusRepo "github.com/yourorg/clickup/internal/repository/status"
	tagRepo "github.com/yourorg/clickup/internal/repository/tag"
	taskRepo "github.com/yourorg/clickup/internal/repository/task"
	templateRepo "github.com/yourorg/clickup/internal/repository/template"
	timeentryRepo "github.com/yourorg/clickup/internal/repository/timeentry"
	userRepo "github.com/yourorg/clickup/internal/repository/user"
	viewRepo "github.com/yourorg/clickup/internal/repository/view"
	whiteboardRepo "github.com/yourorg/clickup/internal/repository/whiteboard"
	workspaceRepo "github.com/yourorg/clickup/internal/repository/workspace"

	attachmentSvc "github.com/yourorg/clickup/internal/service/attachment"
	auditSvc "github.com/yourorg/clickup/internal/service/audit"
	automationSvc "github.com/yourorg/clickup/internal/service/automation"
	chatSvc "github.com/yourorg/clickup/internal/service/chat"
	commentSvc "github.com/yourorg/clickup/internal/service/comment"
	credentialSvc "github.com/yourorg/clickup/internal/service/credential"
	customfieldSvc "github.com/yourorg/clickup/internal/service/customfield"
	dashboardSvc "github.com/yourorg/clickup/internal/service/dashboard"
	dependencySvc "github.com/yourorg/clickup/internal/service/dependency"
	docSvc "github.com/yourorg/clickup/internal/service/doc"
	folderSvc "github.com/yourorg/clickup/internal/service/folder"
	formSvc "github.com/yourorg/clickup/internal/service/form"
	gdprSvc "github.com/yourorg/clickup/internal/service/gdpr"
	goalSvc "github.com/yourorg/clickup/internal/service/goal"
	listSvc "github.com/yourorg/clickup/internal/service/list"
	memberSvc "github.com/yourorg/clickup/internal/service/member"
	mfaSvc "github.com/yourorg/clickup/internal/service/mfa"
	milestoneSvc "github.com/yourorg/clickup/internal/service/milestone"
	portfolioSvc "github.com/yourorg/clickup/internal/service/portfolio"
	notificationSvc "github.com/yourorg/clickup/internal/service/notification"
	reportSvc "github.com/yourorg/clickup/internal/service/report"
	roleSvc "github.com/yourorg/clickup/internal/service/role"
	savedsearchSvc "github.com/yourorg/clickup/internal/service/savedsearch"
	searchSvc "github.com/yourorg/clickup/internal/service/search"
	sessionSvc "github.com/yourorg/clickup/internal/service/session"
	spaceSvc "github.com/yourorg/clickup/internal/service/space"
	sprintSvc "github.com/yourorg/clickup/internal/service/sprint"
	statusSvc "github.com/yourorg/clickup/internal/service/status"
	tagSvc "github.com/yourorg/clickup/internal/service/tag"
	taskSvc "github.com/yourorg/clickup/internal/service/task"
	templateSvc "github.com/yourorg/clickup/internal/service/template"
	timeentrySvc "github.com/yourorg/clickup/internal/service/timeentry"
	userSvc "github.com/yourorg/clickup/internal/service/user"
	viewSvc "github.com/yourorg/clickup/internal/service/view"
	whiteboardSvc "github.com/yourorg/clickup/internal/service/whiteboard"
	workloadSvc "github.com/yourorg/clickup/internal/service/workload"
	workspaceSvc "github.com/yourorg/clickup/internal/service/workspace"

	attachmentHandler "github.com/yourorg/clickup/internal/handler/attachment"
	auditHandler "github.com/yourorg/clickup/internal/handler/audit"
	automationHandler "github.com/yourorg/clickup/internal/handler/automation"
	chatHandler "github.com/yourorg/clickup/internal/handler/chat"
	commentHandler "github.com/yourorg/clickup/internal/handler/comment"
	credentialHandler "github.com/yourorg/clickup/internal/handler/credential"
	customfieldHandler "github.com/yourorg/clickup/internal/handler/customfield"
	dashboardHandler "github.com/yourorg/clickup/internal/handler/dashboard"
	dependencyHandler "github.com/yourorg/clickup/internal/handler/dependency"
	docHandler "github.com/yourorg/clickup/internal/handler/doc"
	folderHandler "github.com/yourorg/clickup/internal/handler/folder"
	formHandler "github.com/yourorg/clickup/internal/handler/form"
	gdprHandler "github.com/yourorg/clickup/internal/handler/gdpr"
	goalHandler "github.com/yourorg/clickup/internal/handler/goal"
	listHandler "github.com/yourorg/clickup/internal/handler/list"
	memberHandler "github.com/yourorg/clickup/internal/handler/member"
	mfaHandler "github.com/yourorg/clickup/internal/handler/mfa"
	milestoneHandler "github.com/yourorg/clickup/internal/handler/milestone"
	portfolioHandler "github.com/yourorg/clickup/internal/handler/portfolio"
	notificationHandler "github.com/yourorg/clickup/internal/handler/notification"
	oidcHandler "github.com/yourorg/clickup/internal/handler/oidc"
	reportHandler "github.com/yourorg/clickup/internal/handler/report"
	roleHandler "github.com/yourorg/clickup/internal/handler/role"
	savedsearchHandler "github.com/yourorg/clickup/internal/handler/savedsearch"
	searchHandler "github.com/yourorg/clickup/internal/handler/search"
	sessionHandler "github.com/yourorg/clickup/internal/handler/session"
	spaceHandler "github.com/yourorg/clickup/internal/handler/space"
	sprintHandler "github.com/yourorg/clickup/internal/handler/sprint"
	statusHandler "github.com/yourorg/clickup/internal/handler/status"
	tagHandler "github.com/yourorg/clickup/internal/handler/tag"
	taskHandler "github.com/yourorg/clickup/internal/handler/task"
	templateHandler "github.com/yourorg/clickup/internal/handler/template"
	timeentryHandler "github.com/yourorg/clickup/internal/handler/timeentry"
	userHandler "github.com/yourorg/clickup/internal/handler/user"
	viewHandler "github.com/yourorg/clickup/internal/handler/view"
	whiteboardHandler "github.com/yourorg/clickup/internal/handler/whiteboard"
	workloadHandler "github.com/yourorg/clickup/internal/handler/workload"
	workspaceHandler "github.com/yourorg/clickup/internal/handler/workspace"
)

// minMigrationVersion is bumped whenever a new schema migration is added that
// the binary depends on at runtime. /readyz fails if the DB is behind this
// number — i.e. a freshly-deployed binary against a not-yet-migrated DB
// stays out of the load balancer until migrations finish.
const minMigrationVersion = 38

func main() {
	migrateOnly := flag.Bool("migrate", false, "run migrations then exit")
	flag.Parse()

	cfg := config.Load()
	// In development, surface DEBUG-level logs (includes hub room/publish traces).
	// Production drops back to INFO via APP_ENV.
	logLevel := slog.LevelInfo
	if cfg.AppEnv == "development" {
		logLevel = slog.LevelDebug
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))
	slog.SetDefault(logger)

	if cfg.DBDSN == "" {
		logger.Error("DB_DSN is required (see .env.example)")
		os.Exit(1)
	}

	// Resolve the credentials-vault encryption key. Production must set a real
	// 32-byte hex key; in development we generate a deterministic dev key from
	// the JWT secret so a fresh checkout boots without extra config — but log a
	// warning so it's obvious this is not safe for prod.
	credKey, err := resolveCredentialsKey(cfg, logger)
	if err != nil {
		logger.Error("credentials key invalid", "err", err)
		os.Exit(1)
	}

	// ── Observability bring-up ───────────────────────────────────────────
	// All three are env-gated and degrade to no-ops when their primary
	// endpoint/DSN is unset. Order: tracing first (so spans cover startup),
	// metrics next (registry is shared across the rest of bring-up),
	// Sentry last (just hooks into the HTTP chain).

	rootCtx := context.Background()
	traceShutdown, err := observability.InitTracing(rootCtx, observability.TraceConfig{
		ServiceName:    cfg.OTELServiceName,
		ServiceVersion: cfg.OTELServiceVersion,
		Environment:    cfg.AppEnv,
		OTLPEndpoint:   cfg.OTLPEndpoint,
		OTLPInsecure:   cfg.OTLPInsecure,
		SampleRatio:    cfg.OTELSampleRatio,
	}, logger)
	if err != nil {
		logger.Error("otel init failed", "err", err)
		os.Exit(1)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = traceShutdown(ctx)
	}()

	metrics := observability.NewMetrics()

	sentryFlush, sentryMW, err := observability.InitSentry(observability.SentryConfig{
		DSN:         cfg.SentryDSN,
		Environment: cfg.AppEnv,
		Release:     cfg.SentryRelease,
		SampleRate:  cfg.SentrySampleRate,
	}, logger)
	if err != nil {
		logger.Error("sentry init failed", "err", err)
		os.Exit(1)
	}
	defer sentryFlush(2 * time.Second)
	defer sentry.Recover() // last-ditch panic capture in main itself

	pool, err := db.NewPostgres(cfg.DBDSN)
	if err != nil {
		logger.Error("postgres connect failed", "err", err)
		os.Exit(1)
	}
	defer pool.Close()
	metrics.WirePgxPoolStats(pool)

	if cfg.AutoMigrate || *migrateOnly {
		if err := migrate.Up(context.Background(), pool, cfg.MigrationsDir); err != nil {
			logger.Error("migrate failed", "err", err)
			os.Exit(1)
		}
		logger.Info("migrations applied")
		if *migrateOnly {
			return
		}
	}

	rdbClient := db.NewRedis(cfg.RedisAddr)
	defer rdbClient.Close()

	hub := ws.NewHub()
	hub.SetAuth(
		func(r *http.Request) (uuid.UUID, error) {
			return middleware.ParseToken(cfg.JWTSecret, middleware.TokenFromRequest(r))
		},
		ws.AllowedOrigins{
			"http://localhost:5173": true, // vite dev
			"http://127.0.0.1:5173": true,
			"http://localhost:8080": true, // served by Go directly
			"http://127.0.0.1:8080": true,
		},
	)
	hub.SetLogger(logger)
	go hub.Run()

	store, err := storage.NewLocal(cfg.StorageDir)
	if err != nil {
		logger.Error("storage init failed", "err", err)
		os.Exit(1)
	}

	// repos
	uRepo := userRepo.New(pool)
	wsRepo := workspaceRepo.New(pool)
	spRepo := spaceRepo.New(pool)
	fRepo := folderRepo.New(pool)
	lRepo := listRepo.New(pool)
	tRepo := taskRepo.New(pool)
	cRepo := commentRepo.New(pool)
	credRepo := credentialRepo.New(pool)
	stRepo := statusRepo.New(pool)
	tgRepo := tagRepo.New(pool)
	atRepo := attachmentRepo.New(pool)
	asRepo := assigneeRepo.New(pool)
	nRepo := notificationRepo.New(pool)
	auRepo := auditRepo.New(pool)
	mRepo := memberRepo.New(pool)
	vRepo := viewRepo.New(pool)
	cfRepo := customfieldRepo.New(pool)
	cvRepo := customvalueRepo.New(pool)
	teRepo := timeentryRepo.New(pool)
	depRepo := dependencyRepo.New(pool)
	autoRepo := automationRepo.New(pool)
	dRepo := docRepo.New(pool)
	chRepo := channelRepo.New(pool)
	msgRepo := messageRepo.New(pool)
	sRepo := searchRepo.New(pool)
	goalsRepo := goalRepo.New(pool)
	targetsRepo := goaltargetRepo.New(pool)
	sprintsRepo := sprintRepo.New(pool)
	dashRepo := dashboardRepo.New(pool)
	wbRepo := whiteboardRepo.New(pool)
	fmRepo := formRepo.New(pool)
	tplRepo := templateRepo.New(pool)
	repRepo := reportRepo.New(pool)
	mfaRepoImpl := mfaRepo.New(pool)
	oidcRepoImpl := oidcRepo.New(pool)
	roleRepoImpl := roleRepo.New(pool)
	sessRepoImpl := sessionRepo.New(pool)
	savedSearchRepoImpl := savedsearchRepo.New(pool)
	milestoneRepoImpl := milestoneRepo.New(pool)
	portfolioRepoImpl := portfolioRepo.New(pool)

	// cross-cutting infra
	dispatcher := notify.New(nRepo, hub, logger)
	recorder := audit.New(auRepo, logger)
	policy := authz.New(pool) // policy layer enforced by handlers (audit, automation, etc.)

	// services
	mfaSvcImpl := mfaSvc.New(mfaRepoImpl, uRepo, cfg.MFAIssuer)
	sessSvcImpl := sessionSvc.New(sessRepoImpl, recorder)
	uSvc := userSvc.New(uRepo, cfg.JWTSecret).
		WithMFA(mfaSvcImpl).
		WithSessions(sessionUserAdapter{svc: sessSvcImpl})
	gdprSvcImpl := gdprSvc.New(pool)
	roleSvcImpl := roleSvc.New(roleRepoImpl)
	savedSearchSvcImpl := savedsearchSvc.New(savedSearchRepoImpl)
	milestoneSvcImpl := milestoneSvc.New(milestoneRepoImpl, recorder)
	workloadSvcImpl := workloadSvc.New(tRepo, workloadSvc.Lookups{
		GetUser: func(ctx context.Context, id uuid.UUID) (*domain.User, error) {
			return uRepo.GetByID(ctx, id)
		},
	})
	portfolioSvcImpl := portfolioSvc.New(portfolioRepoImpl, recorder)

	// OIDC providers — only the ones with full env config get registered.
	var oidcProviders []oidcsvc.ProviderConfig
	if cfg.GoogleOIDCClientID != "" && cfg.GoogleOIDCSecret != "" && cfg.GoogleOIDCRedirect != "" {
		oidcProviders = append(oidcProviders, oidcsvc.ProviderConfig{
			Name:         "google",
			Issuer:       "https://accounts.google.com",
			ClientID:     cfg.GoogleOIDCClientID,
			ClientSecret: cfg.GoogleOIDCSecret,
			RedirectURL:  cfg.GoogleOIDCRedirect,
		})
	}
	oidcSvcImpl := oidcsvc.New(uRepo, oidcRepoImpl, cfg.JWTSecret, oidcProviders)

	wSvc := workspaceSvc.New(wsRepo)
	spSvc := spaceSvc.New(spRepo)
	fSvc := folderSvc.New(fRepo)
	lSvc := listSvc.New(lRepo)
	tSvc := taskSvc.New(taskSvc.Deps{
		Tasks:     tRepo,
		Statuses:  stRepo,
		Assignees: asRepo,
		Hub:       hub,
		Notify:    dispatcher,
		Audit:     recorder,
	})
	cSvc := commentSvc.New(commentSvc.Deps{
		Comments: cRepo,
		Tasks:    tRepo,
		Hub:      hub,
		Notify:   dispatcher,
		Audit:    recorder,
	})
	credSvc := credentialSvc.New(credRepo, credKey, wSvc)
	stSvc := statusSvc.New(stRepo, hub, recorder)
	tgSvc := tagSvc.New(tgRepo, recorder)
	atSvc := attachmentSvc.New(atRepo, store, hub, recorder)
	nSvc := notificationSvc.New(nRepo)
	auSvc := auditSvc.New(auRepo)
	mSvc := memberSvc.New(mRepo, uRepo, hub, recorder)
	vSvc := viewSvc.New(vRepo, recorder)
	cfSvc := customfieldSvc.New(cfRepo, cvRepo, recorder)
	teSvc := timeentrySvc.New(teRepo, recorder)
	depSvc := dependencySvc.New(depRepo, tRepo, hub, recorder)
	autoSvcImpl := automationSvc.New(autoRepo, recorder).WithListLookup(
		// Resolve a listID to the workspace that owns it so the automation
		// handler can reject cross-workspace scope assignments without each
		// service growing a pool dependency.
		func(ctx context.Context, listID uuid.UUID) (uuid.UUID, error) {
			var wsID uuid.UUID
			err := pool.QueryRow(ctx,
				`SELECT s.workspace_id FROM lists l JOIN spaces s ON s.id = l.space_id WHERE l.id = $1`,
				listID,
			).Scan(&wsID)
			return wsID, err
		},
	)
	dSvc := docSvc.New(dRepo, hub, recorder)
	chSvc := chatSvc.New(chatSvc.Deps{
		Channels: chRepo,
		Messages: msgRepo,
		Hub:      hub,
		Notify:   dispatcher,
		Audit:    recorder,
	})
	sSvc := searchSvc.New(sRepo)
	gSvc := goalSvc.New(goalsRepo, targetsRepo, recorder)
	spSprintSvc := sprintSvc.New(sprintsRepo, recorder)
	dashSvc_ := dashboardSvc.New(dashRepo, dashboardSvc.DataProviders{
		Burndown: func(ctx context.Context, sprintID uuid.UUID) ([]domain.BurndownPoint, error) {
			return spSprintSvc.Burndown(ctx, sprintID)
		},
		TaskCount: func(ctx context.Context, listID uuid.UUID) (map[string]int, error) {
			return tRepo.CountByStatus(ctx, listID)
		},
		Velocity: func(ctx context.Context, listID uuid.UUID) ([]dashboardSvc.VelocityPoint, error) {
			rows, err := sprintsRepo.Velocity(ctx, listID, 10)
			if err != nil {
				return nil, err
			}
			out := make([]dashboardSvc.VelocityPoint, 0, len(rows))
			for _, r := range rows {
				out = append(out, dashboardSvc.VelocityPoint{
					SprintID:   r.SprintID,
					SprintName: r.Name,
					EndsAt:     r.EndsAt,
					Completed:  r.Completed,
					Goal:       r.GoalPoints,
				})
			}
			return out, nil
		},
		TimePerUser: func(ctx context.Context, wsID uuid.UUID, from, to *time.Time) ([]domain.TimeReportBucket, error) {
			return teSvc.Report(ctx, domain.TimeEntryFilter{WorkspaceID: &wsID, From: from, To: to})
		},
		MyTasks: func(ctx context.Context, wsID, viewer uuid.UUID) ([]domain.Task, error) {
			if viewer == uuid.Nil {
				return []domain.Task{}, nil
			}
			return tRepo.ListByAssigneeInWorkspace(ctx, wsID, viewer, 50)
		},
		Activity: func(ctx context.Context, wsID uuid.UUID, limit int) ([]domain.AuditEntry, error) {
			if limit <= 0 {
				limit = 25
			}
			return auSvc.List(ctx, domain.AuditFilter{WorkspaceID: &wsID, Limit: limit})
		},
		GoalProgress: func(ctx context.Context, wsID uuid.UUID) ([]dashboardSvc.GoalProgressItem, error) {
			goals, err := gSvc.ListByWorkspace(ctx, wsID)
			if err != nil {
				return nil, err
			}
			out := make([]dashboardSvc.GoalProgressItem, 0, len(goals))
			for _, g := range goals {
				if g.Archived {
					continue
				}
				out = append(out, dashboardSvc.GoalProgressItem{
					GoalID:   g.ID,
					Name:     g.Name,
					DueAt:    g.DueAt,
					Progress: g.Progress,
					OwnerID:  g.OwnerID,
				})
			}
			return out, nil
		},
	}, recorder)
	wbSvc := whiteboardSvc.New(wbRepo, hub, recorder)
	fmSvc := formSvc.New(fmRepo, func(ctx context.Context, listID uuid.UUID, name, description string) (*domain.Task, error) {
		// Public submissions run unauthenticated; use the form's creator as actor
		// if known — otherwise a zero-uuid represents "system". Real auth-gated
		// attribution lands in M8 alongside SSO.
		return tSvc.Create(ctx, uuid.Nil, taskSvc.CreateInput{ListID: listID, Name: name, Description: description})
	}, recorder)
	tplSvc := templateSvc.New(tplRepo, templateSvc.Appliers{
		CreateList: func(ctx context.Context, actor, spaceID uuid.UUID, name string) (*domain.List, error) {
			return lSvc.Create(ctx, listSvc.CreateInput{SpaceID: spaceID, Name: name})
		},
		CreateStatus: func(ctx context.Context, actor, listID uuid.UUID, name, color, category string, order int) error {
			_, err := stSvc.Create(ctx, actor, statusSvc.CreateInput{ListID: listID, Name: name, Color: color, Category: category, OrderIndex: order})
			return err
		},
		CreateTask: func(ctx context.Context, actor, listID uuid.UUID, parentTaskID *uuid.UUID, name, description string) (*domain.Task, error) {
			return tSvc.Create(ctx, actor, taskSvc.CreateInput{ListID: listID, ParentTaskID: parentTaskID, Name: name, Description: description})
		},
		CreateDoc: func(ctx context.Context, actor, workspaceID uuid.UUID, parentID *uuid.UUID, title string, content json.RawMessage) (*domain.Doc, error) {
			return dSvc.Create(ctx, actor, docSvc.CreateInput{WorkspaceID: workspaceID, ParentID: parentID, Title: title, Content: content})
		},
	}, recorder)
	repSvc := reportSvc.New(repRepo, wSvc, reportSvc.Lookups{
		GetWorkspace: func(ctx context.Context, id uuid.UUID) (*domain.Workspace, error) {
			return wsRepo.GetByID(ctx, id)
		},
		GetUser: func(ctx context.Context, id uuid.UUID) (*domain.User, error) {
			return uRepo.GetByID(ctx, id)
		},
	})

	// Automation engine needs callbacks into the services it can already invoke.
	engine := autoeng.New(autoRepo, pool, autoeng.Actions{
		ChangeStatus: func(ctx context.Context, actor, taskID, statusID uuid.UUID) error {
			return tSvc.ChangeStatus(ctx, actor, taskID, statusID)
		},
		AssignUser: func(ctx context.Context, actor, taskID, userID uuid.UUID) error {
			return tSvc.AddAssignee(ctx, actor, taskID, userID)
		},
		AddTag: func(ctx context.Context, actor, taskID, tagID uuid.UUID) error {
			return tgSvc.AttachToTask(ctx, actor, taskID, tagID)
		},
		AddComment: func(ctx context.Context, author, taskID uuid.UUID, body string) error {
			_, err := cSvc.Create(ctx, author, commentSvc.CreateInput{TaskID: taskID, Body: body})
			return err
		},
		Notify: func(ctx context.Context, actor, userID uuid.UUID, kind string, payload map[string]any) {
			a := actor
			dispatcher.Enqueue(notify.Notice{
				Recipient: userID,
				Actor:     &a,
				Kind:      kind,
				Payload:   payload,
			})
		},
	}, logger)
	tSvc.SetEngine(engine)

	// handlers
	uH := userHandler.New(uSvc).
		WithMetrics(metrics).
		WithAdminDeps(policy, mRepo, mSvc)
	wH := workspaceHandler.New(wSvc)
	spH := spaceHandler.New(spSvc, policy)
	fH := folderHandler.New(fSvc, policy)
	lH := listHandler.New(lSvc, policy)
	tH := taskHandler.New(tSvc, policy)
	cH := commentHandler.New(cSvc)
	credH := credentialHandler.New(credSvc)
	stH := statusHandler.New(stSvc)
	tgH := tagHandler.New(tgSvc)
	atH := attachmentHandler.New(atSvc, cfg.AttachmentMaxBytes)
	nH := notificationHandler.New(nSvc)
	auH := auditHandler.NewWithPolicy(auSvc, policy)
	mH := memberHandler.NewWithPolicy(mSvc, policy)
	vH := viewHandler.New(vSvc, policy)
	cfH := customfieldHandler.New(cfSvc)
	teH := timeentryHandler.New(teSvc)
	depH := dependencyHandler.New(depSvc)
	autoH := automationHandler.New(autoSvcImpl, policy)
	dH := docHandler.New(dSvc)
	chH := chatHandler.New(chSvc)
	sH := searchHandler.New(sSvc)
	gH := goalHandler.New(gSvc)
	spH2 := sprintHandler.New(spSprintSvc)
	dashH := dashboardHandler.New(dashSvc_)
	wbH := whiteboardHandler.New(wbSvc)
	fmH := formHandler.New(fmSvc)
	tplH := templateHandler.New(tplSvc)
	repH := reportHandler.New(repSvc)
	mfaH := mfaHandler.New(mfaSvcImpl)
	oidcH := oidcHandler.New(oidcSvcImpl, cfg.OIDCSuccessRedirect, cfg.OIDCFailureRedirect, cfg.OIDCCookieSecure).
		WithSessions(sessSvcImpl)
	gdprH := gdprHandler.New(gdprSvcImpl, policy)
	roleH := roleHandler.New(roleSvcImpl, policy)
	sessH := sessionHandler.New(sessSvcImpl)
	savedSearchH := savedsearchHandler.New(savedSearchSvcImpl)
	milestoneH := milestoneHandler.New(milestoneSvcImpl)
	workloadH := workloadHandler.New(workloadSvcImpl)
	portfolioH := portfolioHandler.New(portfolioSvcImpl)

	r := chi.NewRouter()
	// Order matters here:
	//   1. RequestID stamps every request with a stable id chimw.GetReqID can read.
	//   2. RequestLogger derives a per-request logger that carries that id.
	//   3. RealIP, Recoverer, CORS — unchanged.
	//   4. Sentry middleware (no-op when DSN unset) catches panics.
	//   5. HTTPMetrics records RED for everything below.
	//   6. Logger emits one JSON line per request using the per-request logger.
	r.Use(chimw.RequestID)
	r.Use(middleware.RequestLogger(logger))
	r.Use(chimw.RealIP)
	r.Use(chimw.Recoverer)
	r.Use(corsMiddleware())
	r.Use(sentryMW)
	r.Use(middleware.HTTPMetrics(metrics))
	r.Use(middleware.Logger(logger))

	// Liveness — always 200 unless the binary itself is wedged.
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// Readiness — 200 only when DB is reachable, migrations are at the
	// expected version, and Redis is reachable (if configured). The
	// orchestrator should point its readiness probe at this URL.
	probes := map[string]observability.Probe{
		"postgres":   observability.PgxProbe(pool),
		"migrations": observability.MigrationsProbe(pool, minMigrationVersion),
	}
	if rdbClient != nil {
		probes["redis"] = observability.RedisProbe(rdbClient)
	}
	r.Method(http.MethodGet, "/readyz", observability.HealthHandler(probes, true))

	// Prometheus exposition. Mount under /metrics with no auth — scraper
	// access is restricted at the network layer in production.
	r.Method(http.MethodGet, "/metrics", metrics.Handler())

	r.Route("/api/v1", func(r chi.Router) {
		// API root – returns version info and available endpoint groups.
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"name":    "ClickUp Clone API",
				"version": "v1",
				"status":  "ok",
				"endpoints": []string{
					"/api/v1/auth/login",
					"/api/v1/auth/mfa/verify",
					"/api/v1/auth/oidc/{provider}/login",
					"/api/v1/auth/oidc/{provider}/callback",
					"/api/v1/me",
					"/api/v1/me/mfa",
					"/api/v1/workspaces",
					"/api/v1/spaces",
					"/api/v1/folders",
					"/api/v1/lists",
					"/api/v1/tasks",
					"/api/v1/comments",
					"/api/v1/statuses",
					"/api/v1/tags",
					"/api/v1/attachments",
					"/api/v1/notifications",
					"/api/v1/audit",
					"/api/v1/members",
					"/api/v1/views",
					"/api/v1/custom-fields",
					"/api/v1/time-entries",
					"/api/v1/dependencies",
					"/api/v1/automations",
					"/api/v1/docs",
					"/api/v1/chat",
					"/api/v1/search",
					"/api/v1/goals",
					"/api/v1/sprints",
					"/api/v1/dashboards",
					"/api/v1/whiteboards",
					"/api/v1/forms",
					"/api/v1/templates",
					"/api/v1/reports",
					"/api/v1/credentials",
					"/api/v1/workspaces/{workspaceID}/audit/export",
					"/api/v1/workspaces/{workspaceID}/gdpr/export",
					"/api/v1/permissions",
					"/api/v1/workspaces/{workspaceID}/roles",
					"/api/v1/workspaces/{workspaceID}/users",
					"/api/v1/admin/users",
				},
			})
		})

		r.Group(func(pub chi.Router) {
			pub.Use(middleware.RateLimit(rdbClient, 10, time.Minute))
			uH.PublicRoutes(pub)
			fmH.PublicRoutes(pub)
			oidcH.Routes(pub)
		})

		r.Group(func(pr chi.Router) {
			pr.Use(middleware.Auth(cfg.JWTSecret))
			// SessionGate runs *after* Auth so the jti is in context. Tokens
			// without a jti (legacy, pre-sessions) pass through; tokens whose
			// session row was revoked are rejected with 401.
			pr.Use(middleware.SessionGate(sessSvcImpl, time.Minute))
			pr.Use(middleware.RateLimit(rdbClient, 300, time.Minute))
			uH.PrivateRoutes(pr)
			uH.AdminRoutes(pr)
			uH.AdminGlobalRoutes(pr)
			mfaH.Routes(pr)
			sessH.Routes(pr)
			savedSearchH.Routes(pr)
			milestoneH.Routes(pr)
			workloadH.Routes(pr)
			portfolioH.Routes(pr)
			fmH.PrivateRoutes(pr)
			wbH.Routes(pr)
			tplH.Routes(pr)
			wH.Routes(pr)
			spH.Routes(pr)
			fH.Routes(pr)
			lH.Routes(pr)
			tH.Routes(pr)
			cH.Routes(pr)
			credH.Routes(pr)
			repH.Routes(pr)
			stH.Routes(pr)
			tgH.Routes(pr)
			atH.Routes(pr)
			nH.Routes(pr)
			auH.Routes(pr)
			mH.Routes(pr)
			vH.Routes(pr)
			cfH.Routes(pr)
			teH.Routes(pr)
			depH.Routes(pr)
			autoH.Routes(pr)
			dH.Routes(pr)
			chH.Routes(pr)
			sH.Routes(pr)
			gH.Routes(pr)
			spH2.Routes(pr)
			dashH.Routes(pr)
			gdprH.Routes(pr)
			roleH.Routes(pr)
		})
	})

	// WebSocket is mounted on the top-level ServeMux, ABOVE the chi router, so
	// the upgrade handler receives a raw net/http ResponseWriter that still
	// implements http.Hijacker. Putting /ws behind chi's middleware stack
	// causes one of the wrappers to shadow the Hijacker interface and the
	// upgrade fails with "websocket: response does not implement http.Hijacker".
	// Non-/ws traffic falls through to chi where all HTTP middleware runs.
	root := http.NewServeMux()
	root.HandleFunc("/ws", hub.ServeWS)
	root.Handle("/", r)

	logger.Info("ws mounted on root mux (bypasses chi middleware)", "path", "/ws")

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%s", cfg.AppPort),
		Handler:           root,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		logger.Info("server started", "port", cfg.AppPort)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server error", "err", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
	logger.Info("server stopped")
}

// resolveCredentialsKey returns the 32-byte AES-GCM key for the credentials
// vault. In production CREDENTIALS_ENCRYPTION_KEY (hex, 64 chars) is required;
// in development we derive a deterministic key from JWT_SECRET so a fresh
// checkout boots, but log a loud warning.
func resolveCredentialsKey(cfg *config.Config, logger *slog.Logger) ([]byte, error) {
	if cfg.CredentialsEncryptionKey != "" {
		key, err := hex.DecodeString(cfg.CredentialsEncryptionKey)
		if err != nil {
			return nil, fmt.Errorf("CREDENTIALS_ENCRYPTION_KEY is not valid hex: %w", err)
		}
		if len(key) != cryptox.KeySize {
			return nil, fmt.Errorf("CREDENTIALS_ENCRYPTION_KEY must decode to %d bytes (got %d)", cryptox.KeySize, len(key))
		}
		return key, nil
	}
	if cfg.AppEnv != "development" {
		return nil, errors.New("CREDENTIALS_ENCRYPTION_KEY is required in non-development environments")
	}
	logger.Warn("USING DEV ENCRYPTION KEY for credentials vault — DO NOT USE IN PROD. Set CREDENTIALS_ENCRYPTION_KEY to a 32-byte hex value.")
	// Deterministic dev key derived from the JWT secret. Stable across restarts
	// so previously-stored entries can still be decrypted in dev.
	key := make([]byte, cryptox.KeySize)
	seed := []byte("clickup-credentials-dev-key/" + cfg.JWTSecret)
	for i := range key {
		key[i] = seed[i%len(seed)]
	}
	return key, nil
}

// sessionUserAdapter bridges *sessionSvc.Service to the user service's
// SessionIssuer interface — same type signature, different field names.
// We could rename the user-service input struct to match, but keeping
// each service's input shape local avoids forcing one to depend on the
// other's package.
type sessionUserAdapter struct{ svc *sessionSvc.Service }

func (a sessionUserAdapter) Issue(ctx context.Context, in userSvc.SessionIssueInput) error {
	var ip *netip.Addr
	if in.IP != "" {
		// r.RemoteAddr is "host:port"; strip the port if present.
		host := in.IP
		if h, _, err := net.SplitHostPort(in.IP); err == nil {
			host = h
		}
		if a, err := netip.ParseAddr(host); err == nil {
			ip = &a
		}
	}
	return a.svc.Issue(ctx, sessionSvc.IssueInput{
		JTI:       in.JTI,
		UserID:    in.UserID,
		UserAgent: in.UserAgent,
		IP:        ip,
		ExpiresAt: in.ExpiresAt,
	})
}

func corsMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
