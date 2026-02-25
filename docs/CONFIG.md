# Configuration Reference

## Server Configuration

```yaml
server:
  host: "0.0.0.0"    # Host to bind to
  port: 8080         # Port to listen on
```

## Database Configuration

```yaml
database:
  host: "postgres"      # Database host
  port: 5432            # Database port
  user: "postgres"      # Database user
  password: "secret"    # Database password
  name: "oauth2"        # Database name
  sslmode: "disable"    # SSL mode (disable, require, verify-ca, verify-full)
```

## JWT Configuration

```yaml
jwt:
  issuer: "http://localhost:8080"    # Token issuer URL
  access_token_ttl: "15m"             # Access token TTL (s, m, h, d)
  refresh_token_ttl: "24h"            # Refresh token TTL
  refresh_token_rotation: true        # Enable refresh token rotation
  code_ttl: "5m"                      # Authorization code TTL
```

### Token TTL Format

Token TTL can be specified in the following formats:
- `s` - seconds (e.g., `300s` or `300`)
- `m` - minutes (e.g., `15m`)
- `h` - hours (e.g., `24h`)
- `d` - days (e.g., `7d`)

## OAuth Providers

```yaml
oauth:
  providers:
    - name: "google"                 # Provider identifier
      client_id: ""                  # OAuth2 client ID
      client_secret: ""              # OAuth2 client secret
      scopes:                        # OAuth2 scopes
        - "openid"
        - "profile"
        - "email"
      auth_url: "https://accounts.google.com/o/oauth2/v2/auth"
      token_url: "https://oauth2.googleapis.com/token"
      user_info_url: "https://www.googleapis.com/oauth2/v3/userinfo"
```

### Supported Providers

#### Google
```yaml
- name: "google"
  client_id: "${GOOGLE_CLIENT_ID}"
  client_secret: "${GOOGLE_CLIENT_SECRET}"
  scopes:
    - "openid"
    - "profile"
    - "email"
  auth_url: "https://accounts.google.com/o/oauth2/v2/auth"
  token_url: "https://oauth2.googleapis.com/token"
  user_info_url: "https://www.googleapis.com/oauth2/v3/userinfo"
```

#### GitHub
```yaml
- name: "github"
  client_id: "${GITHUB_CLIENT_ID}"
  client_secret: "${GITHUB_CLIENT_SECRET}"
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
  enabled: true          # Enable admin API
  username: "admin"      # Admin username
  password: "changeme"    # Admin password
```

## Environment Variables

You can also use environment variables in the configuration file:

```yaml
database:
  password: "${DB_PASSWORD}"
```

## Complete Example

```yaml
server:
  host: "0.0.0.0"
  port: 8080

database:
  host: "postgres"
  port: 5432
  user: "postgres"
  password: "${DB_PASSWORD}"
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
      client_id: "${GOOGLE_CLIENT_ID}"
      client_secret: "${GOOGLE_CLIENT_SECRET}"
      scopes:
        - "openid"
        - "profile"
        - "email"
      auth_url: "https://accounts.google.com/o/oauth2/v2/auth"
      token_url: "https://oauth2.googleapis.com/token"
      user_info_url: "https://www.googleapis.com/oauth2/v3/userinfo"

admin:
  enabled: true
  username: "${ADMIN_USERNAME}"
  password: "${ADMIN_PASSWORD}"
```
