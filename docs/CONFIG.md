# Configuration Reference

All configuration is loaded from `configs/config.yaml`. The server must be started from the project root directory since the config path is relative.

## Server Configuration

```yaml
server:
  host: "0.0.0.0"    # Host to bind to
  port: 8080          # Port to listen on
```

## Database Configuration

```yaml
database:
  host: "localhost"       # Database host
  port: 5432              # Database port
  user: "postgres"        # Database user
  password: "changeme"    # Database password (change in production)
  name: "oauth2"          # Database name
  sslmode: "disable"      # SSL mode (disable, require, verify-ca, verify-full)
```

> **Production:** Always set `sslmode: "require"` or stricter in production environments.

## JWT Configuration

```yaml
jwt:
  issuer: "http://localhost:8080"    # Token issuer URL (must match your public URL)
  access_token_ttl: "15m"           # Access token time-to-live
  refresh_token_ttl: "24h"          # Refresh token time-to-live
  refresh_token_rotation: true      # (Legacy field — rotation is now always mandatory)
  code_ttl: "5m"                    # Authorization code time-to-live
```

### Token TTL Format

Supported duration suffixes:
- `s` — seconds (e.g., `300s`)
- `m` — minutes (e.g., `15m`)
- `h` — hours (e.g., `24h`)
- `d` — days (e.g., `7d`)

### Key Management

The server generates a fresh RSA-2048 key pair on every startup. This means:
- All previously issued tokens become invalid after a restart
- The JWKS endpoint (`/.well-known/jwks.json`) always reflects the current key
- For production, consider implementing persistent key storage

## OAuth Providers

External identity providers can be configured for federated login:

```yaml
oauth:
  providers:
    - name: "google"
      client_id: ""          # Your OAuth2 client ID
      client_secret: ""      # Your OAuth2 client secret
      scopes:
        - "openid"
        - "profile"
        - "email"
      auth_url: "https://accounts.google.com/o/oauth2/v2/auth"
      token_url: "https://oauth2.googleapis.com/token"
      user_info_url: "https://www.googleapis.com/oauth2/v3/userinfo"

    - name: "github"
      client_id: ""
      client_secret: ""
      scopes:
        - "read:user"
        - "user:email"
      auth_url: "https://github.com/login/oauth/authorize"
      token_url: "https://github.com/login/oauth/access_token"
      user_info_url: "https://api.github.com/user"
```

## Admin Configuration

```yaml
admin:
  enabled: true            # Enable/disable admin API endpoints
  username: "admin"        # Admin username for Basic Auth
  password: "changeme"     # Admin password (change in production)
```

## Complete Example

```yaml
server:
  host: "0.0.0.0"
  port: 8080

database:
  host: "localhost"
  port: 5432
  user: "postgres"
  password: "changeme"
  name: "oauth2"
  sslmode: "disable"

jwt:
  issuer: "http://localhost:8080"
  access_token_ttl: "15m"
  refresh_token_ttl: "24h"
  refresh_token_rotation: true
  code_ttl: "5m"

oauth:
  providers:
    - name: "google"
      client_id: ""
      client_secret: ""
      scopes:
        - "openid"
        - "profile"
        - "email"
      auth_url: "https://accounts.google.com/o/oauth2/v2/auth"
      token_url: "https://oauth2.googleapis.com/token"
      user_info_url: "https://www.googleapis.com/oauth2/v3/userinfo"
    - name: "github"
      client_id: ""
      client_secret: ""
      scopes:
        - "read:user"
        - "user:email"
      auth_url: "https://github.com/login/oauth/authorize"
      token_url: "https://github.com/login/oauth/access_token"
      user_info_url: "https://api.github.com/user"

admin:
  enabled: true
  username: "admin"
  password: "changeme"
```
