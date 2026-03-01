# OAuth2 / OpenID Connect Server

A standards-compliant OAuth 2.1 and OpenID Connect server implementation in Go, built with Clean Architecture principles and Go's native `net/http` standard library (Go 1.22+ ServeMux with method routing).

## Features

### OAuth 2.1 Compliance

- **Authorization Code Flow with mandatory PKCE** (S256 only, per OAuth 2.1)
- **Refresh Token Flow** with mandatory token rotation
- **Device Authorization Flow** (RFC 8628)
- **Client authentication** via `client_secret_basic`, `client_secret_post`, or `none`
- **Token revocation** (RFC 7009) and **introspection** (RFC 7662)

### OpenID Connect

- **OIDC Discovery** (`/.well-known/openid-configuration`) with full metadata
- **JWKS endpoint** (`/.well-known/jwks.json`) serving the actual RSA public key
- **Proper ID tokens** — separate JWT from the access token, with `sub`, `aud`, `iss`, `exp`, `iat`, `nonce` claims
- **UserInfo endpoint** with scope-based claim filtering (`profile`, `email`)
- **Dynamic Client Registration** (RFC 7591)

### Security

- **RS256 JWT signing** with RSA-2048 key pair (ephemeral, regenerated on restart)
- **Password hashing** via bcrypt
- **Token hashing** — access and refresh tokens stored as SHA-256 hashes
- **Session-based authentication** for the authorization flow (no hardcoded demo user)
- **PKCE S256 enforced** — `plain` method is rejected

### Architecture

- **Zero framework dependencies** — uses Go's `net/http` standard library with Go 1.22+ routing
- **Clean Architecture** with strict dependency direction: `presentation -> application -> domain <- infrastructure`
- **External Identity Provider support** (Google, GitHub) — configurable via YAML

### Admin API

- User and client management (CRUD)
- Protected by HTTP Basic Auth

## Quick Start

### Prerequisites

- Go 1.24+
- PostgreSQL 15+
- Docker & Docker Compose (optional)

### Using Docker Compose

```bash
# Clone the repository
git clone https://github.com/andipiee/go-oidc.git
cd go-oidc

# Start the server
docker-compose up -d

# Access the server
open http://localhost:8080

# View logs
docker-compose logs -f app
```

### Manual Setup

```bash
# Install dependencies
go mod download

# Copy the example config and edit with your credentials
cp configs/config.yaml.example configs/config.yaml
# Edit configs/config.yaml with your database password and other settings

# Create the database
createdb -U postgres oauth2

# Run migrations
go run ./cmd/migrate up

# Run the server (must run from project root)
go run ./cmd/server
```

> **Note:** The server must be started from the project root directory because `configs/config.yaml` and `web/static/` are loaded via relative paths.

## Configuration

Configuration is managed via a single file: `configs/config.yaml`. There is no `.env` support — all settings live in the YAML file. See [docs/CONFIG.md](docs/CONFIG.md) for the full reference.

```yaml
server:
  host: "0.0.0.0"
  port: 8080

database:
  host: "localhost"          # Use "localhost" for local dev, "postgres" for Docker
  port: 5432
  user: "postgres"
  password: "changeme"       # Change in production
  name: "oauth2"
  sslmode: "disable"         # Use "require" in production

jwt:
  issuer: "http://localhost:8080"
  access_token_ttl: "15m"
  refresh_token_ttl: "24h"
  refresh_token_rotation: true
  code_ttl: "5m"

admin:
  enabled: true
  username: "admin"
  password: "changeme"       # Change in production
```

> **Note:** The checked-in config has `database.host: "postgres"` (the Docker Compose service name). For local development without Docker, change this to `"localhost"`.

## Default Credentials

These are seeded by `migrations/001_init.sql` for development only. **Change them before deploying to production.**

| Resource | Identifier | Secret |
|---|---|---|
| Admin UI (`/admin/`) | `admin` | `changeme` |
| Demo user | `demo@example.com` | `default` |
| Default client | `default-client` | `default` |

## Project Structure

```
cmd/server/                      # Application entry point
  config.go                      # YAML config loading
  main.go                        # Wiring, route registration, server startup
cmd/migrate/                     # Migration CLI (up/down/reset/status via goose)
internal/
  config/config.go               # Shared config types and loader
  domain/
    entity/entity.go             # Domain types: User, Client, tokens, sessions, DeviceCode
    repository/repository.go     # Repository interfaces
  application/usecase/           # Business logic (depends only on domain interfaces)
    authorize.go                 # Authorization Code flow
    token.go                     # Token exchange, refresh, revocation, introspection
    user.go                      # User info, session management, device flow
  infrastructure/
    database/postgres.go         # pgx/v5 connection pool
    repository/                  # PostgreSQL implementations of domain interfaces
    auth/jwt.go                  # JWTService (RS256), CryptoService (bcrypt, PKCE)
    oauth/providers.go           # External IdP (Google, GitHub) OAuth2 configuration
  presentation/
    httputil/response.go         # HTTP response helpers (JSON, Error, HTML, DecodeJSON)
    handler/                     # HTTP handlers (net/http)
      authorize.go               # /oauth2/authorize
      token.go                   # /oauth2/token, /oauth2/revoke, /oauth2/introspect
      userinfo.go                # /oidc/userinfo
      device.go                  # /oidc/device, /oidc/device/authorize
      admin.go                   # /admin/*, discovery, JWKS, client registration
      auth.go                    # /auth/login, /auth/register, /auth/logout
    middleware/middleware.go      # Logger, CORS, BasicAuth (standard net/http middleware)
web/static/                      # HTML templates (login, register, index)
configs/config.yaml              # Application configuration
migrations/                      # SQL migration files (managed by goose)
docker/Dockerfile                # Multi-stage Docker build
```

## API Endpoints

### Discovery & Keys

| Endpoint | Method | Description |
|---|---|---|
| `/.well-known/openid-configuration` | GET | OIDC Discovery metadata |
| `/.well-known/jwks.json` | GET | JSON Web Key Set (actual RSA public key) |

### OAuth2

| Endpoint | Method | Description |
|---|---|---|
| `/oauth2/authorize` | GET, POST | Authorization endpoint (requires PKCE) |
| `/oauth2/token` | POST | Token exchange (auth code or refresh token) |
| `/oauth2/revoke` | POST | Token revocation |
| `/oauth2/introspect` | POST | Token introspection |

### OpenID Connect

| Endpoint | Method | Description |
|---|---|---|
| `/oidc/userinfo` | GET | User claims (scope-filtered) |
| `/oidc/register` | POST | Dynamic client registration |
| `/oidc/device/authorize` | POST | Device authorization request |
| `/oidc/device` | GET | Device verification page |

### Authentication (UI)

| Endpoint | Method | Description |
|---|---|---|
| `/auth/login` | GET | Login page |
| `/auth/login` | POST | Login form submission |
| `/auth/login/json` | POST | Login via JSON API |
| `/auth/register` | GET | Registration page |
| `/auth/register` | POST | Registration form submission |
| `/auth/register/json` | POST | Registration via JSON API |
| `/auth/logout` | POST | Logout (clears session) |

### Admin (Basic Auth protected)

| Endpoint | Method | Description |
|---|---|---|
| `/admin/` | GET | Admin API index |
| `/admin/users` | GET | List users |
| `/admin/users` | POST | Create user |
| `/admin/users/{id}` | DELETE | Delete user |
| `/admin/clients` | GET | List clients |
| `/admin/clients` | POST | Create client |
| `/admin/clients/{id}` | DELETE | Delete client |

## Example: Authorization Code Flow with PKCE

### 1. Generate PKCE Challenge

```bash
# Generate code_verifier (43-128 chars, URL-safe)
CODE_VERIFIER=$(openssl rand -base64 32 | tr -d '=' | tr '+/' '-_')

# Generate code_challenge (S256)
CODE_CHALLENGE=$(echo -n "$CODE_VERIFIER" | openssl dgst -sha256 -binary | openssl base64 -A | tr -d '=' | tr '+/' '-_')
```

### 2. Redirect User to Authorization Endpoint

```
GET /oauth2/authorize?
  client_id=default-client&
  redirect_uri=http://localhost:3000/callback&
  response_type=code&
  scope=openid%20profile%20email&
  state=random-csrf-token&
  code_challenge=$CODE_CHALLENGE&
  code_challenge_method=S256
```

If the user is not logged in, they will be redirected to `/auth/login` to authenticate. After login, they are redirected back to complete the authorization.

### 3. Exchange Code for Tokens

```bash
curl -X POST http://localhost:8080/oauth2/token \
  -u "default-client:default" \
  -d "grant_type=authorization_code" \
  -d "code=AUTHORIZATION_CODE" \
  -d "redirect_uri=http://localhost:3000/callback" \
  -d "code_verifier=$CODE_VERIFIER"
```

Client credentials can also be sent via form body (`client_id` + `client_secret` fields) instead of Basic Auth.

**Response:**
```json
{
  "access_token": "eyJ...",
  "token_type": "Bearer",
  "expires_in": 900,
  "refresh_token": "eyJ...",
  "id_token": "eyJ...",
  "scope": "openid profile email"
}
```

The `id_token` is a separate JWT from `access_token`, containing OIDC-standard claims (`sub`, `aud`, `iss`, `exp`, `iat`, `nonce`).

### 4. Get User Info

```bash
curl -H "Authorization: Bearer ACCESS_TOKEN" \
  http://localhost:8080/oidc/userinfo
```

**Response** (claims depend on granted scopes):
```json
{
  "sub": "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12",
  "name": "Demo User",
  "email": "demo@example.com",
  "email_verified": true,
  "picture": ""
}
```

- `sub` is always returned
- `name`, `picture` require `profile` scope
- `email`, `email_verified` require `email` scope

### 5. Refresh Tokens

```bash
curl -X POST http://localhost:8080/oauth2/token \
  -u "default-client:default" \
  -d "grant_type=refresh_token" \
  -d "refresh_token=REFRESH_TOKEN"
```

Refresh token rotation is mandatory — the old refresh token is always invalidated.

## Key Behaviors

- **RSA key is ephemeral**: A fresh RSA-2048 key pair is generated on every startup. All previously issued tokens become invalid after a server restart.
- **PKCE is mandatory**: Authorization requests without a `code_challenge` are rejected with `400 Bad Request`. Only `S256` is supported.
- **Client secret is validated**: Token exchange and refresh requests validate the client secret against the stored bcrypt hash (unless `token_endpoint_auth_method` is `none`).
- **Session-based authorization**: The `/oauth2/authorize` endpoint checks for a `session_id` cookie. If no valid session exists, the user is redirected to `/auth/login` which creates a session upon successful authentication and redirects back.
- **Token hashing**: Access and refresh tokens are stored as SHA-256 hashes in the database.
- **Scope filtering**: The UserInfo endpoint only returns claims matching the granted scopes.

## Development

```bash
# Run the server (from project root)
go run ./cmd/server

# Build a binary
go build -o server ./cmd/server && ./server

# Hot reload
air

# Migrations (powered by goose)
go run ./cmd/migrate up        # apply all pending migrations
go run ./cmd/migrate down      # rollback the last migration
go run ./cmd/migrate reset     # rollback all migrations
go run ./cmd/migrate status    # show migration status

# Run tests
go test ./...

# Format
go fmt ./...

# Lint
golangci-lint run

# Docker
docker-compose up -d
docker-compose logs -f app
```

## Database Migrations

Migrations are managed by [goose](https://github.com/pressly/goose) via `cmd/migrate`:

```bash
go run ./cmd/migrate up        # apply all pending migrations
go run ./cmd/migrate down      # rollback the last migration
go run ./cmd/migrate reset     # rollback all migrations
go run ./cmd/migrate status    # show migration status
```

Override the database name with the `DATABASE_NAME` env var:

```bash
DATABASE_NAME=oauth2_test go run ./cmd/migrate up
```

## Dependencies

Minimal dependency footprint:

| Dependency | Purpose |
|---|---|
| `golang-jwt/jwt/v5` | JWT signing and validation (RS256) |
| `google/uuid` | UUID generation |
| `jackc/pgx/v5` | PostgreSQL driver and connection pool |
| `pressly/goose/v3` | Database migration management |
| `golang.org/x/crypto` | bcrypt password hashing |
| `golang.org/x/oauth2` | External identity provider support |
| `gopkg.in/yaml.v3` | YAML configuration parsing |

No web framework — uses Go's standard `net/http` with Go 1.22+ method routing.

## License

MIT
