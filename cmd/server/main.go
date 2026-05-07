package main

import (
	"context"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"encoding/json"

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
	messageRepo "github.com/yourorg/clickup/internal/repository/message"
	listRepo "github.com/yourorg/clickup/internal/repository/list"
	sprintRepo "github.com/yourorg/clickup/internal/repository/sprint"
	memberRepo "github.com/yourorg/clickup/internal/repository/member"
	notificationRepo "github.com/yourorg/clickup/internal/repository/notification"
	reportRepo "github.com/yourorg/clickup/internal/repository/report"
	searchRepo "github.com/yourorg/clickup/internal/repository/search"
	spaceRepo "github.com/yourorg/clickup/internal/repository/space"
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
	goalSvc "github.com/yourorg/clickup/internal/service/goal"
	sprintSvc "github.com/yourorg/clickup/internal/service/sprint"
	listSvc "github.com/yourorg/clickup/internal/service/list"
	memberSvc "github.com/yourorg/clickup/internal/service/member"
	notificationSvc "github.com/yourorg/clickup/internal/service/notification"
	reportSvc "github.com/yourorg/clickup/internal/service/report"
	searchSvc "github.com/yourorg/clickup/internal/service/search"
	spaceSvc "github.com/yourorg/clickup/internal/service/space"
	statusSvc "github.com/yourorg/clickup/internal/service/status"
	tagSvc "github.com/yourorg/clickup/internal/service/tag"
	taskSvc "github.com/yourorg/clickup/internal/service/task"
	templateSvc "github.com/yourorg/clickup/internal/service/template"
	timeentrySvc "github.com/yourorg/clickup/internal/service/timeentry"
	userSvc "github.com/yourorg/clickup/internal/service/user"
	viewSvc "github.com/yourorg/clickup/internal/service/view"
	whiteboardSvc "github.com/yourorg/clickup/internal/service/whiteboard"
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
	formHandler "github.com/yourorg/clickup/internal/handler/form"
	goalHandler "github.com/yourorg/clickup/internal/handler/goal"
	sprintHandler "github.com/yourorg/clickup/internal/handler/sprint"
	folderHandler "github.com/yourorg/clickup/internal/handler/folder"
	listHandler "github.com/yourorg/clickup/internal/handler/list"
	memberHandler "github.com/yourorg/clickup/internal/handler/member"
	notificationHandler "github.com/yourorg/clickup/internal/handler/notification"
	reportHandler "github.com/yourorg/clickup/internal/handler/report"
	searchHandler "github.com/yourorg/clickup/internal/handler/search"
	spaceHandler "github.com/yourorg/clickup/internal/handler/space"
	statusHandler "github.com/yourorg/clickup/internal/handler/status"
	tagHandler "github.com/yourorg/clickup/internal/handler/tag"
	taskHandler "github.com/yourorg/clickup/internal/handler/task"
	templateHandler "github.com/yourorg/clickup/internal/handler/template"
	timeentryHandler "github.com/yourorg/clickup/internal/handler/timeentry"
	userHandler "github.com/yourorg/clickup/internal/handler/user"
	viewHandler "github.com/yourorg/clickup/internal/handler/view"
	whiteboardHandler "github.com/yourorg/clickup/internal/handler/whiteboard"
	workspaceHandler "github.com/yourorg/clickup/internal/handler/workspace"
)

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

	pool, err := db.NewPostgres(cfg.DBDSN)
	if err != nil {
		logger.Error("postgres connect failed", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

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
			"http://localhost:5173":  true, // vite dev
			"http://127.0.0.1:5173":  true,
			"http://localhost:8080":  true, // served by Go directly
			"http://127.0.0.1:8080":  true,
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

	// cross-cutting infra
	dispatcher := notify.New(nRepo, hub, logger)
	recorder := audit.New(auRepo, logger)
	policy := authz.New(pool)

	// services
	uSvc := userSvc.New(uRepo, cfg.JWTSecret)
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
	autoSvcImpl := automationSvc.New(autoRepo, recorder)
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
	uH := userHandler.New(uSvc)
	wH := workspaceHandler.New(wSvc)
	spH := spaceHandler.New(spSvc, policy)
	fH := folderHandler.New(fSvc, policy)
	lH := listHandler.New(lSvc, policy)
	tH := taskHandler.New(tSvc, policy)
	cH := commentHandler.New(cSvc, policy)
	credH := credentialHandler.New(credSvc)
	stH := statusHandler.New(stSvc)
	tgH := tagHandler.New(tgSvc)
	atH := attachmentHandler.New(atSvc)
	nH := notificationHandler.New(nSvc)
	auH := auditHandler.New(auSvc)
	mH := memberHandler.New(mSvc)
	vH := viewHandler.New(vSvc, policy)
	cfH := customfieldHandler.New(cfSvc)
	teH := timeentryHandler.New(teSvc)
	depH := dependencyHandler.New(depSvc)
	autoH := automationHandler.New(autoSvcImpl)
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

	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Recoverer)
	r.Use(corsMiddleware())
	r.Use(middleware.Logger(logger))

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	r.Route("/api/v1", func(r chi.Router) {
		r.Group(func(pub chi.Router) {
			uH.PublicRoutes(pub)
			fmH.PublicRoutes(pub)
		})

		r.Group(func(pr chi.Router) {
			pr.Use(middleware.Auth(cfg.JWTSecret))
			pr.Use(middleware.RateLimit(rdbClient, 300, time.Minute))
			uH.PrivateRoutes(pr)
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
