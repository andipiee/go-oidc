# OAuth2 Server

A standards-compliant OAuth2 and OpenID Connect (OIDC) server implementation in Go using Clean Architecture.

## Features

- **OAuth2 Flows**
  - Authorization Code Flow with PKCE
  - Refresh Token Flow
  - Device Authorization Flow

- **OpenID Connect**
  - OIDC Discovery (`/.well-known/openid-configuration`)
  - UserInfo endpoint
  - JWKS endpoint
  - Dynamic Client Registration

- **Security**
  - JWT tokens (RS256)
  - Password hashing (bcrypt)
  - Token rotation
  - Code challenge verification

- **External Identity Providers**
  - Google OAuth2
  - GitHub OAuth2
  - Generic OIDC support

- **Admin API**
  - User management
  - Client management

## Quick Start

### Prerequisites

- Go 1.21+
- Docker & Docker Compose
- PostgreSQL 15 (optional if using Docker)

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

# Run migrations
psql -U postgres -d oauth2 -f migrations/001_init.sql

# Build and run
go build -o server ./cmd/server
./server
```

## Configuration

Configuration is managed via `configs/config.yaml`:

```yaml
server:
  host: "0.0.0.0"
  port: 8080

database:
  host: "postgres"
  port: 5432
  user: "postgres"
  password: "secret"
  name: "oauth2"
  sslmode: "disable"

jwt:
  issuer: "http://localhost:8080"
  access_token_ttl: "15m"
  refresh_token_ttl: "24h"
  refresh_token_rotation: true
  code_ttl: "5m"

admin:
  enabled: true
  username: "admin"
  password: "changeme"
```

## Default Credentials

- **Admin UI**: http://localhost:8080/admin/
- **Username**: admin
- **Password**: changeme

- **Demo User**: demo@example.com / default
- **Default Client ID**: default-client
- **Default Client Secret**: default

## Architecture

The project follows Clean Architecture:

```
cmd/server/          # Application entry point
internal/
  domain/            # Entities and repository interfaces
  application/       # Use cases and business logic
  infrastructure/   # Database, auth, OAuth providers
  presentation/     # HTTP handlers and middleware
web/static/          # Static web files
migrations/          # SQL migrations
configs/            # Configuration files
docker/              # Docker configuration
```

## API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/.well-known/openid-configuration` | GET | OIDC Discovery |
| `/.well-known/jwks.json` | GET | JSON Web Key Set |
| `/oauth2/authorize` | GET/POST | Authorization endpoint |
| `/oauth2/token` | POST | Token exchange |
| `/oidc/userinfo` | GET | User claims |
| `/oauth2/revoke` | POST | Token revocation |
| `/oauth2/introspect` | POST | Token introspection |
| `/oidc/register` | POST | Dynamic client registration |
| `/oidc/device/authorize` | POST | Device authorization |
| `/admin/users` | GET/POST/DELETE | User management |
| `/admin/clients` | GET/POST/DELETE | Client management |

## Example: Authorization Code Flow

### 1. Redirect to Authorization Endpoint

```
GET /oauth2/authorize?
  client_id=default-client&
  redirect_uri=http://localhost:3000/callback&
  response_type=code&
  scope=openid%20profile%20email&
  state=random&
  code_challenge=...&
  code_challenge_method=S256
```

### 2. Exchange Code for Tokens

```bash
curl -X POST http://localhost:8080/oauth2/token \
  -d "grant_type=authorization_code" \
  -d "code=..." \
  -d "redirect_uri=http://localhost:3000/callback" \
  -d "client_id=default-client" \
  -d "client_secret=default" \
  -d "code_verifier=..."
```

Response:
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

### 3. Get User Info

```bash
curl -H "Authorization: Bearer <access_token>" \
  http://localhost:8080/oidc/userinfo
```

## Development

```bash
# Run tests
go test ./...

# Run with hot reload
air

# Format code
go fmt ./...

# Lint
golangci-lint run
```

## License

MIT
