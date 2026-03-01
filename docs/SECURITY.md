# Security Considerations

## OAuth 2.1 Compliance

This server implements OAuth 2.1 security requirements:

- **PKCE is mandatory** — all authorization requests must include `code_challenge` with `code_challenge_method=S256`. The `plain` method is rejected.
- **Refresh token rotation is mandatory** — old refresh tokens are always revoked when a new one is issued.
- **Client authentication is enforced** — token exchange and refresh requests validate the client secret (unless `token_endpoint_auth_method` is `none`).

## Authentication

### Password Storage
- Passwords are hashed using bcrypt with default cost factor (10)
- Plain-text passwords are never stored or logged

### Session Management
- Sessions expire after 24 hours
- Session IDs are cryptographically random (UUID v4)
- Sessions are stored server-side in PostgreSQL
- The `session_id` cookie is `HttpOnly` and `Secure` (when TLS is detected)

## Authorization

### Authorization Code Flow
- Short-lived authorization codes (5 minutes by default, configurable)
- PKCE (S256) is **required** — requests without `code_challenge` are rejected
- `redirect_uri` is validated against the client's registered URIs
- Authorization codes are single-use (deleted after exchange)

### Refresh Token Rotation
- Always enabled (mandatory per OAuth 2.1)
- New refresh token issued on each use
- Old token is revoked immediately
- Prevents replay attacks with stolen refresh tokens

### Client Secret Validation
- Client secrets are stored as bcrypt hashes
- Token requests validate the secret via `client_secret_basic` (HTTP Basic Auth) or `client_secret_post` (form body)
- Clients with `token_endpoint_auth_method: "none"` skip secret validation

## Token Security

### Access Tokens
- JWT format signed with RS256 (RSA-2048)
- Contains: `sub` (user ID), `client_id`, `scope`, `iss`, `exp`, `iat`
- Default expiration: 15 minutes (configurable)
- Stored as SHA-256 hashes in the database for introspection

### ID Tokens
- Separate JWT from the access token (not a copy)
- Contains OIDC-standard claims: `sub`, `aud` (client ID), `iss`, `exp`, `iat`, `nonce`
- Also includes `email`, `name`, `picture` for convenience
- Signed with the same RS256 key as access tokens

### Refresh Tokens
- JWT format signed with RS256
- Default expiration: 24 hours (configurable)
- Stored as SHA-256 hashes in the database
- Can be revoked via the `/oauth2/revoke` endpoint

### Key Management
- RSA-2048 key pair generated on startup
- **Ephemeral** — keys are not persisted, so all tokens are invalidated on restart
- The JWKS endpoint serves the actual public key (modulus `n` and exponent `e`)
- Key ID (`kid`) is `"1"`

> **Production consideration:** For zero-downtime deployments, implement persistent key storage with key rotation.

## Data Protection

### Database
- Use SSL/TLS in production (`sslmode: require`)
- Regular database backups recommended
- Apply principle of least privilege for the database user

### Secrets Management
- Never commit real secrets to version control
- Use environment variables or a secrets manager for sensitive values
- Rotate client secrets and admin passwords periodically

## HTTPS Requirements

In production:
- Always use HTTPS (required by OAuth 2.1)
- Enable HSTS
- Use TLS 1.2+ with modern cipher suites
- Set `Secure` flag on cookies (automatic when TLS is detected)

## Scope-Based Claims

The UserInfo endpoint filters claims based on the granted scopes:
- `sub` is always returned
- `name`, `picture` require the `profile` scope
- `email`, `email_verified` require the `email` scope

## Security Headers

The server includes CORS headers via middleware. For production, add these headers via your reverse proxy:
- `Strict-Transport-Security: max-age=31536000; includeSubDomains`
- `Content-Security-Policy`
- `X-Frame-Options: DENY`
- `X-Content-Type-Options: nosniff`

## Rate Limiting

Not built-in. Implement rate limiting at the reverse proxy layer for:
- Token endpoint (`/oauth2/token`)
- Authorization endpoint (`/oauth2/authorize`)
- Login endpoints (`/auth/login`, `/auth/login/json`)
- Device authorization (`/oidc/device/authorize`)

## Vulnerability Disclosure

If you find a security vulnerability, please report it responsibly.
