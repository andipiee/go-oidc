package entity

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID            uuid.UUID `json:"id"`
	Email         string    `json:"email"`
	PasswordHash  string    `json:"-"`
	Name          string    `json:"name"`
	EmailVerified bool      `json:"email_verified"`
	Picture       string    `json:"picture"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type Updated struct {
	UpdatedAt time.Time `json:"updated_at"`
}

type Client struct {
	ID                      uuid.UUID `json:"id"`
	ClientID                string    `json:"client_id"`
	ClientSecretHash        string    `json:"-"`
	Name                    string    `json:"name"`
	RedirectURIs            []string  `json:"redirect_uris"`
	GrantTypes              []string  `json:"grant_types"`
	ResponseTypes           []string  `json:"response_types"`
	TokenEndpointAuthMethod string    `json:"token_endpoint_auth_method"`
	Scopes                  []string  `json:"scopes"`
	CreatedAt               time.Time `json:"created_at"`
	UpdatedAt               time.Time `json:"updated_at"`
}

type AuthorizationCode struct {
	ID                  uuid.UUID `json:"id"`
	Code                string    `json:"code"`
	ClientID            string    `json:"client_id"`
	UserID              uuid.UUID `json:"user_id"`
	RedirectURI         string    `json:"redirect_uri"`
	Scope               string    `json:"scope"`
	Nonce               string    `json:"nonce"`
	CodeChallenge       string    `json:"code_challenge"`
	CodeChallengeMethod string    `json:"code_challenge_method"`
	ExpiresAt           time.Time `json:"expires_at"`
	CreatedAt           time.Time `json:"created_at"`
}

type AccessToken struct {
	ID        uuid.UUID `json:"id"`
	TokenHash string    `json:"-"`
	ClientID  string    `json:"client_id"`
	UserID    uuid.UUID `json:"user_id"`
	Scope     string    `json:"scope"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

type RefreshToken struct {
	ID        uuid.UUID  `json:"id"`
	TokenHash string     `json:"-"`
	ClientID  string     `json:"client_id"`
	UserID    uuid.UUID  `json:"user_id"`
	Scope     string     `json:"scope"`
	ExpiresAt time.Time  `json:"expires_at"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

type UserSession struct {
	ID        uuid.UUID `json:"id"`
	SessionID string    `json:"session_id"`
	UserID    uuid.UUID `json:"user_id"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

type ExternalAccount struct {
	ID             uuid.UUID `json:"id"`
	UserID         uuid.UUID `json:"user_id"`
	Provider       string    `json:"provider"`
	ProviderUserID string    `json:"provider_user_id"`
	CreatedAt      time.Time `json:"created_at"`
}

type DeviceCode struct {
	ID              uuid.UUID  `json:"id"`
	DeviceCode      string     `json:"device_code"`
	UserCode        string     `json:"user_code"`
	ClientID        string     `json:"client_id"`
	Scope           string     `json:"scope"`
	ExpiresAt       time.Time  `json:"expires_at"`
	PollingInterval int        `json:"polling_interval"`
	Verified        bool       `json:"verified"`
	UserID          *uuid.UUID `json:"user_id,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}
