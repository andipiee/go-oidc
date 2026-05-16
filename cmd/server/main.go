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
	"github.com/andipiee/go-oidc/internal/infrastructure/auth"
	"github.com/andipiee/go-oidc/internal/infrastructure/database"
	"github.com/andipiee/go-oidc/internal/infrastructure/oauth"
	"github.com/andipiee/go-oidc/internal/infrastructure/repository"
	"github.com/andipiee/go-oidc/internal/infrastructure/telemetry"
	"github.com/andipiee/go-oidc/internal/presentation/handler"
	"github.com/andipiee/go-oidc/internal/presentation/httputil"
	"github.com/andipiee/go-oidc/internal/presentation/middleware"
)

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
	adminHandler := handler.NewAdminHandler(userRepo, clientRepo, handler.AdminConfig(cfg.Admin))
	discoveryHandler := handler.NewDiscoveryHandler(cfg.JWT.Issuer, cfg.Server.Port, jwtService)
	authHandler := handler.NewAuthHandler(userRepo, cryptoService, jwtService, userUseCase, tmpl)
	registrationHandler := handler.NewClientRegistrationHandler(clientRepo, cryptoService)

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

	// Admin (with BasicAuth)
	if cfg.Admin.Enabled {
		adminAuth := middleware.BasicAuth(cfg.Admin.Username, cfg.Admin.Password)
		mux.Handle("GET /admin/", adminAuth(http.HandlerFunc(adminHandler.HandleIndex)))
		mux.Handle("GET /admin/users", adminAuth(http.HandlerFunc(adminHandler.HandleListUsers)))
		mux.Handle("POST /admin/users", adminAuth(http.HandlerFunc(adminHandler.HandleCreateUser)))
		mux.Handle("DELETE /admin/users/{id}", adminAuth(http.HandlerFunc(adminHandler.HandleDeleteUser)))
		mux.Handle("GET /admin/clients", adminAuth(http.HandlerFunc(adminHandler.HandleListClients)))
		mux.Handle("POST /admin/clients", adminAuth(http.HandlerFunc(adminHandler.HandleCreateClient)))
		mux.Handle("DELETE /admin/clients/{id}", adminAuth(http.HandlerFunc(adminHandler.HandleDeleteClient)))
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
