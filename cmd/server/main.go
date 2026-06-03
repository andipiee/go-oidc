package main

import (
	"context"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/andipiee/go-oidc/internal/application/usecase"
	"github.com/andipiee/go-oidc/internal/domain/entity"
	"github.com/andipiee/go-oidc/internal/infrastructure/auth"
	"github.com/andipiee/go-oidc/internal/infrastructure/database"
	"github.com/andipiee/go-oidc/internal/infrastructure/oauth"
	"github.com/andipiee/go-oidc/internal/infrastructure/repository"
	"github.com/andipiee/go-oidc/internal/infrastructure/telemetry"
	"github.com/andipiee/go-oidc/internal/presentation/handler"
	"github.com/andipiee/go-oidc/internal/presentation/httputil"
	"github.com/andipiee/go-oidc/internal/presentation/middleware"
	"github.com/google/uuid"
)

// adminConsoleClientID is the fixed client_id for the OIDC-authenticated admin
// console. Bootstrapped at startup by ensureAdminConsoleClient.
const adminConsoleClientID = "admin-console"

func main() {
	telemetry.InitLogger()

	ctx := context.Background()
	otelShutdown, err := telemetry.Init(ctx)
	if err != nil {
		slog.Error("failed to initialize OpenTelemetry", "error", err)
		os.Exit(1)
	}

	cfg, err := LoadConfig(configPath())
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	db, err := database.NewPostgresDB(cfg.GetDSN())
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	tmpl := template.Must(template.ParseGlob("web/static/*.html"))

	userRepo := repository.NewUserRepository(db)
	clientRepo := repository.NewClientRepository(db)
	tokenRepo := repository.NewTokenRepository(db)
	sessionRepo := repository.NewSessionRepository(db)

	jwtService := auth.NewJWTService(cfg.JWT.Issuer, cfg.JWT.AccessTokenTTLDur, cfg.JWT.RefreshTokenTTLDur, cfg.JWT.PrivateKey)
	cryptoService := auth.NewCryptoService()

	providerConfigs := make([]oauth.ProviderConfig, len(cfg.OAuth.Providers))
	for i, p := range cfg.OAuth.Providers {
		providerConfigs[i] = oauth.ProviderConfig{
			Name:         p.Name,
			ClientID:     p.ClientID,
			ClientSecret: p.ClientSecret,
			Scopes:       p.Scopes,
			AuthURL:      p.AuthURL,
			TokenURL:     p.TokenURL,
			UserInfoURL:  p.UserInfoURL,
		}
	}
	providerService := oauth.NewProviderService(providerConfigs)
	_ = providerService

	userUseCase := usecase.NewUserUseCase(userRepo, sessionRepo)

	authorizeUseCase := usecase.NewAuthorizeUseCase(clientRepo, userRepo, tokenRepo, jwtService, cryptoService, cfg.JWT.CodeTTLDur)

	tokenUseCaseCfg := usecase.JWTConfig{
		Issuer:               cfg.JWT.Issuer,
		AccessTokenTTLDur:    cfg.JWT.AccessTokenTTLDur,
		RefreshTokenTTLDur:   cfg.JWT.RefreshTokenTTLDur,
		RefreshTokenRotation: cfg.JWT.RefreshTokenRotation,
		CodeTTLDur:           cfg.JWT.CodeTTLDur,
	}
	tokenUseCase := usecase.NewTokenUseCase(clientRepo, userRepo, tokenRepo, jwtService, cryptoService, tokenUseCaseCfg)
	deviceUseCase := usecase.NewDeviceUseCase(clientRepo, userRepo, tokenRepo, jwtService, cfg.JWT.CodeTTLDur)

	authorizeHandler := handler.NewAuthorizeHandler(authorizeUseCase, userUseCase)
	tokenHandler := handler.NewTokenHandler(tokenUseCase)
	userinfoHandler := handler.NewUserInfoHandler(userUseCase, jwtService)
	deviceHandler := handler.NewDeviceHandler(deviceUseCase, tmpl)
	adminHandler := handler.NewAdminHandler(userRepo)
	discoveryHandler := handler.NewDiscoveryHandler(cfg.JWT.Issuer, cfg.Server.Port, jwtService)
	authHandler := handler.NewAuthHandler(userRepo, cryptoService, jwtService, userUseCase, tmpl)
	registrationHandler := handler.NewClientRegistrationHandler(clientRepo, cryptoService)

	// The admin console authenticates via the normal OIDC flow as a public
	// (PKCE) client. Bootstrap that client idempotently so its redirect_uri
	// always tracks the configured issuer across environments.
	adminRedirectURI := cfg.JWT.Issuer + "/admin/callback"
	if cfg.Admin.Enabled {
		if err := ensureAdminConsoleClient(ctx, clientRepo, adminConsoleClientID, adminRedirectURI); err != nil {
			slog.Error("failed to bootstrap admin console client", "error", err)
			os.Exit(1)
		}
	}
	adminConsoleHandler := handler.NewAdminConsoleHandler(jwtService, tokenUseCase, cryptoService, clientRepo, adminConsoleClientID, adminRedirectURI, tmpl)

	// RequireAdmin validates the admin_session id_token and gates on role=admin.
	validateAdminToken := func(token string) (string, error) {
		claims, err := jwtService.ValidateToken(token)
		if err != nil {
			return "", err
		}
		return claims.Role, nil
	}

	mux := http.NewServeMux()

	// Health
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		httputil.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// Discovery
	mux.HandleFunc("GET /.well-known/openid-configuration", discoveryHandler.HandleDiscovery)
	mux.HandleFunc("GET /.well-known/jwks.json", discoveryHandler.HandleJWKS)

	// OAuth2
	mux.HandleFunc("GET /oauth2/authorize", authorizeHandler.HandleAuthorize)
	mux.HandleFunc("POST /oauth2/authorize", authorizeHandler.HandleAuthorizePost)
	mux.HandleFunc("POST /oauth2/token", tokenHandler.HandleToken)
	mux.HandleFunc("POST /oauth2/revoke", tokenHandler.HandleRevoke)
	mux.HandleFunc("POST /oauth2/introspect", tokenHandler.HandleIntrospect)

	// OIDC
	mux.HandleFunc("GET /oidc/userinfo", userinfoHandler.HandleUserInfo)
	mux.HandleFunc("POST /oidc/device/authorize", deviceHandler.HandleDeviceAuthorize)
	mux.HandleFunc("GET /oidc/device", deviceHandler.HandleDevice)
	mux.HandleFunc("POST /oidc/register", registrationHandler.HandleRegistration)

	// Admin console — gated by OIDC (role=admin claim), not Basic Auth.
	if cfg.Admin.Enabled {
		requireAdminPage := middleware.RequireAdmin(validateAdminToken, true) // redirect to /admin/login
		requireAdminAPI := middleware.RequireAdmin(validateAdminToken, false) // 401/403 JSON

		// OIDC relying-party endpoints (public — they ARE the login path).
		mux.HandleFunc("GET /admin/login", adminConsoleHandler.HandleLogin)
		mux.HandleFunc("GET /admin/callback", adminConsoleHandler.HandleCallback)
		mux.HandleFunc("POST /admin/logout", adminConsoleHandler.HandleLogout)

		// Dashboard page + client management — server-rendered, form POSTs.
		mux.Handle("GET /admin", requireAdminPage(http.HandlerFunc(adminConsoleHandler.ShowDashboard)))
		mux.Handle("POST /admin/clients", requireAdminPage(http.HandlerFunc(adminConsoleHandler.HandleCreateClient)))
		mux.Handle("POST /admin/clients/{id}/rotate-secret", requireAdminPage(http.HandlerFunc(adminConsoleHandler.HandleRotateSecret)))
		mux.Handle("POST /admin/clients/{id}/delete", requireAdminPage(http.HandlerFunc(adminConsoleHandler.HandleDeleteClient)))

		// User management — JSON API (no dashboard UI yet), gated the same way.
		mux.Handle("GET /admin/users", requireAdminAPI(http.HandlerFunc(adminHandler.HandleListUsers)))
		mux.Handle("POST /admin/users", requireAdminAPI(http.HandlerFunc(adminHandler.HandleCreateUser)))
		mux.Handle("DELETE /admin/users/{id}", requireAdminAPI(http.HandlerFunc(adminHandler.HandleDeleteUser)))
	}

	// Auth
	mux.HandleFunc("GET /auth/login", authHandler.ShowLoginPage)
	mux.HandleFunc("POST /auth/login", authHandler.HandleLoginForm)
	mux.HandleFunc("POST /auth/login/json", authHandler.HandleLogin)
	mux.HandleFunc("GET /auth/register", authHandler.ShowRegisterPage)
	mux.HandleFunc("POST /auth/register", authHandler.HandleRegisterForm)
	mux.HandleFunc("POST /auth/register/json", authHandler.HandleRegister)
	mux.HandleFunc("POST /auth/logout", authHandler.HandleLogout)

	// Static files
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))

	// Root
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "web/static/index.html")
	})

	// Global middleware chain (outermost runs first)
	var h http.Handler = mux
	h = middleware.CORS(h)
	h = middleware.Recovery(h)
	h = middleware.Logger(h)

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: h,
	}

	go func() {
		slog.Info("server starting", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("failed to start server", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
	}
	if err := otelShutdown(shutdownCtx); err != nil {
		slog.Error("failed to shutdown OpenTelemetry", "error", err)
	}
	slog.Info("server exited")
}

// ensureAdminConsoleClient upserts the public PKCE client the admin console
// uses to authenticate. Idempotent: created on first boot; on later boots its
// redirect_uri is realigned with the configured issuer (so prod/staging each
// get the right callback without an env-specific migration).
func ensureAdminConsoleClient(ctx context.Context, clientRepo *repository.ClientRepository, clientID, redirectURI string) error {
	existing, err := clientRepo.GetByClientID(ctx, clientID)
	if err != nil {
		return err
	}
	if existing != nil {
		existing.RedirectURIs = []string{redirectURI}
		existing.TokenEndpointAuthMethod = "none"
		existing.UpdatedAt = time.Now()
		return clientRepo.Update(ctx, existing)
	}
	return clientRepo.Create(ctx, &entity.Client{
		ID:                      uuid.Must(uuid.NewV7()),
		ClientID:                clientID,
		ClientSecretHash:        "", // public client — PKCE only, no secret
		Name:                    "Admin Console",
		RedirectURIs:            []string{redirectURI},
		GrantTypes:              []string{"authorization_code", "refresh_token"},
		ResponseTypes:           []string{"code"},
		TokenEndpointAuthMethod: "none",
		Scopes:                  []string{"openid", "profile", "email"},
		CreatedAt:               time.Now(),
		UpdatedAt:               time.Now(),
	})
}
