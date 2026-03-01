package handler

import (
	"net/http"
	"time"

	"github.com/andipiee/go-oidc/internal/domain/entity"
	"github.com/andipiee/go-oidc/internal/domain/repository"
	"github.com/andipiee/go-oidc/internal/infrastructure/auth"
	"github.com/andipiee/go-oidc/internal/presentation/httputil"
	"github.com/google/uuid"
)

type DiscoveryHandler struct {
	issuer     string
	port       int
	jwtService *auth.JWTService
}

func NewDiscoveryHandler(issuer string, port int, jwtService *auth.JWTService) *DiscoveryHandler {
	return &DiscoveryHandler{issuer: issuer, port: port, jwtService: jwtService}
}

func (h *DiscoveryHandler) HandleDiscovery(w http.ResponseWriter, r *http.Request) {
	httputil.JSON(w, http.StatusOK, map[string]any{
		"issuer":                                h.issuer,
		"authorization_endpoint":                h.issuer + "/oauth2/authorize",
		"token_endpoint":                        h.issuer + "/oauth2/token",
		"userinfo_endpoint":                     h.issuer + "/oidc/userinfo",
		"jwks_uri":                              h.issuer + "/.well-known/jwks.json",
		"registration_endpoint":                 h.issuer + "/oidc/register",
		"revocation_endpoint":                   h.issuer + "/oauth2/revoke",
		"introspection_endpoint":                h.issuer + "/oauth2/introspect",
		"device_authorization_endpoint":         h.issuer + "/oidc/device/authorize",
		"scopes_supported":                      []string{"openid", "profile", "email", "offline_access"},
		"response_types_supported":              []string{"code"},
		"response_modes_supported":              []string{"query", "form_post"},
		"grant_types_supported":                 []string{"authorization_code", "refresh_token", "urn:ietf:params:oauth:grant-type:device_code"},
		"token_endpoint_auth_methods_supported": []string{"client_secret_basic", "client_secret_post", "none"},
		"subject_types_supported":               []string{"public"},
		"id_token_signing_alg_values_supported": []string{"RS256"},
		"code_challenge_methods_supported":      []string{"S256"},
		"claims_supported":                      []string{"sub", "name", "email", "email_verified", "picture"},
		"claims_parameter_supported":            false,
		"request_parameter_supported":           false,
		"request_uri_parameter_supported":       false,
	})
}

func (h *DiscoveryHandler) HandleJWKS(w http.ResponseWriter, r *http.Request) {
	httputil.JSON(w, http.StatusOK, h.jwtService.GetJWKS())
}

type AdminHandler struct {
	userRepo   repository.UserRepository
	clientRepo repository.ClientRepository
	config     AdminConfig
}

type AdminConfig struct {
	Enabled  bool
	Username string
	Password string
}

func NewAdminHandler(userRepo repository.UserRepository, clientRepo repository.ClientRepository, config AdminConfig) *AdminHandler {
	return &AdminHandler{
		userRepo:   userRepo,
		clientRepo: clientRepo,
		config:     config,
	}
}

func (h *AdminHandler) HandleIndex(w http.ResponseWriter, r *http.Request) {
	httputil.JSON(w, http.StatusOK, map[string]any{"message": "Admin API", "endpoints": []string{"/admin/users", "/admin/clients"}})
}

func (h *AdminHandler) HandleListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.userRepo.List(r.Context(), 100, 0)
	if err != nil {
		httputil.Error(w, http.StatusInternalServerError, err.Error(), "")
		return
	}
	httputil.JSON(w, http.StatusOK, users)
}

func (h *AdminHandler) HandleCreateUser(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Name     string `json:"name"`
		Password string `json:"password"`
	}
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, err.Error(), "")
		return
	}

	crypto := auth.NewCryptoService()
	hash, _ := crypto.HashPassword(req.Password)

	user := &entity.User{
		ID:            uuid.Must(uuid.NewV7()),
		Email:         req.Email,
		PasswordHash:  hash,
		Name:          req.Name,
		EmailVerified: true,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := h.userRepo.Create(r.Context(), user); err != nil {
		httputil.Error(w, http.StatusInternalServerError, err.Error(), "")
		return
	}

	httputil.JSON(w, http.StatusCreated, user)
}

func (h *AdminHandler) HandleDeleteUser(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid id", "")
		return
	}

	if err := h.userRepo.Delete(r.Context(), id); err != nil {
		httputil.Error(w, http.StatusInternalServerError, err.Error(), "")
		return
	}

	httputil.JSON(w, http.StatusOK, map[string]any{})
}

func (h *AdminHandler) HandleListClients(w http.ResponseWriter, r *http.Request) {
	clients, err := h.clientRepo.List(r.Context(), 100, 0)
	if err != nil {
		httputil.Error(w, http.StatusInternalServerError, err.Error(), "")
		return
	}
	httputil.JSON(w, http.StatusOK, clients)
}

func (h *AdminHandler) HandleCreateClient(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name                    string   `json:"name"`
		RedirectURIs            []string `json:"redirect_uris"`
		GrantTypes              []string `json:"grant_types"`
		TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
	}
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, err.Error(), "")
		return
	}

	crypto := auth.NewCryptoService()
	clientSecret := crypto.GenerateRandomString(32)
	clientSecretHash, _ := crypto.HashPassword(clientSecret)

	client := &entity.Client{
		ID:                      uuid.Must(uuid.NewV7()),
		ClientID:                crypto.GenerateRandomString(16),
		ClientSecretHash:        clientSecretHash,
		Name:                    req.Name,
		RedirectURIs:            req.RedirectURIs,
		GrantTypes:              req.GrantTypes,
		TokenEndpointAuthMethod: req.TokenEndpointAuthMethod,
		CreatedAt:               time.Now(),
		UpdatedAt:               time.Now(),
	}

	if err := h.clientRepo.Create(r.Context(), client); err != nil {
		httputil.Error(w, http.StatusInternalServerError, err.Error(), "")
		return
	}

	httputil.JSON(w, http.StatusCreated, map[string]any{
		"id":                         client.ID,
		"client_id":                  client.ClientID,
		"client_secret":              clientSecret,
		"name":                       client.Name,
		"redirect_uris":              client.RedirectURIs,
		"grant_types":                client.GrantTypes,
		"token_endpoint_auth_method": client.TokenEndpointAuthMethod,
	})
}

func (h *AdminHandler) HandleDeleteClient(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid id", "")
		return
	}

	if err := h.clientRepo.Delete(r.Context(), id); err != nil {
		httputil.Error(w, http.StatusInternalServerError, err.Error(), "")
		return
	}

	httputil.JSON(w, http.StatusOK, map[string]any{})
}

type ClientRegistrationHandler struct {
	clientRepo    repository.ClientRepository
	cryptoService *auth.CryptoService
}

func NewClientRegistrationHandler(clientRepo repository.ClientRepository, cryptoService *auth.CryptoService) *ClientRegistrationHandler {
	return &ClientRegistrationHandler{
		clientRepo:    clientRepo,
		cryptoService: cryptoService,
	}
}

func (h *ClientRegistrationHandler) HandleRegistration(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RedirectURIs            []string `json:"redirect_uris"`
		GrantTypes              []string `json:"grant_types"`
		TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
		ApplicationType         string   `json:"application_type"`
	}

	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid_request", "")
		return
	}

	clientSecret := h.cryptoService.GenerateRandomString(32)
	clientSecretHash, _ := h.cryptoService.HashPassword(clientSecret)

	client := &entity.Client{
		ID:                      uuid.Must(uuid.NewV7()),
		ClientID:                h.cryptoService.GenerateRandomString(16),
		ClientSecretHash:        clientSecretHash,
		Name:                    "Dynamic Client",
		RedirectURIs:            req.RedirectURIs,
		GrantTypes:              req.GrantTypes,
		TokenEndpointAuthMethod: req.TokenEndpointAuthMethod,
		CreatedAt:               time.Now(),
		UpdatedAt:               time.Now(),
	}

	if err := h.clientRepo.Create(r.Context(), client); err != nil {
		httputil.Error(w, http.StatusInternalServerError, err.Error(), "")
		return
	}

	httputil.JSON(w, http.StatusCreated, map[string]any{
		"client_id":                  client.ClientID,
		"client_secret":              clientSecret,
		"client_id_issued_at":        client.CreatedAt.Unix(),
		"redirect_uris":              client.RedirectURIs,
		"grant_types":                client.GrantTypes,
		"token_endpoint_auth_method": client.TokenEndpointAuthMethod,
	})
}
