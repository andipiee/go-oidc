# Security Considerations

## Authentication

### Password Storage
- Passwords are hashed using bcrypt with default cost factor (10)
- Never store plain-text passwords

### Session Management
- Sessions expire after 24 hours
- Session IDs are cryptographically random (UUID v4)

## Authorization

### OAuth2 Flows

#### Authorization Code Flow
- Short-lived authorization codes (5 minutes)
- PKCE (Proof Key for Code Exchange) support
- Required `redirect_uri` validation

#### Refresh Token Rotation
- Enabled by default
- New refresh token issued on each use
- Old token is revoked

### Scope Validation
- Only allowed scopes are granted
- Client configuration specifies allowed scopes

## Token Security

### Access Tokens
- JWT format with RS256 signing
- Expiration: 15 minutes (configurable)
- Contains: user ID, client ID, scope, claims

### ID Tokens
- JWT format with RS256 signing
- Contains: issuer, subject, audience, expiry, nonce

### Key Management
- RSA 2048-bit key pair
- Keys rotated on restart (for simplicity)

## Data Protection

### Database
- Use SSL/TLS in production (`sslmode: require`)
- Regular database backups
- Principle of least privilege for database user

### Secrets Management
- Never commit secrets to version control
- Use environment variables for sensitive values
- Rotate client secrets periodically

## HTTPS Requirements

In production:
- Always use HTTPS
- Enable HSTS
- Use modern TLS versions (TLS 1.2+)
- Configure secure ciphers

## Rate Limiting

Consider implementing rate limiting for:
- Token endpoint (`/oauth2/token`)
- Authorization endpoint (`/oauth2/authorize`)
- Device authorization (`/oidc/device/authorize`)

## Security Headers

The server includes basic security headers via CORS middleware.

For production, add additional headers:
- `Strict-Transport-Security`
- `Content-Security-Policy`
- `X-Frame-Options`
- `X-Content-Type-Options`

## External Identity Providers

When using external IdPs:
- Validate redirect URIs
- Use state parameter for CSRF protection
- Verify ID token signatures
- Validate issuer and audience claims

## Audit Logging

Recommended to log:
- Authentication attempts
- Token issuances and revocations
- Client registrations
- Authorization code creations

## Vulnerability Disclosure

If you find a security vulnerability, please report it responsibly.
