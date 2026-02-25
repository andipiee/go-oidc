package main

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	JWT      JWTConfig      `yaml:"jwt"`
	OAuth    OAuthConfig    `yaml:"oauth"`
	Admin    AdminConfig    `yaml:"admin"`
}

type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Name     string `yaml:"name"`
	SSLMode  string `yaml:"sslmode"`
}

type JWTConfig struct {
	Issuer               string `yaml:"issuer"`
	AccessTokenTTL       string `yaml:"access_token_ttl"`
	RefreshTokenTTL      string `yaml:"refresh_token_ttl"`
	RefreshTokenRotation bool   `yaml:"refresh_token_rotation"`
	CodeTTL              string `yaml:"code_ttl"`
	AccessTokenTTLDur    int64  `yaml:"-"`
	RefreshTokenTTLDur   int64  `yaml:"-"`
	CodeTTLDur           int64  `yaml:"-"`
}

type ProviderConfig struct {
	Name         string   `yaml:"name"`
	ClientID     string   `yaml:"client_id"`
	ClientSecret string   `yaml:"client_secret"`
	Scopes       []string `yaml:"scopes"`
	AuthURL      string   `yaml:"auth_url"`
	TokenURL     string   `yaml:"token_url"`
	UserInfoURL  string   `yaml:"user_info_url"`
}

type OAuthConfig struct {
	Providers []ProviderConfig `yaml:"providers"`
}

type AdminConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

func (c *Config) GetDSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.Database.User,
		c.Database.Password,
		c.Database.Host,
		c.Database.Port,
		c.Database.Name,
		c.Database.SSLMode,
	)
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	parseDuration := func(s string) int64 {
		d, _ := parseDurationString(s)
		return d
	}

	cfg.JWT.AccessTokenTTLDur = parseDuration(cfg.JWT.AccessTokenTTL)
	cfg.JWT.RefreshTokenTTLDur = parseDuration(cfg.JWT.RefreshTokenTTL)
	cfg.JWT.CodeTTLDur = parseDuration(cfg.JWT.CodeTTL)

	return &cfg, nil
}

func parseDurationString(s string) (int64, error) {
	var multiplier int64 = 1
	var value int64 = 0

	if len(s) < 2 {
		return 0, fmt.Errorf("invalid duration format")
	}

	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= '0' && c <= '9' {
			value = value*10 + int64(c-'0')
		} else {
			switch s[i:] {
			case "s":
				multiplier = 1
			case "m":
				multiplier = 60
			case "h":
				multiplier = 3600
			case "d":
				multiplier = 86400
			default:
				return 0, fmt.Errorf("unknown unit: %s", s[i:])
			}
			break
		}
	}
	return value * multiplier, nil
}
