# API Reference

## OpenID Connect Discovery

### GET /.well-known/openid-configuration

Returns the OpenID Provider configuration.

**Response:**
```json
{
  "issuer": "http://localhost:8080",
  "authorization_endpoint": "http://localhost:8080/oauth2/authorize",
  "token_endpoint": "http://localhost:8080/oauth2/token",
  "userinfo_endpoint": "http://localhost:8080/oidc/userinfo",
  "jwks_uri": "http://localhost:8080/.well-known/jwks.json",
  "registration_endpoint": "http://localhost:8080/oidc/register",
  "revocation_endpoint": "http://localhost:8080/oauth2/revoke",
  "introspection_endpoint": "http://localhost:8080/oauth2/introspect",
  "device_authorization_endpoint": "http://localhost:8080/oidc/device/authorize",
  "scopes_supported": ["openid", "profile", "email", "offline_access"],
  "response_types_supported": ["code"],
  "response_modes_supported": ["query", "form_post"],
  "grant_types_supported": ["authorization_code", "refresh_token", "urn:ietf:params:oauth:grant-type:device_code"],
  "token_endpoint_auth_methods_supported": ["client_secret_basic", "client_secret_post", "none"],
  "subject_types_supported": ["public"],
  "id_token_signing_alg_values_supported": ["RS256"],
  "code_challenge_methods_supported": ["S256", "plain"]
}
```

## Authorization Endpoint

### GET /oauth2/authorize

Initiates the OAuth2 authorization flow.

**Query Parameters:**

| Parameter | Required | Description |
|-----------|----------|-------------|
| `client_id` | Yes | The client ID |
| `redirect_uri` | Yes | The redirect URI |
| `response_type` | Yes | Must be `code` |
| `scope` | No | Space-separated scopes (default: `openid`) |
| `state` | Recommended | CSRF protection token |
| `nonce` | No | For ID token validation |
| `code_challenge` | No | PKCE code challenge |
| `code_challenge_method` | No | `S256` or `plain` |

**Response:** Redirects to `redirect_uri` with `code` and `state` query parameters.

### POST /oauth2/authorize

Same as GET but accepts form parameters.

## Token Endpoint

### POST /oauth2/token

Exchanges authorization codes or refresh tokens for tokens.

**Form Parameters:**

| Parameter | Required | Description |
|-----------|----------|-------------|
| `grant_type` | Yes | `authorization_code` or `refresh_token` |
| `code` | Yes* | Authorization code (for `authorization_code` flow) |
| `redirect_uri` | Yes* | Redirect URI (for `authorization_code` flow) |
| `client_id` | Yes | Client ID |
| `client_secret` | Yes* | Client secret (for `client_secret_post` auth) |
| `code_verifier` | No* | PKCE code verifier (if `code_challenge` was used) |
| `refresh_token` | Yes* | Refresh token (for `refresh_token` flow) |

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

## UserInfo Endpoint

### GET /oidc/userinfo

Returns user claims.

**Headers:**
```
Authorization: Bearer <access_token>
```

**Response:**
```json
{
  "sub": "user-id",
  "name": "John Doe",
  "email": "john@example.com",
  "email_verified": true,
  "picture": "https://..."
}
```

## Token Introspection

### POST /oauth2/introspect

Introspects an access or refresh token.

**Form Parameters:**
- `token`: The token to introspect
- `token_type_hint`: Hint about token type (`access_token` or `refresh_token`)

**Response:**
```json
{
  "active": true,
  "client_id": "default-client",
  "scope": "openid profile email",
  "exp": 1699999999
}
```

## Token Revocation

### POST /oauth2/revoke

Revokes an access or refresh token.

**Form Parameters:**
- `token`: The token to revoke
- `token_type_hint`: Hint about token type

## Dynamic Client Registration

### POST /oidc/register

Registers a new OAuth2 client.

**Request:**
```json
{
  "redirect_uris": ["http://localhost:3000/callback"],
  "grant_types": ["authorization_code", "refresh_token"],
  "token_endpoint_auth_method": "client_secret_post",
  "application_type": "web"
}
```

**Response:**
```json
{
  "client_id": "generated-client-id",
  "client_secret": "generated-client-secret",
  "client_id_issued_at": 1699999999,
  "redirect_uris": ["http://localhost:3000/callback"],
  "grant_types": ["authorization_code", "refresh_token"],
  "token_endpoint_auth_method": "client_secret_post"
}
```

## Device Authorization

### POST /oidc/device/authorize

Initiates device authorization flow.

**Form Parameters:**
- `client_id`: Client ID
- `scope`: Space-separated scopes

**Response:**
```json
{
  "device_code": "...",
  "user_code": "ABCD-1234",
  "verification_uri": "http://localhost:8080/oidc/device",
  "verification_uri_complete": "http://localhost:8080/oidc/device?user_code=ABCD-1234",
  "expires_in": 300,
  "interval": 5
}
```

### GET /oidc/device

User visits this URL and enters the user code to authorize the device.
