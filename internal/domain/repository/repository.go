package repository

import (
	"context"

	"github.com/andipiee/go-oidc/internal/domain/entity"
	"github.com/google/uuid"
)

type UserRepository interface {
	Create(ctx context.Context, user *entity.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*entity.User, error)
	GetByEmail(ctx context.Context, email string) (*entity.User, error)
	Update(ctx context.Context, user *entity.User) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, limit, offset int) ([]*entity.User, error)
	GetByExternalAccount(ctx context.Context, provider, providerUserID string) (*entity.User, error)
	CreateExternalAccount(ctx context.Context, account *entity.ExternalAccount) error
}

type ClientRepository interface {
	Create(ctx context.Context, client *entity.Client) error
	GetByID(ctx context.Context, id uuid.UUID) (*entity.Client, error)
	GetByClientID(ctx context.Context, clientID string) (*entity.Client, error)
	Update(ctx context.Context, client *entity.Client) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, limit, offset int) ([]*entity.Client, error)
}

type TokenRepository interface {
	CreateAuthorizationCode(ctx context.Context, code *entity.AuthorizationCode) error
	GetAuthorizationCode(ctx context.Context, code string) (*entity.AuthorizationCode, error)
	DeleteAuthorizationCode(ctx context.Context, id uuid.UUID) error

	CreateAccessToken(ctx context.Context, token *entity.AccessToken) error
	GetAccessToken(ctx context.Context, tokenHash string) (*entity.AccessToken, error)
	DeleteAccessToken(ctx context.Context, id uuid.UUID) error

	CreateRefreshToken(ctx context.Context, token *entity.RefreshToken) error
	GetRefreshToken(ctx context.Context, tokenHash string) (*entity.RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, tokenHash string) error
	DeleteRefreshToken(ctx context.Context, id uuid.UUID) error

	CreateDeviceCode(ctx context.Context, deviceCode *entity.DeviceCode) error
	GetDeviceCode(ctx context.Context, deviceCode string) (*entity.DeviceCode, error)
	UpdateDeviceCode(ctx context.Context, deviceCode *entity.DeviceCode) error
	GetDeviceCodeByUserCode(ctx context.Context, userCode string) (*entity.DeviceCode, error)
}

type SessionRepository interface {
	Create(ctx context.Context, session *entity.UserSession) error
	GetBySessionID(ctx context.Context, sessionID string) (*entity.UserSession, error)
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteBySessionID(ctx context.Context, sessionID string) error
}
