package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/andipiee/go-oidc/internal/domain/entity"
	"github.com/andipiee/go-oidc/internal/domain/repository"
	"github.com/andipiee/go-oidc/internal/infrastructure/auth"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type DiscoveryHandler struct {
	issuer string
	port   int
}

func NewDiscoveryHandler(issuer string, port int) *DiscoveryHandler {
	return &DiscoveryHandler{issuer: issuer, port: port}
}

func (h *DiscoveryHandler) HandleDiscovery(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
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
		"code_challenge_methods_supported":      []string{"S256", "plain"},
	})
}

func (h *DiscoveryHandler) HandleJWKS(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"keys": []gin.H{
			{
				"kty": "RSA",
				"use": "sig",
				"kid": "1",
				"alg": "RS256",
			},
		},
	})
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

func (h *AdminHandler) HandleIndex(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Admin API", "endpoints": []string{"/admin/users", "/admin/clients"}})
}

func (h *AdminHandler) HandleListUsers(c *gin.Context) {
	users, err := h.userRepo.List(c.Request.Context(), 100, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, users)
}

func (h *AdminHandler) HandleCreateUser(c *gin.Context) {
	var req struct {
		Email    string `json:"email"`
		Name     string `json:"name"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	crypto := auth.NewCryptoService()
	hash, _ := crypto.HashPassword(req.Password)

	user := &entity.User{
		ID:            uuid.New(),
		Email:         req.Email,
		PasswordHash:  hash,
		Name:          req.Name,
		EmailVerified: true,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := h.userRepo.Create(c.Request.Context(), user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, user)
}

func (h *AdminHandler) HandleDeleteUser(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.userRepo.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{})
}

func (h *AdminHandler) HandleListClients(c *gin.Context) {
	clients, err := h.clientRepo.List(c.Request.Context(), 100, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, clients)
}

func (h *AdminHandler) HandleCreateClient(c *gin.Context) {
	var req struct {
		Name                    string   `json:"name"`
		RedirectURIs            []string `json:"redirect_uris"`
		GrantTypes              []string `json:"grant_types"`
		TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	crypto := auth.NewCryptoService()
	clientSecret := crypto.GenerateRandomString(32)
	clientSecretHash, _ := crypto.HashPassword(clientSecret)

	client := &entity.Client{
		ID:                      uuid.New(),
		ClientID:                crypto.GenerateRandomString(16),
		ClientSecretHash:        clientSecretHash,
		Name:                    req.Name,
		RedirectURIs:            req.RedirectURIs,
		GrantTypes:              req.GrantTypes,
		TokenEndpointAuthMethod: req.TokenEndpointAuthMethod,
		CreatedAt:               time.Now(),
		UpdatedAt:               time.Now(),
	}

	if err := h.clientRepo.Create(c.Request.Context(), client); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	resp := map[string]interface{}{
		"id":                         client.ID,
		"client_id":                  client.ClientID,
		"client_secret":              clientSecret,
		"name":                       client.Name,
		"redirect_uris":              client.RedirectURIs,
		"grant_types":                client.GrantTypes,
		"token_endpoint_auth_method": client.TokenEndpointAuthMethod,
	}

	c.JSON(http.StatusCreated, resp)
}

func (h *AdminHandler) HandleDeleteClient(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.clientRepo.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{})
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

func (h *ClientRegistrationHandler) HandleRegistration(c *gin.Context) {
	var req struct {
		RedirectURIs            []string `json:"redirect_uris"`
		GrantTypes              []string `json:"grant_types"`
		TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
		ApplicationType         string   `json:"application_type"`
	}

	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}

	clientSecret := h.cryptoService.GenerateRandomString(32)
	clientSecretHash, _ := h.cryptoService.HashPassword(clientSecret)

	client := &entity.Client{
		ID:                      uuid.New(),
		ClientID:                h.cryptoService.GenerateRandomString(16),
		ClientSecretHash:        clientSecretHash,
		Name:                    "Dynamic Client",
		RedirectURIs:            req.RedirectURIs,
		GrantTypes:              req.GrantTypes,
		TokenEndpointAuthMethod: req.TokenEndpointAuthMethod,
		CreatedAt:               time.Now(),
		UpdatedAt:               time.Now(),
	}

	if err := h.clientRepo.Create(c.Request.Context(), client); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"client_id":                  client.ClientID,
		"client_secret":              clientSecret,
		"client_id_issued_at":        client.CreatedAt.Unix(),
		"redirect_uris":              client.RedirectURIs,
		"grant_types":                client.GrantTypes,
		"token_endpoint_auth_method": client.TokenEndpointAuthMethod,
	})
}
