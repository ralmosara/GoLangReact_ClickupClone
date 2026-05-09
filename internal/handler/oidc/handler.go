// Package oidc serves the OIDC login & callback endpoints. State is
// validated via a short-lived signed cookie (HttpOnly + SameSite=Lax) so
// the callback can detect CSRF.
package oidc

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/yourorg/clickup/internal/httpx"
	oidcsvc "github.com/yourorg/clickup/internal/service/oidc"
	ssvc "github.com/yourorg/clickup/internal/service/session"
)

const stateCookieName = "oidc_state"

// SessionIssuer is the optional dep that lets the OIDC callback record a
// session row alongside the freshly-minted JWT, so OIDC logins also show
// up in Settings → Active Sessions. Wired post-construction by main.
type SessionIssuer interface {
	Issue(ctx context.Context, in ssvc.IssueInput) error
}

type Handler struct {
	svc           *oidcsvc.Service
	successRedir  string // frontend URL the user lands on with the token in fragment
	failureRedir  string // frontend URL on failure (no token)
	cookieSecure  bool   // true in production (HTTPS); false in dev
	sessions      SessionIssuer
}

// New builds the handler. successRedirect is e.g. "http://localhost:5173/auth/callback"
// — the page that reads `window.location.hash` and stuffs the token into
// the auth store. failureRedirect can be the same page (it inspects the
// query string for `error`).
func New(svc *oidcsvc.Service, successRedirect, failureRedirect string, cookieSecure bool) *Handler {
	if successRedirect == "" {
		successRedirect = "/"
	}
	if failureRedirect == "" {
		failureRedirect = successRedirect
	}
	return &Handler{
		svc:          svc,
		successRedir: successRedirect,
		failureRedir: failureRedirect,
		cookieSecure: cookieSecure,
	}
}

// WithSessions wires the session issuer so OIDC logins persist a row.
// Optional — without it the JWT still works (the auth middleware
// tolerates missing sessions), the user just won't appear in their own
// Active Sessions list for that browser until next non-OIDC login.
func (h *Handler) WithSessions(s SessionIssuer) *Handler {
	h.sessions = s
	return h
}

func (h *Handler) Routes(r chi.Router) {
	r.Get("/auth/oidc/providers", h.listProviders)
	r.Get("/auth/oidc/{provider}/login", h.login)
	r.Get("/auth/oidc/{provider}/callback", h.callback)
}

// listProviders surfaces the configured providers to the frontend so the
// login page can show "Sign in with Google" only when it's actually
// configured. Returns names only — no client IDs or secrets.
func (h *Handler) listProviders(w http.ResponseWriter, r *http.Request) {
	httpx.JSON(w, http.StatusOK, map[string]any{
		"providers": h.svc.ConfiguredProviders(),
	})
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	provider := strings.ToLower(chi.URLParam(r, "provider"))
	if !h.svc.Configured(provider) {
		http.NotFound(w, r)
		return
	}
	authURL, state, err := h.svc.AuthorizeURL(r.Context(), provider)
	if err != nil {
		httpx.Fail(w, r, http.StatusInternalServerError, "oidc init failed", err)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     stateCookieName,
		Value:    state,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   600, // 10 minutes — plenty for an interactive IdP roundtrip
	})
	http.Redirect(w, r, authURL, http.StatusFound)
}

func (h *Handler) callback(w http.ResponseWriter, r *http.Request) {
	provider := strings.ToLower(chi.URLParam(r, "provider"))
	if !h.svc.Configured(provider) {
		http.NotFound(w, r)
		return
	}

	q := r.URL.Query()
	if errMsg := q.Get("error"); errMsg != "" {
		h.redirectFail(w, r, "idp_error: "+errMsg)
		return
	}
	state := q.Get("state")
	code := q.Get("code")
	if state == "" || code == "" {
		h.redirectFail(w, r, "missing_state_or_code")
		return
	}

	// Verify the state cookie matches the URL state.
	c, err := r.Cookie(stateCookieName)
	if err != nil || c.Value == "" || c.Value != state {
		h.redirectFail(w, r, "state_mismatch")
		return
	}
	// Burn the cookie so a replay can't reuse it.
	http.SetCookie(w, &http.Cookie{
		Name:     stateCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})

	res, err := h.svc.Callback(r.Context(), provider, code)
	if err != nil {
		// Don't leak provider error detail to the frontend — keep the
		// reason in the structured log via httpx.Fail's audit shape via
		// the request-scoped logger.
		_ = err // logged downstream by the service via slog
		h.redirectFail(w, r, "callback_failed")
		return
	}

	// Persist the session row so this OIDC login appears in the Active
	// Sessions UI alongside password logins. Best-effort: a failure
	// here is logged-and-ignored so a session-table outage doesn't
	// block the user from completing the login.
	if h.sessions != nil && res.User != nil {
		if jti, err := uuid.Parse(res.JTI); err == nil && jti != uuid.Nil {
			_ = h.sessions.Issue(r.Context(), ssvc.IssueInput{
				JTI:       jti,
				UserID:    res.User.ID,
				UserAgent: r.UserAgent(),
				ExpiresAt: res.ExpiresAt,
			})
		}
	}

	// Hand the token to the frontend in the URL fragment so it never
	// hits the server log via the access-log layer.
	dst := h.successRedir
	if !strings.Contains(dst, "#") {
		dst += "#"
	} else {
		dst += "&"
	}
	dst += "token=" + url.QueryEscape(res.Token)
	http.Redirect(w, r, dst, http.StatusFound)
}

func (h *Handler) redirectFail(w http.ResponseWriter, r *http.Request, reason string) {
	dst := h.failureRedir
	if strings.Contains(dst, "?") {
		dst += "&error=" + url.QueryEscape(reason)
	} else {
		dst += "?error=" + url.QueryEscape(reason)
	}
	http.Redirect(w, r, dst, http.StatusFound)
}
