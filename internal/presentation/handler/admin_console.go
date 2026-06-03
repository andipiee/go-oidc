package handler

import (
	"crypto/sha256"
	"encoding/base64"
	"html/template"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/andipiee/go-oidc/internal/application/usecase"
	"github.com/andipiee/go-oidc/internal/domain/entity"
	"github.com/andipiee/go-oidc/internal/domain/repository"
	"github.com/andipiee/go-oidc/internal/infrastructure/auth"
	"github.com/andipiee/go-oidc/internal/presentation/httputil"
	"github.com/google/uuid"
)

// AdminConsoleHandler turns the auth server's own admin console into an OIDC
// relying party. Rather than a shared Basic-Auth password, access is gated by
// the `role` claim on an id_token minted through the normal authorization-code
// + PKCE flow. The console is a public client (token_endpoint_auth_method=none)
// so there is no console secret to manage — PKCE binds the code to this flow.
type AdminConsoleHandler struct {
	jwtService    *auth.JWTService
	tokenUseCase  *usecase.TokenUseCase
	cryptoService *auth.CryptoService
	clientRepo    repository.ClientRepository
	clientID      string
	redirectURI   string
	tmpl          *template.Template
}

func NewAdminConsoleHandler(
	jwtService *auth.JWTService,
	tokenUseCase *usecase.TokenUseCase,
	cryptoService *auth.CryptoService,
	clientRepo repository.ClientRepository,
	clientID string,
	redirectURI string,
	tmpl *template.Template,
) *AdminConsoleHandler {
	return &AdminConsoleHandler{
		jwtService:    jwtService,
		tokenUseCase:  tokenUseCase,
		cryptoService: cryptoService,
		clientRepo:    clientRepo,
		clientID:      clientID,
		redirectURI:   redirectURI,
		tmpl:          tmpl,
	}
}

// dashboardData is the view model for admin.html. NewClientID/NewSecret are set
// only on the response that immediately follows a create or rotate, so the
// plaintext secret is rendered exactly once and never persisted client-side.
type dashboardData struct {
	Clients     []*entity.Client
	NewClientID string
	NewSecret   string
	Error       string
}

// ShowDashboard renders the admin landing page. It is mounted behind
// RequireAdmin, so reaching here means the caller already holds a valid
// admin_session id_token with role=admin.
func (h *AdminConsoleHandler) ShowDashboard(w http.ResponseWriter, r *http.Request) {
	h.renderDashboard(w, r, dashboardData{})
}

// renderDashboard loads the current client list and renders admin.html, merging
// in any flash fields (a freshly minted secret, or an error) from a mutation.
func (h *AdminConsoleHandler) renderDashboard(w http.ResponseWriter, r *http.Request, data dashboardData) {
	clients, err := h.clientRepo.List(r.Context(), 100, 0)
	if err != nil {
		data.Error = "Failed to load clients: " + err.Error()
	} else {
		data.Clients = clients
	}
	httputil.HTML(w, h.tmpl, "admin.html", http.StatusOK, data)
}

// HandleCreateClient handles the create form (POST /admin/clients). On success
// it re-renders the dashboard with the new secret shown once.
func (h *AdminConsoleHandler) HandleCreateClient(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		h.renderDashboard(w, r, dashboardData{Error: "Name is required."})
		return
	}
	authMethod := r.FormValue("token_endpoint_auth_method")
	grants := strings.Split(r.FormValue("grant_types"), ",")

	secret := h.cryptoService.GenerateRandomString(32)
	hash, _ := h.cryptoService.HashPassword(secret)

	client := &entity.Client{
		ID:                      uuid.Must(uuid.NewV7()),
		ClientID:                h.cryptoService.GenerateRandomString(16),
		ClientSecretHash:        hash,
		Name:                    name,
		RedirectURIs:            splitLines(r.FormValue("redirect_uris")),
		GrantTypes:              grants,
		ResponseTypes:           []string{"code"},
		TokenEndpointAuthMethod: authMethod,
		Scopes:                  []string{"openid", "profile", "email"},
		CreatedAt:               time.Now(),
		UpdatedAt:               time.Now(),
	}
	if err := h.clientRepo.Create(r.Context(), client); err != nil {
		h.renderDashboard(w, r, dashboardData{Error: "Failed to create client: " + err.Error()})
		return
	}
	h.renderDashboard(w, r, dashboardData{NewClientID: client.ClientID, NewSecret: secret})
}

// HandleRotateSecret regenerates a client's secret (POST
// /admin/clients/{id}/rotate-secret), invalidating the old one immediately, and
// renders the new plaintext once. A lost secret can only be recovered this way.
func (h *AdminConsoleHandler) HandleRotateSecret(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		h.renderDashboard(w, r, dashboardData{Error: "Invalid client id."})
		return
	}
	client, err := h.clientRepo.GetByID(r.Context(), id)
	if err != nil || client == nil {
		h.renderDashboard(w, r, dashboardData{Error: "Client not found."})
		return
	}

	secret := h.cryptoService.GenerateRandomString(32)
	hash, _ := h.cryptoService.HashPassword(secret)
	client.ClientSecretHash = hash
	client.UpdatedAt = time.Now()
	if err := h.clientRepo.Update(r.Context(), client); err != nil {
		h.renderDashboard(w, r, dashboardData{Error: "Failed to rotate secret: " + err.Error()})
		return
	}
	h.renderDashboard(w, r, dashboardData{NewClientID: client.ClientID, NewSecret: secret})
}

// HandleDeleteClient deletes a client (POST /admin/clients/{id}/delete) then
// redirects back to the dashboard (Post/Redirect/Get).
func (h *AdminConsoleHandler) HandleDeleteClient(w http.ResponseWriter, r *http.Request) {
	if id, err := uuid.Parse(r.PathValue("id")); err == nil {
		h.clientRepo.Delete(r.Context(), id)
	}
	http.Redirect(w, r, "/admin", http.StatusFound)
}

// splitLines returns the non-empty, trimmed lines of s (used to parse the
// redirect-URIs textarea, one URI per line).
func splitLines(s string) []string {
	var out []string
	for _, line := range strings.Split(s, "\n") {
		if t := strings.TrimSpace(line); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// HandleLogin kicks off the OIDC flow: generate PKCE + state, stash them in
// short-lived HttpOnly cookies, and bounce to the authorize endpoint. If the
// caller still has a valid session_id cookie, authorize issues a code without
// re-prompting for a password (silent SSO).
func (h *AdminConsoleHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	verifier := h.cryptoService.GenerateRandomString(64)
	sum := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(sum[:])
	state := h.cryptoService.GenerateRandomString(32)

	secure := r.TLS != nil
	// 10 minutes is plenty to complete login; these are single-use.
	http.SetCookie(w, &http.Cookie{Name: "admin_oauth_state", Value: state, MaxAge: 600, Path: "/admin", Secure: secure, HttpOnly: true})
	http.SetCookie(w, &http.Cookie{Name: "admin_oauth_verifier", Value: verifier, MaxAge: 600, Path: "/admin", Secure: secure, HttpOnly: true})

	authorizeURL := "/oauth2/authorize?" + url.Values{
		"response_type":         {"code"},
		"client_id":             {h.clientID},
		"redirect_uri":          {h.redirectURI},
		"scope":                 {"openid profile email"},
		"state":                 {state},
		"code_challenge":        {challenge},
		"code_challenge_method": {"S256"},
	}.Encode()
	http.Redirect(w, r, authorizeURL, http.StatusFound)
}

// HandleCallback completes the flow: validate state, exchange the code (PKCE,
// no client secret), verify the id_token's role claim, and on success persist
// the id_token as the admin_session cookie.
func (h *AdminConsoleHandler) HandleCallback(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	code := q.Get("code")
	state := q.Get("state")

	stateCookie, err := r.Cookie("admin_oauth_state")
	if err != nil || stateCookie.Value == "" || state == "" || stateCookie.Value != state {
		h.denyHTML(w, "Invalid or expired login state. Please try again.")
		return
	}
	verifierCookie, err := r.Cookie("admin_oauth_verifier")
	if err != nil || verifierCookie.Value == "" {
		h.denyHTML(w, "Missing PKCE verifier. Please try again.")
		return
	}

	// Single-use: clear the transient cookies regardless of outcome.
	secure := r.TLS != nil
	http.SetCookie(w, &http.Cookie{Name: "admin_oauth_state", Value: "", MaxAge: -1, Path: "/admin", HttpOnly: true})
	http.SetCookie(w, &http.Cookie{Name: "admin_oauth_verifier", Value: "", MaxAge: -1, Path: "/admin", HttpOnly: true})

	resp, err := h.tokenUseCase.ExchangeCode(r.Context(), usecase.TokenRequest{
		GrantType:    "authorization_code",
		Code:         code,
		RedirectURI:  h.redirectURI,
		ClientID:     h.clientID,
		CodeVerifier: verifierCookie.Value,
	})
	if err != nil || resp.IDToken == "" {
		h.denyHTML(w, "Login failed: could not exchange authorization code.")
		return
	}

	claims, err := h.jwtService.ValidateToken(resp.IDToken)
	if err != nil {
		h.denyHTML(w, "Login failed: invalid id token.")
		return
	}
	if claims.Role != "admin" {
		h.denyHTML(w, "Access denied: your account does not have the admin role.")
		return
	}

	// Persist the id_token as the admin session. Its own exp (access-token TTL)
	// bounds validity; when it lapses, RequireAdmin redirects back through
	// /admin/login and the still-valid session_id re-issues silently.
	http.SetCookie(w, &http.Cookie{Name: "admin_session", Value: resp.IDToken, MaxAge: 86400, Path: "/admin", Secure: secure, HttpOnly: true})
	http.Redirect(w, r, "/admin", http.StatusFound)
}

// HandleLogout clears the admin session. The underlying user session_id is left
// intact (logout of the console, not of the whole SSO session).
func (h *AdminConsoleHandler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: "admin_session", Value: "", MaxAge: -1, Path: "/admin", HttpOnly: true})
	http.Redirect(w, r, "/admin/login", http.StatusFound)
}

func (h *AdminConsoleHandler) denyHTML(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusForbidden)
	w.Write([]byte(`<!doctype html><html><head><meta charset="utf-8"><title>Admin — Access denied</title>` +
		`<style>body{font-family:system-ui,sans-serif;background:#0f1115;color:#e6e6e6;display:flex;min-height:100vh;align-items:center;justify-content:center;margin:0}` +
		`.card{background:#181b22;border:1px solid #262b36;border-radius:12px;padding:32px 36px;max-width:420px;text-align:center}` +
		`a{color:#6ea8fe;text-decoration:none}h1{font-size:18px;margin:0 0 12px}p{color:#9aa4b2;line-height:1.5}</style></head>` +
		`<body><div class="card"><h1>Access denied</h1><p>` + template.HTMLEscapeString(msg) + `</p>` +
		`<p><a href="/admin/login">Try logging in again</a></p></div></body></html>`))
}
