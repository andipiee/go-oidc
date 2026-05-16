package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/andipiee/go-oidc/internal/domain/repository"
	"github.com/andipiee/go-oidc/internal/infrastructure/auth"
	"github.com/google/uuid"

	"github.com/andipiee/go-oidc/internal/domain/entity"
)

var (
	ErrInvalidClient      = errors.New("invalid client")
	ErrInvalidRedirectURI = errors.New("invalid redirect_uri")
	ErrInvalidUser        = errors.New("invalid user")
	ErrInvalidScope       = errors.New("invalid scope")
	ErrInvalidCode        = errors.New("invalid authorization code")
	ErrExpiredCode        = errors.New("authorization code expired")
	ErrInvalidGrantType   = errors.New("invalid grant type")
	ErrInvalidToken       = errors.New("invalid token")
)

type AuthorizeUseCase struct {
	clientRepo    repository.ClientRepository
	userRepo      repository.UserRepository
	tokenRepo     repository.TokenRepository
	jwtService    *auth.JWTService
	cryptoService *auth.CryptoService
	codeTTL       int64
}

func NewAuthorizeUseCase(
	clientRepo repository.ClientRepository,
	userRepo repository.UserRepository,
	tokenRepo repository.TokenRepository,
	jwtService *auth.JWTService,
	cryptoService *auth.CryptoService,
	codeTTL int64,
) *AuthorizeUseCase {
	return &AuthorizeUseCase{
		clientRepo:    clientRepo,
		userRepo:      userRepo,
		tokenRepo:     tokenRepo,
		jwtService:    jwtService,
		cryptoService: cryptoService,
		codeTTL:       codeTTL,
	}
}

type AuthorizeParams struct {
	ClientID            string
	RedirectURI         string
	ResponseType        string
	Scope               string
	State               string
	Nonce               string
	CodeChallenge       string
	CodeChallengeMethod string
	UserID              uuid.UUID
}

type AuthorizeResult struct {
	Code        string
	State       string
	RedirectURI string
}

func (uc *AuthorizeUseCase) Authorize(ctx context.Context, params AuthorizeParams) (*AuthorizeResult, error) {
	client, err := uc.clientRepo.GetByClientID(ctx, params.ClientID)
	if err != nil || client == nil {
		return nil, ErrInvalidClient
	}

	validRedirect := false
	for _, uri := range client.RedirectURIs {
		if uri == params.RedirectURI {
			validRedirect = true
			break
		}
	}
	if !validRedirect {
		return nil, ErrInvalidRedirectURI
	}

	user, err := uc.userRepo.GetByID(ctx, params.UserID)
	if err != nil || user == nil {
		return nil, ErrInvalidUser
	}

	code := uc.cryptoService.GenerateRandomString(32)
	authCode := &entity.AuthorizationCode{
		ID:                  uuid.Must(uuid.NewV7()),
		Code:                code,
		ClientID:            params.ClientID,
		UserID:              user.ID,
		RedirectURI:         params.RedirectURI,
		Scope:               params.Scope,
		Nonce:               params.Nonce,
		CodeChallenge:       params.CodeChallenge,
		CodeChallengeMethod: params.CodeChallengeMethod,
		ExpiresAt:           time.Now().Add(time.Duration(uc.codeTTL) * time.Second),
		CreatedAt:           time.Now(),
	}

	if err := uc.tokenRepo.CreateAuthorizationCode(ctx, authCode); err != nil {
		return nil, fmt.Errorf("failed to create authorization code: %w", err)
	}

	return &AuthorizeResult{
		Code:        code,
		State:       params.State,
		RedirectURI: params.RedirectURI,
	}, nil
}
