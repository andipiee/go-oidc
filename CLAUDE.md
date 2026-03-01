# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Build
go build -o server ./cmd/server

# Run (must run from project root — config and web/static paths are relative)
./server

# Hot reload
air

# Test
go test ./...

# Single package test
go test ./internal/application/usecase/...

# Format
go fmt ./...

# Lint
golangci-lint run

# Migrations (powered by goose)
go run ./cmd/migrate up       # apply all pending migrations
go run ./cmd/migrate down     # rollback the last migration
go run ./cmd/migrate reset    # rollback all migrations
go run ./cmd/migrate status   # show migration status

# Docker
docker-compose up -d
docker-compose logs -f app
```

## Database Setup

PostgreSQL is required. Migrations use [goose](https://github.com/pressly/goose) via `cmd/migrate`:

```bash
go run ./cmd/migrate up
```

You can override the database name with the `DATABASE_NAME` env var:

```bash
DATABASE_NAME=oauth2_test go run ./cmd/migrate up
```

The `docker-compose.yml` includes an Adminer UI on port 8081 but no managed Postgres container — the app expects an external Postgres instance matching `configs/config.yaml`.

## Architecture

Clean Architecture with strict dependency direction: `presentation → application → domain ← infrastructure`.

```
cmd/server/                  # Wire-up, config loading (main.go)
cmd/migrate/                 # Migration CLI (up/down/reset/status via goose)
internal/
  domain/
    entity/entity.go         # All domain types: User, Client, tokens, sessions, DeviceCode
    repository/repository.go # Repository interfaces (UserRepository, ClientRepository, TokenRepository, SessionRepository)
  application/usecase/       # Business logic; depends only on domain interfaces + auth infra
    authorize.go             # Authorization Code flow
    token.go                 # Token exchange, refresh, revocation, introspection
    user.go                  # User info + session management
  infrastructure/
    database/postgres.go     # pgx/v5 connection pool
    repository/              # Concrete postgres implementations of domain interfaces
    auth/jwt.go              # JWTService (RS256 sign/verify) + CryptoService (bcrypt, PKCE, random)
    oauth/providers.go       # External IdP (Google, GitHub) OAuth2 flow
  presentation/
    handler/                 # Gin handlers: authorize, token, userinfo, device, admin, auth, discovery
    middleware/middleware.go  # Logger, CORS, BasicAuth
web/static/                  # HTML pages served by Gin (index, login, register)
configs/config.yaml          # Server, DB, JWT, OAuth providers, Admin config
internal/config/config.go    # Shared config types and loader
migrations/                  # SQL files with goose Up/Down annotations
```

## Key Behaviours to Know

- **RSA key is ephemeral**: `auth.NewJWTService` generates a fresh RSA-2048 key pair on every startup. All previously issued tokens become invalid after a server restart.

- **AuthorizeUseCase hardcodes demo user**: `authorize.go:87` fetches `demo@example.com` unconditionally as the authorized user for the Authorization Code flow. This is a placeholder — real user authentication flows through `presentation/handler/auth.go` and the `/auth/login` routes.

- **Token hashing**: Access and refresh tokens are stored as SHA-256 hashes (`JWTService.HashToken`). Lookups always use the hash, not the raw token.

- **Config path**: The server loads `configs/config.yaml` as a relative path, so it must be started from the project root.

- **Default credentials** (seeded in `migrations/001_init.sql`):
  - Client: `default-client` / `default`
  - User: `demo@example.com` / `default`
  - Admin UI (`/admin/`): `admin` / `changeme`

## Module

`github.com/andipiee/go-oidc` (Go 1.24.3)

Key dependencies: `gin`, `golang-jwt/jwt/v5`, `pgx/v5`, `golang.org/x/crypto`, `golang.org/x/oauth2`
