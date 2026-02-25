package main

import (
	"context"
	"fmt"
	"log"
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
	"github.com/andipiee/go-oidc/internal/presentation/handler"
	"github.com/andipiee/go-oidc/internal/presentation/middleware"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg, err := LoadConfig("configs/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	db, err := database.NewPostgresDB(cfg.GetDSN())
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	userRepo := repository.NewUserRepository(db)
	clientRepo := repository.NewClientRepository(db)
	tokenRepo := repository.NewTokenRepository(db)
	sessionRepo := repository.NewSessionRepository(db)

	jwtService := auth.NewJWTService(cfg.JWT.Issuer, cfg.JWT.AccessTokenTTLDur, cfg.JWT.RefreshTokenTTLDur)
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

	authorizeUseCase := usecase.NewAuthorizeUseCase(clientRepo, userRepo, tokenRepo, jwtService, cryptoService, cfg.JWT.CodeTTLDur)

	tokenUseCaseCfg := usecase.JWTConfig{
		Issuer:               cfg.JWT.Issuer,
		AccessTokenTTLDur:    cfg.JWT.AccessTokenTTLDur,
		RefreshTokenTTLDur:   cfg.JWT.RefreshTokenTTLDur,
		RefreshTokenRotation: cfg.JWT.RefreshTokenRotation,
		CodeTTLDur:           cfg.JWT.CodeTTLDur,
	}
	tokenUseCase := usecase.NewTokenUseCase(clientRepo, userRepo, tokenRepo, jwtService, cryptoService, tokenUseCaseCfg)
	userUseCase := usecase.NewUserUseCase(userRepo, sessionRepo)
	deviceUseCase := usecase.NewDeviceUseCase(clientRepo, userRepo, tokenRepo, jwtService, cfg.JWT.CodeTTLDur)

	authorizeHandler := handler.NewAuthorizeHandler(authorizeUseCase)
	tokenHandler := handler.NewTokenHandler(tokenUseCase)
	userinfoHandler := handler.NewUserInfoHandler(userUseCase, jwtService)
	deviceHandler := handler.NewDeviceHandler(deviceUseCase)
	adminHandler := handler.NewAdminHandler(userRepo, clientRepo, handler.AdminConfig(cfg.Admin))
	discoveryHandler := handler.NewDiscoveryHandler(cfg.JWT.Issuer, cfg.Server.Port)

	router := gin.Default()
	router.Use(middleware.Logger())
	router.Use(middleware.CORS())

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	router.GET("/.well-known/openid-configuration", discoveryHandler.HandleDiscovery)
	router.GET("/.well-known/jwks.json", discoveryHandler.HandleJWKS)

	oauth2Group := router.Group("/oauth2")
	{
		oauth2Group.GET("/authorize", authorizeHandler.HandleAuthorize)
		oauth2Group.POST("/authorize", authorizeHandler.HandleAuthorizePost)
		oauth2Group.POST("/token", tokenHandler.HandleToken)
		oauth2Group.POST("/revoke", tokenHandler.HandleRevoke)
		oauth2Group.POST("/introspect", tokenHandler.HandleIntrospect)
	}

	oidcGroup := router.Group("/oidc")
	{
		oidcGroup.GET("/userinfo", userinfoHandler.HandleUserInfo)
		oidcGroup.POST("/device/authorize", deviceHandler.HandleDeviceAuthorize)
		oidcGroup.GET("/device", deviceHandler.HandleDevice)
		oidcGroup.POST("/register", handler.NewClientRegistrationHandler(clientRepo, cryptoService).HandleRegistration)
	}

	if cfg.Admin.Enabled {
		adminGroup := router.Group("/admin")
		adminGroup.Use(middleware.BasicAuth(cfg.Admin.Username, cfg.Admin.Password))
		{
			adminGroup.GET("/", adminHandler.HandleIndex)
			adminGroup.GET("/users", adminHandler.HandleListUsers)
			adminGroup.POST("/users", adminHandler.HandleCreateUser)
			adminGroup.DELETE("/users/:id", adminHandler.HandleDeleteUser)
			adminGroup.GET("/clients", adminHandler.HandleListClients)
			adminGroup.POST("/clients", adminHandler.HandleCreateClient)
			adminGroup.DELETE("/clients/:id", adminHandler.HandleDeleteClient)
		}
	}

	staticHandler := http.FileServer(http.Dir("web/static"))
	router.GET("/static/*any", gin.WrapH(staticHandler))
	router.GET("/", func(c *gin.Context) {
		c.File("web/static/index.html")
	})

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	go func() {
		log.Printf("Server starting on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
	log.Println("Server exited")
}
