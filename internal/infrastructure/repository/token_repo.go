package repository

import (
	"context"
	"time"

	"github.com/andipiee/go-oidc/internal/domain/entity"
	"github.com/andipiee/go-oidc/internal/infrastructure/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type TokenRepository struct {
	db *database.PostgresDB
}

func NewTokenRepository(db *database.PostgresDB) *TokenRepository {
	return &TokenRepository{db: db}
}

func (r *TokenRepository) CreateAuthorizationCode(ctx context.Context, code *entity.AuthorizationCode) error {
	query := `
		INSERT INTO authorization_codes (id, code, client_id, user_id, redirect_uri, scope, nonce,
		code_challenge, code_challenge_method, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	_, err := r.db.Pool().Exec(ctx, query,
		code.ID, code.Code, code.ClientID, code.UserID, code.RedirectURI, code.Scope,
		code.Nonce, code.CodeChallenge, code.CodeChallengeMethod, code.ExpiresAt, code.CreatedAt,
	)
	return err
}

func (r *TokenRepository) GetAuthorizationCode(ctx context.Context, code string) (*entity.AuthorizationCode, error) {
	query := `
		SELECT id, code, client_id, user_id, redirect_uri, scope, nonce, code_challenge,
		code_challenge_method, expires_at, created_at
		FROM authorization_codes WHERE code = $1
	`
	var c entity.AuthorizationCode
	err := r.db.Pool().QueryRow(ctx, query, code).Scan(
		&c.ID, &c.Code, &c.ClientID, &c.UserID, &c.RedirectURI, &c.Scope, &c.Nonce,
		&c.CodeChallenge, &c.CodeChallengeMethod, &c.ExpiresAt, &c.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &c, err
}

func (r *TokenRepository) DeleteAuthorizationCode(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM authorization_codes WHERE id = $1`
	_, err := r.db.Pool().Exec(ctx, query, id)
	return err
}

func (r *TokenRepository) CreateAccessToken(ctx context.Context, token *entity.AccessToken) error {
	query := `
		INSERT INTO access_tokens (id, token_hash, client_id, user_id, scope, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.db.Pool().Exec(ctx, query,
		token.ID, token.TokenHash, token.ClientID, token.UserID, token.Scope, token.ExpiresAt, token.CreatedAt,
	)
	return err
}

func (r *TokenRepository) GetAccessToken(ctx context.Context, tokenHash string) (*entity.AccessToken, error) {
	query := `
		SELECT id, token_hash, client_id, user_id, scope, expires_at, created_at
		FROM access_tokens WHERE token_hash = $1 AND expires_at > NOW()
	`
	var token entity.AccessToken
	err := r.db.Pool().QueryRow(ctx, query, tokenHash).Scan(
		&token.ID, &token.TokenHash, &token.ClientID, &token.UserID, &token.Scope,
		&token.ExpiresAt, &token.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &token, err
}

func (r *TokenRepository) DeleteAccessToken(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM access_tokens WHERE id = $1`
	_, err := r.db.Pool().Exec(ctx, query, id)
	return err
}

func (r *TokenRepository) CreateRefreshToken(ctx context.Context, token *entity.RefreshToken) error {
	query := `
		INSERT INTO refresh_tokens (id, token_hash, client_id, user_id, scope, expires_at, revoked_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.db.Pool().Exec(ctx, query,
		token.ID, token.TokenHash, token.ClientID, token.UserID, token.Scope,
		token.ExpiresAt, token.RevokedAt, token.CreatedAt,
	)
	return err
}

func (r *TokenRepository) GetRefreshToken(ctx context.Context, tokenHash string) (*entity.RefreshToken, error) {
	query := `
		SELECT id, token_hash, client_id, user_id, scope, expires_at, revoked_at, created_at
		FROM refresh_tokens WHERE token_hash = $1
	`
	var token entity.RefreshToken
	var revokedAt *time.Time
	err := r.db.Pool().QueryRow(ctx, query, tokenHash).Scan(
		&token.ID, &token.TokenHash, &token.ClientID, &token.UserID, &token.Scope,
		&token.ExpiresAt, &revokedAt, &token.CreatedAt,
	)
	token.RevokedAt = revokedAt
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &token, err
}

func (r *TokenRepository) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	query := `UPDATE refresh_tokens SET revoked_at = $1 WHERE token_hash = $2`
	_, err := r.db.Pool().Exec(ctx, query, time.Now(), tokenHash)
	return err
}

func (r *TokenRepository) DeleteRefreshToken(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM refresh_tokens WHERE id = $1`
	_, err := r.db.Pool().Exec(ctx, query, id)
	return err
}

func (r *TokenRepository) CreateDeviceCode(ctx context.Context, deviceCode *entity.DeviceCode) error {
	query := `
		INSERT INTO device_codes (id, device_code, user_code, client_id, scope, expires_at,
		polling_interval, verified, user_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := r.db.Pool().Exec(ctx, query,
		deviceCode.ID, deviceCode.DeviceCode, deviceCode.UserCode, deviceCode.ClientID,
		deviceCode.Scope, deviceCode.ExpiresAt, deviceCode.PollingInterval, deviceCode.Verified,
		deviceCode.UserID, deviceCode.CreatedAt,
	)
	return err
}

func (r *TokenRepository) GetDeviceCode(ctx context.Context, deviceCode string) (*entity.DeviceCode, error) {
	query := `
		SELECT id, device_code, user_code, client_id, scope, expires_at, polling_interval, verified, user_id, created_at
		FROM device_codes WHERE device_code = $1
	`
	var dc entity.DeviceCode
	var userID *uuid.UUID
	err := r.db.Pool().QueryRow(ctx, query, deviceCode).Scan(
		&dc.ID, &dc.DeviceCode, &dc.UserCode, &dc.ClientID, &dc.Scope,
		&dc.ExpiresAt, &dc.PollingInterval, &dc.Verified, &userID, &dc.CreatedAt,
	)
	dc.UserID = userID
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &dc, err
}

func (r *TokenRepository) GetDeviceCodeByUserCode(ctx context.Context, userCode string) (*entity.DeviceCode, error) {
	query := `
		SELECT id, device_code, user_code, client_id, scope, expires_at, polling_interval, verified, user_id, created_at
		FROM device_codes WHERE user_code = $1
	`
	var dc entity.DeviceCode
	var userID *uuid.UUID
	err := r.db.Pool().QueryRow(ctx, query, userCode).Scan(
		&dc.ID, &dc.DeviceCode, &dc.UserCode, &dc.ClientID, &dc.Scope,
		&dc.ExpiresAt, &dc.PollingInterval, &dc.Verified, &userID, &dc.CreatedAt,
	)
	dc.UserID = userID
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &dc, err
}

func (r *TokenRepository) UpdateDeviceCode(ctx context.Context, deviceCode *entity.DeviceCode) error {
	query := `
		UPDATE device_codes SET verified = $2, user_id = $3 WHERE id = $1
	`
	_, err := r.db.Pool().Exec(ctx, query, deviceCode.ID, deviceCode.Verified, deviceCode.UserID)
	return err
}

type SessionRepository struct {
	db *database.PostgresDB
}

func NewSessionRepository(db *database.PostgresDB) *SessionRepository {
	return &SessionRepository{db: db}
}

func (r *SessionRepository) Create(ctx context.Context, session *entity.UserSession) error {
	query := `
		INSERT INTO user_sessions (id, session_id, user_id, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := r.db.Pool().Exec(ctx, query,
		session.ID, session.SessionID, session.UserID, session.ExpiresAt, session.CreatedAt,
	)
	return err
}

func (r *SessionRepository) GetBySessionID(ctx context.Context, sessionID string) (*entity.UserSession, error) {
	query := `
		SELECT id, session_id, user_id, expires_at, created_at
		FROM user_sessions WHERE session_id = $1 AND expires_at > NOW()
	`
	var session entity.UserSession
	err := r.db.Pool().QueryRow(ctx, query, sessionID).Scan(
		&session.ID, &session.SessionID, &session.UserID, &session.ExpiresAt, &session.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &session, err
}

func (r *SessionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM user_sessions WHERE id = $1`
	_, err := r.db.Pool().Exec(ctx, query, id)
	return err
}

func (r *SessionRepository) DeleteBySessionID(ctx context.Context, sessionID string) error {
	query := `DELETE FROM user_sessions WHERE session_id = $1`
	_, err := r.db.Pool().Exec(ctx, query, sessionID)
	return err
}
