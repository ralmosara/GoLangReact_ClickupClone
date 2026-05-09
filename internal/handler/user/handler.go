package user

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/yourorg/clickup/internal/authz"
	"github.com/yourorg/clickup/internal/domain"
	"github.com/yourorg/clickup/internal/httpx"
	"github.com/yourorg/clickup/internal/middleware"
	memberSvc "github.com/yourorg/clickup/internal/service/member"
	usersvc "github.com/yourorg/clickup/internal/service/user"
)

// metricsRecorder is the slice of *observability.Metrics this handler uses.
// Defining it as an interface here avoids a build-time dep on the metrics
// package being non-nil at construction.
type metricsRecorder interface {
	RecordAuthAttempt(event, outcome string)
}

// Handler wires the public auth routes (/auth/login, /auth/mfa/verify),
// the private /me route, and the workspace-scoped User Management admin
// routes (/workspaces/{id}/users/...). metrics may be nil — when nil, no
// auth events are emitted to Prometheus.
type Handler struct {
	svc     *usersvc.Service
	members domain.MemberRepo // used by AdminRoutes to gate "target user must be in this workspace"
	memSvc  *memberSvc.Service // used to add the freshly-created user as a workspace member
	policy  authz.Policy       // used by AdminRoutes for permission checks
	metrics metricsRecorder
}

func New(svc *usersvc.Service) *Handler { return &Handler{svc: svc} }

// WithMetrics attaches a metrics recorder. main.go composes a small
// adapter around *observability.Metrics so this handler doesn't have to
// import the metrics type directly.
func (h *Handler) WithMetrics(m metricsRecorder) *Handler {
	h.metrics = m
	return h
}

// WithAdminDeps attaches the dependencies that AdminRoutes needs. main.go
// is the only caller. Passing them post-construction keeps `New` shape-
// compatible with existing call sites that only need the public/private
// routes (e.g. the integration tests).
func (h *Handler) WithAdminDeps(policy authz.Policy, members domain.MemberRepo, mSvc *memberSvc.Service) *Handler {
	h.policy = policy
	h.members = members
	h.memSvc = mSvc
	return h
}

func (h *Handler) PublicRoutes(r chi.Router) {
	r.Post("/auth/login", h.login)
	r.Post("/auth/mfa/verify", h.mfaVerify)
}

func (h *Handler) PrivateRoutes(r chi.Router) {
	r.Get("/me", h.me)
	r.Patch("/me", h.updateMe)
	r.Post("/me/password", h.changeMyPassword)
}

// AdminRoutes mounts the User Management admin endpoints. Each handler
// requires `user.manage` on the workspace named in the URL, and the
// per-user mutation handlers additionally require the target user to be
// a member of that workspace (multi-tenant isolation — a user with
// user.manage on workspace A cannot touch a user who only belongs to
// workspace B).
//
// Caller must have invoked WithAdminDeps before reaching here; if not,
// the routes 500 with "admin deps not wired" so the misconfiguration
// shows up at the first request rather than silently returning success.
func (h *Handler) AdminRoutes(r chi.Router) {
	r.Get("/workspaces/{workspaceID}/users", h.adminList)
	r.Post("/workspaces/{workspaceID}/users", h.adminCreate)
	r.Patch("/workspaces/{workspaceID}/users/{userID}", h.adminUpdate)
	r.Post("/workspaces/{workspaceID}/users/{userID}/reset-password", h.adminResetPassword)
	r.Delete("/workspaces/{workspaceID}/users/{userID}", h.adminDelete)
}

// AdminGlobalRoutes mounts the cross-workspace User Management endpoints.
// Authorization is `user.manage on AT LEAST ONE workspace` — a workspace
// owner can therefore create accounts that don't yet belong to any
// workspace (the user logs in, lands on /workspaces with an empty list,
// and is later invited via Members or User Management). Every endpoint
// does its own RequirePermissionAnywhere check via globalAdminContext.
//
// These routes are surfaced on the WorkspacesPage's "Manage Users" nav
// button so admins don't have to enter a workspace just to onboard a new
// account.
func (h *Handler) AdminGlobalRoutes(r chi.Router) {
	r.Get("/admin/users", h.adminGlobalList)
	r.Post("/admin/users", h.adminGlobalCreate)
	r.Patch("/admin/users/{userID}", h.adminGlobalUpdate)
	r.Post("/admin/users/{userID}/reset-password", h.adminGlobalResetPassword)
	r.Delete("/admin/users/{userID}", h.adminGlobalDelete)
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var in usersvc.LoginInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Err(w, http.StatusBadRequest, err.Error())
		return
	}
	res, err := h.svc.LoginWithContext(r.Context(), in, loginContextFrom(r))
	if err != nil {
		h.recordAuth("login", "denied")
		httpx.Err(w, http.StatusUnauthorized, err.Error())
		return
	}
	if res.MFARequired {
		h.recordAuth("login", "mfa_required")
	} else {
		h.recordAuth("login", "ok")
	}
	httpx.JSON(w, http.StatusOK, res)
}

// mfaVerify exchanges a (challenge_token, code) pair for a session token.
// Called once the client has a challenge_token from /auth/login.
func (h *Handler) mfaVerify(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ChallengeToken string `json:"challenge_token"`
		Code           string `json:"code"`
	}
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Err(w, http.StatusBadRequest, err.Error())
		return
	}
	res, err := h.svc.VerifyMFAChallengeWithContext(r.Context(), in.ChallengeToken, in.Code, loginContextFrom(r))
	if err != nil {
		h.recordAuth("mfa_verify", "denied")
		httpx.Err(w, http.StatusUnauthorized, "invalid challenge or code")
		return
	}
	h.recordAuth("mfa_verify", "ok")
	httpx.JSON(w, http.StatusOK, res)
}

// loginContextFrom extracts the User-Agent + remote IP from the request
// for the session row. Trusts the X-Forwarded-For header set by the
// chi RealIP middleware (already mounted in main).
func loginContextFrom(r *http.Request) usersvc.LoginContext {
	return usersvc.LoginContext{
		UserAgent: r.UserAgent(),
		IP:        r.RemoteAddr,
	}
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	uid, ok := middleware.UserID(r.Context())
	if !ok {
		httpx.Err(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	u, err := h.svc.Me(r.Context(), uid)
	if err != nil {
		httpx.Fail(w, r, http.StatusInternalServerError, "internal error", err)
		return
	}
	if u == nil {
		httpx.Err(w, http.StatusNotFound, "not found")
		return
	}
	httpx.JSON(w, http.StatusOK, u)
}

// updateMe is the user-self profile editor — currently just `name`.
// Email is intentionally immutable in this iteration: changing it would
// require re-verification, which the product hasn't shipped yet.
func (h *Handler) updateMe(w http.ResponseWriter, r *http.Request) {
	uid, ok := middleware.UserID(r.Context())
	if !ok {
		httpx.Err(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var in struct {
		Name string `json:"name"`
	}
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Err(w, http.StatusBadRequest, err.Error())
		return
	}
	u, err := h.svc.UpdateName(r.Context(), uid, in.Name)
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, u)
}

// changeMyPassword is the user-self password change. Requires the
// current password — not strictly necessary against a hijacked session
// but a baseline expectation for any "Change password" UI.
func (h *Handler) changeMyPassword(w http.ResponseWriter, r *http.Request) {
	uid, ok := middleware.UserID(r.Context())
	if !ok {
		httpx.Err(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var in struct {
		Current string `json:"current_password"`
		New     string `json:"new_password"`
	}
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Err(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.svc.ChangeOwnPassword(r.Context(), uid, in.Current, in.New); err != nil {
		// Map known sentinels to user-facing codes; everything else is a 400.
		switch {
		case errors.Is(err, usersvc.ErrInvalidCredentials):
			httpx.Err(w, http.StatusUnauthorized, "current password is incorrect")
		case errors.Is(err, usersvc.ErrWeakPassword):
			httpx.Err(w, http.StatusBadRequest, err.Error())
		default:
			httpx.Fail(w, r, http.StatusInternalServerError, "internal error", err)
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) recordAuth(event, outcome string) {
	if h.metrics != nil {
		h.metrics.RecordAuthAttempt(event, outcome)
	}
}

// ── User Management (admin) handlers ───────────────────────────────────

// adminContext extracts the calling user, the workspace from the URL,
// confirms admin deps are wired, and runs the user.manage permission
// check. Returns false (after writing the response) when any check
// fails.
func (h *Handler) adminContext(w http.ResponseWriter, r *http.Request) (uuid.UUID, uuid.UUID, bool) {
	if h.policy == nil || h.members == nil || h.memSvc == nil {
		httpx.Err(w, http.StatusInternalServerError, "admin deps not wired")
		return uuid.Nil, uuid.Nil, false
	}
	uid, ok := middleware.UserID(r.Context())
	if !ok {
		httpx.Err(w, http.StatusUnauthorized, "unauthorized")
		return uuid.Nil, uuid.Nil, false
	}
	wsID, err := uuid.Parse(chi.URLParam(r, "workspaceID"))
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, "bad workspaceID")
		return uuid.Nil, uuid.Nil, false
	}
	if err := h.policy.RequirePermission(r.Context(), wsID, uid, authz.PermUserManage); err != nil {
		httpx.Err(w, http.StatusForbidden, "forbidden")
		return uuid.Nil, uuid.Nil, false
	}
	return uid, wsID, true
}

// requireTargetMember confirms the URL's `userID` parameter belongs to a
// member of the workspace the caller has user.manage on. Cross-tenant
// management is intentionally blocked.
func (h *Handler) requireTargetMember(w http.ResponseWriter, r *http.Request, wsID uuid.UUID) (uuid.UUID, bool) {
	target, err := uuid.Parse(chi.URLParam(r, "userID"))
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, "bad userID")
		return uuid.Nil, false
	}
	isMember, err := h.members.IsWorkspaceMember(r.Context(), wsID, target)
	if err != nil {
		httpx.Fail(w, r, http.StatusInternalServerError, "internal error", err)
		return uuid.Nil, false
	}
	if !isMember {
		httpx.Err(w, http.StatusNotFound, "user is not a member of this workspace")
		return uuid.Nil, false
	}
	return target, true
}

func (h *Handler) adminList(w http.ResponseWriter, r *http.Request) {
	_, wsID, ok := h.adminContext(w, r)
	if !ok {
		return
	}
	users, err := h.svc.ListWorkspaceUsers(r.Context(), wsID)
	if err != nil {
		httpx.Fail(w, r, http.StatusInternalServerError, "internal error", err)
		return
	}
	if users == nil {
		users = []domain.WorkspaceUser{}
	}
	httpx.JSON(w, http.StatusOK, users)
}

type adminCreateInput struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

// adminCreate creates a new user account AND adds them as a member of the
// current workspace. Two operations look atomic to the API caller; on
// failure we leave the user account in place rather than rolling back —
// they can be retried into the workspace via the normal invite flow.
func (h *Handler) adminCreate(w http.ResponseWriter, r *http.Request) {
	actor, wsID, ok := h.adminContext(w, r)
	if !ok {
		return
	}
	var in adminCreateInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Err(w, http.StatusBadRequest, err.Error())
		return
	}
	if in.Role == "" {
		in.Role = "member"
	}

	user, err := h.svc.AdminCreate(r.Context(), usersvc.AdminCreateInput{
		Email:    in.Email,
		Password: in.Password,
		Name:     in.Name,
	})
	if err != nil {
		switch {
		case errors.Is(err, usersvc.ErrEmailTaken):
			httpx.Err(w, http.StatusConflict, "an account already exists with this email")
		case errors.Is(err, usersvc.ErrWeakPassword):
			httpx.Err(w, http.StatusBadRequest, "password must be at least 8 characters")
		case errors.Is(err, usersvc.ErrEmailRequired):
			httpx.Err(w, http.StatusBadRequest, "email is required")
		default:
			httpx.Fail(w, r, http.StatusInternalServerError, "internal error", err)
		}
		return
	}

	// Add to workspace as a member. Reuses the existing membership repo
	// path which writes both the legacy `role` text and the role_id FK.
	if _, err := h.memSvc.AddWorkspaceMember(r.Context(), actor, wsID, memberSvc.AddInput{
		Email: user.Email,
		Role:  in.Role,
	}); err != nil {
		// User exists but membership add failed — surface the error and
		// let the operator follow up. The user account is still usable.
		httpx.Fail(w, r, http.StatusInternalServerError, "user created but workspace add failed", err)
		return
	}

	// Re-fetch via ListByWorkspace so the response shape matches GET — the
	// frontend can drop the new row straight into the table.
	rows, _ := h.svc.ListWorkspaceUsers(r.Context(), wsID)
	for i := range rows {
		if rows[i].ID == user.ID {
			httpx.JSON(w, http.StatusCreated, rows[i])
			return
		}
	}
	// Fallback if the listing query misses (shouldn't happen): return the
	// minimal user row.
	httpx.JSON(w, http.StatusCreated, user)
}

type adminUpdateInput struct {
	Name *string `json:"name"`
}

func (h *Handler) adminUpdate(w http.ResponseWriter, r *http.Request) {
	_, wsID, ok := h.adminContext(w, r)
	if !ok {
		return
	}
	target, ok := h.requireTargetMember(w, r, wsID)
	if !ok {
		return
	}
	var in adminUpdateInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Err(w, http.StatusBadRequest, err.Error())
		return
	}
	if in.Name == nil {
		httpx.Err(w, http.StatusBadRequest, "no fields to update")
		return
	}
	user, err := h.svc.UpdateName(r.Context(), target, *in.Name)
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, user)
}

type adminResetPasswordInput struct {
	Password string `json:"password"`
}

func (h *Handler) adminResetPassword(w http.ResponseWriter, r *http.Request) {
	_, wsID, ok := h.adminContext(w, r)
	if !ok {
		return
	}
	target, ok := h.requireTargetMember(w, r, wsID)
	if !ok {
		return
	}
	var in adminResetPasswordInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Err(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.svc.ResetPassword(r.Context(), target, in.Password); err != nil {
		if errors.Is(err, usersvc.ErrWeakPassword) {
			httpx.Err(w, http.StatusBadRequest, "password must be at least 8 characters")
			return
		}
		httpx.Fail(w, r, http.StatusInternalServerError, "internal error", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── Global admin handlers (no workspace in the URL) ────────────────────

// globalAdminContext authorizes the caller for the cross-workspace
// admin endpoints. They must be authenticated and hold user.manage on
// at least one workspace. Returns the actor id; writes the response
// itself when any check fails.
func (h *Handler) globalAdminContext(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	if h.policy == nil {
		httpx.Err(w, http.StatusInternalServerError, "admin deps not wired")
		return uuid.Nil, false
	}
	uid, ok := middleware.UserID(r.Context())
	if !ok {
		httpx.Err(w, http.StatusUnauthorized, "unauthorized")
		return uuid.Nil, false
	}
	// STRICT role check — only the literal `admin` role gets through.
	// Owners (who have the user.manage permission set) are intentionally
	// excluded so this surface matches its name. If you need a looser
	// gate, swap back to RequirePermissionAnywhere(PermUserManage).
	if err := h.policy.RequireRoleAnywhere(r.Context(), uid, authz.RoleAdmin); err != nil {
		httpx.Err(w, http.StatusForbidden, "forbidden")
		return uuid.Nil, false
	}
	return uid, true
}

type adminGlobalListResponse struct {
	Users  []domain.User `json:"users"`
	Total  int           `json:"total"`
	Limit  int           `json:"limit"`
	Offset int           `json:"offset"`
}

func (h *Handler) adminGlobalList(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.globalAdminContext(w, r); !ok {
		return
	}
	limit := parseIntQuery(r, "limit", 50)
	offset := parseIntQuery(r, "offset", 0)
	users, total, err := h.svc.ListAll(r.Context(), limit, offset)
	if err != nil {
		httpx.Fail(w, r, http.StatusInternalServerError, "internal error", err)
		return
	}
	if users == nil {
		users = []domain.User{}
	}
	httpx.JSON(w, http.StatusOK, adminGlobalListResponse{
		Users:  users,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	})
}

// adminGlobalCreate creates a new user account WITHOUT attaching them to
// any workspace. The newly-created user logs in successfully but lands
// on a /workspaces page with an empty list until an admin invites them
// via Members or via the workspace-scoped User Management page.
func (h *Handler) adminGlobalCreate(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.globalAdminContext(w, r); !ok {
		return
	}
	var in struct {
		Email    string `json:"email"`
		Name     string `json:"name"`
		Password string `json:"password"`
	}
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Err(w, http.StatusBadRequest, err.Error())
		return
	}
	user, err := h.svc.AdminCreate(r.Context(), usersvc.AdminCreateInput{
		Email:    in.Email,
		Password: in.Password,
		Name:     in.Name,
	})
	if err != nil {
		switch {
		case errors.Is(err, usersvc.ErrEmailTaken):
			httpx.Err(w, http.StatusConflict, "an account already exists with this email")
		case errors.Is(err, usersvc.ErrWeakPassword):
			httpx.Err(w, http.StatusBadRequest, "password must be at least 8 characters")
		case errors.Is(err, usersvc.ErrEmailRequired):
			httpx.Err(w, http.StatusBadRequest, "email is required")
		default:
			httpx.Fail(w, r, http.StatusInternalServerError, "internal error", err)
		}
		return
	}
	httpx.JSON(w, http.StatusCreated, user)
}

func (h *Handler) adminGlobalUpdate(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.globalAdminContext(w, r); !ok {
		return
	}
	target, err := uuid.Parse(chi.URLParam(r, "userID"))
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, "bad userID")
		return
	}
	var in adminUpdateInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Err(w, http.StatusBadRequest, err.Error())
		return
	}
	if in.Name == nil {
		httpx.Err(w, http.StatusBadRequest, "no fields to update")
		return
	}
	user, err := h.svc.UpdateName(r.Context(), target, *in.Name)
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, user)
}

func (h *Handler) adminGlobalResetPassword(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.globalAdminContext(w, r); !ok {
		return
	}
	target, err := uuid.Parse(chi.URLParam(r, "userID"))
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, "bad userID")
		return
	}
	var in adminResetPasswordInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Err(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.svc.ResetPassword(r.Context(), target, in.Password); err != nil {
		if errors.Is(err, usersvc.ErrWeakPassword) {
			httpx.Err(w, http.StatusBadRequest, "password must be at least 8 characters")
			return
		}
		httpx.Fail(w, r, http.StatusInternalServerError, "internal error", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) adminGlobalDelete(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.globalAdminContext(w, r)
	if !ok {
		return
	}
	target, err := uuid.Parse(chi.URLParam(r, "userID"))
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, "bad userID")
		return
	}
	if target == actor {
		httpx.Err(w, http.StatusBadRequest, "cannot delete your own account from here")
		return
	}
	if err := h.svc.Delete(r.Context(), target); err != nil {
		httpx.Fail(w, r, http.StatusInternalServerError, "internal error", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// parseIntQuery is a tiny helper to keep the global-list handler tidy.
// Returns the fallback when the query parameter is missing OR malformed.
func parseIntQuery(r *http.Request, name string, fallback int) int {
	v := r.URL.Query().Get(name)
	if v == "" {
		return fallback
	}
	var n int
	if _, err := fmt.Sscanf(v, "%d", &n); err != nil {
		return fallback
	}
	return n
}

func (h *Handler) adminDelete(w http.ResponseWriter, r *http.Request) {
	actor, wsID, ok := h.adminContext(w, r)
	if !ok {
		return
	}
	target, ok := h.requireTargetMember(w, r, wsID)
	if !ok {
		return
	}
	// Prevent self-deletion of the workspace owner via this endpoint —
	// it would orphan the workspace. (We could also block self-deletion
	// outright, but an admin removing their own admin account is a
	// legitimate scenario.)
	if target == actor {
		httpx.Err(w, http.StatusBadRequest, "cannot delete your own account from here")
		return
	}
	if err := h.svc.Delete(r.Context(), target); err != nil {
		httpx.Fail(w, r, http.StatusInternalServerError, "internal error", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
