package main

import (
	"os"

	"github.com/andipiee/go-oidc/internal/config"
)

// Type aliases so the rest of cmd/server can use unqualified names.
type Config = config.Config
type ServerConfig = config.ServerConfig
type DatabaseConfig = config.DatabaseConfig
type JWTConfig = config.JWTConfig
type ProviderConfig = config.ProviderConfig
type OAuthConfig = config.OAuthConfig
type AdminConfig = config.AdminConfig

func configPath() string {
	if p := os.Getenv("CONFIG_PATH"); p != "" {
		return p
	}
	return "configs/config.yaml"
}

var LoadConfig = config.LoadConfig
