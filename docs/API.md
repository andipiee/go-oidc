# API Reference

## OpenID Connect Discovery

### GET /.well-known/openid-configuration

Returns the OpenID Provider configuration metadata.

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
  "code_challenge_methods_supported": ["S256"],
  "claims_supported": ["sub", "name", "email", "email_verified", "picture"],
  "claims_parameter_supported": false,
  "request_parameter_supported": false,
  "request_uri_parameter_supported": false
}
```

### GET /.well-known/jwks.json

Returns the JSON Web Key Set containing the server's RSA public key.

**Response:**
```json
{
  "keys": [
    {
      "kty": "RSA",
      "use": "sig",
      "kid": "1",
      "alg": "RS256",
      "n": "<base64url-encoded RSA modulus>",
      "e": "<base64url-encoded RSA exponent>"
    }
  ]
}
```

> **Note:** The RSA key pair is regenerated on every server restart. Previously issued tokens will not validate after a restart.

---

## Authorization Endpoint

### GET /oauth2/authorize

Initiates the OAuth2 Authorization Code flow. Requires PKCE (S256).

**Query Parameters:**

| Parameter | Required | Description |
|---|---|---|
| `client_id` | Yes | The registered client ID |
| `redirect_uri` | Yes | Must match one of the client's registered redirect URIs |
| `response_type` | Yes | Must be `code` |
| `scope` | No | Space-separated scopes (default: `openid`) |
| `state` | Recommended | CSRF protection token, returned in the redirect |
| `nonce` | No | Binds the ID token to the client session |
| `code_challenge` | **Yes** | PKCE code challenge (base64url-encoded SHA-256 of the verifier) |
| `code_challenge_method` | No | Must be `S256` (default: `S256`). `plain` is rejected. |

**Behavior:**
- If the user has no valid session (`session_id` cookie), they are redirected to `/auth/login` with all authorization parameters preserved.
- After successful login, the user is redirected back to `/oauth2/authorize` to complete the flow.
- On success, redirects to `redirect_uri` with `code` and `state` query parameters.

**Error Responses:**
- `400` with `invalid_request` if required parameters are missing or `code_challenge` is absent.
- `400` with `unsupported_response_type` if `response_type` is not `code`.
- `400` with `invalid_request` if `code_challenge_method` is not `S256`.

### POST /oauth2/authorize

Same as GET but accepts form parameters.

---

## Token Endpoint

### POST /oauth2/token

Exchanges authorization codes or refresh tokens for tokens.

**Client Authentication:**
- **Basic Auth** (preferred): `Authorization: Basic base64(client_id:client_secret)`
- **Form body**: Include `client_id` and `client_secret` as form parameters
- Basic Auth takes precedence if both are provided

**Form Parameters (Authorization Code):**

| Parameter | Required | Description |
|---|---|---|
| `grant_type` | Yes | `authorization_code` |
| `code` | Yes | The authorization code received from `/oauth2/authorize` |
| `redirect_uri` | Yes | Must match the `redirect_uri` used in the authorization request |
| `client_id` | Yes | Client ID (unless using Basic Auth) |
| `client_secret` | Yes* | Client secret (* not required if `token_endpoint_auth_method` is `none`) |
| `code_verifier` | **Yes** | PKCE code verifier (the original random string) |

**Form Parameters (Refresh Token):**

| Parameter | Required | Description |
|---|---|---|
| `grant_type` | Yes | `refresh_token` |
| `refresh_token` | Yes | The refresh token |
| `client_id` | Yes | Client ID (unless using Basic Auth) |
| `client_secret` | Yes* | Client secret (* not required if `token_endpoint_auth_method` is `none`) |

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

- `id_token` is only included when `openid` is in the scope. It is a distinct JWT from `access_token`, containing `sub`, `aud` (client ID), `iss`, `exp`, `iat`, and `nonce` (on initial exchange only).
- `refresh_token` is only included when `openid` is in the scope.
- On refresh, the old refresh token is **always** revoked (mandatory rotation).

**Error Responses:**
- `400` with `unsupported_grant_type` for unrecognized grant types
- `400` with `invalid_grant` for expired codes, invalid PKCE verifiers, or invalid client credentials

---

## UserInfo Endpoint

### GET /oidc/userinfo

Returns user claims. Claims are filtered based on the scopes in the access token.

**Headers:**
```
Authorization: Bearer <access_token>
```

**Response (full scopes: `openid profile email`):**
```json
{
  "sub": "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12",
  "name": "Demo User",
  "picture": "",
  "email": "demo@example.com",
  "email_verified": true
}
```

**Scope-based filtering:**

| Scope | Claims included |
|---|---|
| (always) | `sub` |
| `profile` | `name`, `picture` |
| `email` | `email`, `email_verified` |

A request with only `scope=openid` returns `{"sub": "..."}`.

---

## Token Introspection

### POST /oauth2/introspect

Introspects an access token (RFC 7662).

**Form Parameters:**
- `token` (required): The token to introspect
- `token_type_hint` (optional): `access_token` or `refresh_token`

**Response (active):**
```json
{
  "active": true,
  "client_id": "default-client",
  "scope": "openid profile email",
  "exp": 1699999999
}
```

**Response (inactive):**
```json
{
  "active": false
}
```

---

## Token Revocation

### POST /oauth2/revoke

Revokes a refresh token (RFC 7009).

**Form Parameters:**
- `token` (required): The token to revoke
- `token_type_hint` (optional): `refresh_token`

**Response:** `200 OK` with empty JSON body `{}`.

---

## Dynamic Client Registration

### POST /oidc/register

Registers a new OAuth2 client dynamically (RFC 7591).

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

> **Important:** Store the `client_secret` immediately — it is hashed and cannot be retrieved again.

---

## Device Authorization

### POST /oidc/device/authorize

Initiates the device authorization flow (RFC 8628).

**Form Parameters:**
- `client_id` (required): The client ID
- `scope` (optional): Space-separated scopes (default: `openid`)

**Response:**
```json
{
  "device_code": "...",
  "user_code": "ABCD1234",
  "verification_uri": "/oidc/device",
  "verification_uri_complete": "/oidc/device?user_code=ABCD1234",
  "expires_in": 300,
  "interval": 5
}
```

### GET /oidc/device

Device verification page. User visits this URL and enters the `user_code` to authorize the device.

---

## Authentication Endpoints

### GET /auth/login

Renders the login page. Accepts optional query parameters (`client_id`, `redirect_uri`, `state`, `scope`, `nonce`, `code_challenge`, `code_challenge_method`) which are preserved through the login flow and used to redirect back to `/oauth2/authorize`.

### POST /auth/login

Handles login form submission. On success:
- Creates a server-side session
- Sets a `session_id` cookie
- If OAuth2 parameters are present, redirects to `/oauth2/authorize` to complete the authorization flow
- Otherwise, returns a JSON response with user info

### POST /auth/login/json

JSON-based login API. Returns access and refresh tokens directly (sets cookies too).

**Request:**
```json
{
  "email": "demo@example.com",
  "password": "default"
}
```

**Response:**
```json
{
  "access_token": "eyJ...",
  "refresh_token": "eyJ...",
  "token_type": "Bearer",
  "expires_in": 900
}
```

### GET /auth/register

Renders the registration page.

### POST /auth/register

Handles registration form submission.

### POST /auth/register/json

JSON-based registration API.

**Request:**
```json
{
  "email": "user@example.com",
  "name": "New User",
  "password": "securepassword"
}
```

### POST /auth/logout

Clears all session and token cookies.

---

## Admin API

All admin endpoints require HTTP Basic Auth (`admin` / `changeme` by default).

### GET /admin/

Returns available admin endpoints.

### GET /admin/users

Lists all users (limit 100).

### POST /admin/users

Creates a new user.

**Request:**
```json
{
  "email": "newuser@example.com",
  "name": "New User",
  "password": "password123"
}
```

### DELETE /admin/users/{id}

Deletes a user by UUID.

### GET /admin/clients

Lists all registered clients.

### POST /admin/clients

Creates a new client. Returns the generated `client_id` and `client_secret`.

**Request:**
```json
{
  "name": "My App",
  "redirect_uris": ["http://localhost:3000/callback"],
  "grant_types": ["authorization_code", "refresh_token"],
  "token_endpoint_auth_method": "client_secret_post"
}
```

### DELETE /admin/clients/{id}

Deletes a client by UUID.

---

## Health Check

### GET /health

**Response:**
```json
{
  "status": "ok"
}
```
