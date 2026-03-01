package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"

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
	var cfg Config

	data, err := os.ReadFile(path)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// YAML file is optional — env vars can fill in all values.
	} else {
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("failed to parse config: %w", err)
		}
	}

	applyEnvOverrides(&cfg)

	parseDuration := func(s string) int64 {
		d, _ := parseDurationString(s)
		return d
	}

	cfg.JWT.AccessTokenTTLDur = parseDuration(cfg.JWT.AccessTokenTTL)
	cfg.JWT.RefreshTokenTTLDur = parseDuration(cfg.JWT.RefreshTokenTTL)
	cfg.JWT.CodeTTLDur = parseDuration(cfg.JWT.CodeTTL)

	return &cfg, nil
}

func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv("SERVER_HOST"); v != "" {
		cfg.Server.Host = v
	}
	if v := os.Getenv("SERVER_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.Server.Port = port
		}
	}
	if v := os.Getenv("DATABASE_HOST"); v != "" {
		cfg.Database.Host = v
	}
	if v := os.Getenv("DATABASE_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.Database.Port = port
		}
	}
	if v := os.Getenv("DATABASE_USER"); v != "" {
		cfg.Database.User = v
	}
	if v := os.Getenv("DATABASE_PASSWORD"); v != "" {
		cfg.Database.Password = v
	}
	if v := os.Getenv("DATABASE_NAME"); v != "" {
		cfg.Database.Name = v
	}
	if v := os.Getenv("DATABASE_SSLMODE"); v != "" {
		cfg.Database.SSLMode = v
	}
	if v := os.Getenv("JWT_ISSUER"); v != "" {
		cfg.JWT.Issuer = v
	}
	if v := os.Getenv("JWT_ACCESS_TOKEN_TTL"); v != "" {
		cfg.JWT.AccessTokenTTL = v
	}
	if v := os.Getenv("JWT_REFRESH_TOKEN_TTL"); v != "" {
		cfg.JWT.RefreshTokenTTL = v
	}
	if v := os.Getenv("JWT_REFRESH_TOKEN_ROTATION"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			cfg.JWT.RefreshTokenRotation = b
		}
	}
	if v := os.Getenv("JWT_CODE_TTL"); v != "" {
		cfg.JWT.CodeTTL = v
	}
	if v := os.Getenv("ADMIN_ENABLED"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			cfg.Admin.Enabled = b
		}
	}
	if v := os.Getenv("ADMIN_USERNAME"); v != "" {
		cfg.Admin.Username = v
	}
	if v := os.Getenv("ADMIN_PASSWORD"); v != "" {
		cfg.Admin.Password = v
	}
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
