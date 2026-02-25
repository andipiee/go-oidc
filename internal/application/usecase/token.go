package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/andipiee/go-oidc/internal/domain/entity"
	"github.com/andipiee/go-oidc/internal/domain/repository"
	"github.com/andipiee/go-oidc/internal/infrastructure/auth"
	"github.com/google/uuid"
)

type TokenUseCase struct {
	clientRepo    repository.ClientRepository
	userRepo      repository.UserRepository
	tokenRepo     repository.TokenRepository
	jwtService    *auth.JWTService
	cryptoService *auth.CryptoService
	config        JWTConfig
}

type JWTConfig struct {
	Issuer               string
	AccessTokenTTLDur    int64
	RefreshTokenTTLDur   int64
	RefreshTokenRotation bool
	CodeTTLDur           int64
}

func NewTokenUseCase(
	clientRepo repository.ClientRepository,
	userRepo repository.UserRepository,
	tokenRepo repository.TokenRepository,
	jwtService *auth.JWTService,
	cryptoService *auth.CryptoService,
	config JWTConfig,
) *TokenUseCase {
	return &TokenUseCase{
		clientRepo:    clientRepo,
		userRepo:      userRepo,
		tokenRepo:     tokenRepo,
		jwtService:    jwtService,
		cryptoService: cryptoService,
		config:        config,
	}
}

type TokenRequest struct {
	GrantType    string
	Code         string
	RedirectURI  string
	ClientID     string
	ClientSecret string
	CodeVerifier string
	RefreshToken string
	Scope        string
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	IDToken      string `json:"id_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
}

func (uc *TokenUseCase) ExchangeCode(ctx context.Context, req TokenRequest) (*TokenResponse, error) {
	if req.GrantType != "authorization_code" {
		return nil, ErrInvalidGrantType
	}

	code, err := uc.tokenRepo.GetAuthorizationCode(ctx, req.Code)
	if err != nil || code == nil {
		return nil, ErrInvalidCode
	}

	if time.Now().After(code.ExpiresAt) {
		return nil, ErrExpiredCode
	}

	client, err := uc.clientRepo.GetByClientID(ctx, req.ClientID)
	if err != nil || client == nil || client.ClientID != code.ClientID {
		return nil, ErrInvalidClient
	}

	if code.RedirectURI != req.RedirectURI {
		return nil, ErrInvalidRedirectURI
	}

	if code.CodeChallenge != "" {
		if !uc.cryptoService.VerifyCodeChallenge(req.CodeVerifier, code.CodeChallenge, code.CodeChallengeMethod) {
			return nil, ErrInvalidCode
		}
	}

	user, err := uc.userRepo.GetByID(ctx, code.UserID)
	if err != nil || user == nil {
		return nil, ErrInvalidUser
	}

	uc.tokenRepo.DeleteAuthorizationCode(ctx, code.ID)

	accessToken, err := uc.jwtService.GenerateAccessToken(
		user.ID.String(),
		client.ClientID,
		code.Scope,
		user.Email,
		user.Name,
		user.Picture,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := uc.jwtService.GenerateRefreshToken(user.ID.String(), client.ClientID, code.Scope)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	accessTokenHash := uc.jwtService.HashToken(accessToken)
	refreshTokenHash := uc.jwtService.HashToken(refreshToken)

	uc.tokenRepo.CreateAccessToken(ctx, &entity.AccessToken{
		ID:        uuid.New(),
		TokenHash: accessTokenHash,
		ClientID:  client.ClientID,
		UserID:    user.ID,
		Scope:     code.Scope,
		ExpiresAt: time.Now().Add(time.Duration(uc.config.AccessTokenTTLDur) * time.Second),
		CreatedAt: time.Now(),
	})

	uc.tokenRepo.CreateRefreshToken(ctx, &entity.RefreshToken{
		ID:        uuid.New(),
		TokenHash: refreshTokenHash,
		ClientID:  client.ClientID,
		UserID:    user.ID,
		Scope:     code.Scope,
		ExpiresAt: time.Now().Add(time.Duration(uc.config.RefreshTokenTTLDur) * time.Second),
		CreatedAt: time.Now(),
	})

	resp := &TokenResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   int(uc.config.AccessTokenTTLDur),
		Scope:       code.Scope,
	}

	if includesScope(code.Scope, "openid") {
		resp.IDToken = accessToken
		resp.RefreshToken = refreshToken
	}

	return resp, nil
}

func (uc *TokenUseCase) RefreshToken(ctx context.Context, req TokenRequest) (*TokenResponse, error) {
	if req.GrantType != "refresh_token" {
		return nil, ErrInvalidGrantType
	}

	tokenHash := uc.jwtService.HashToken(req.RefreshToken)
	refreshToken, err := uc.tokenRepo.GetRefreshToken(ctx, tokenHash)
	if err != nil || refreshToken == nil {
		return nil, ErrInvalidToken
	}

	if refreshToken.RevokedAt != nil || time.Now().After(refreshToken.ExpiresAt) {
		return nil, ErrInvalidToken
	}

	user, err := uc.userRepo.GetByID(ctx, refreshToken.UserID)
	if err != nil || user == nil {
		return nil, ErrInvalidUser
	}

	accessToken, err := uc.jwtService.GenerateAccessToken(
		user.ID.String(),
		refreshToken.ClientID,
		refreshToken.Scope,
		user.Email,
		user.Name,
		user.Picture,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	newRefreshToken, err := uc.jwtService.GenerateRefreshToken(user.ID.String(), refreshToken.ClientID, refreshToken.Scope)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	accessTokenHash := uc.jwtService.HashToken(accessToken)
	newRefreshTokenHash := uc.jwtService.HashToken(newRefreshToken)

	uc.tokenRepo.CreateAccessToken(ctx, &entity.AccessToken{
		ID:        uuid.New(),
		TokenHash: accessTokenHash,
		ClientID:  refreshToken.ClientID,
		UserID:    user.ID,
		Scope:     refreshToken.Scope,
		ExpiresAt: time.Now().Add(time.Duration(uc.config.AccessTokenTTLDur) * time.Second),
		CreatedAt: time.Now(),
	})

	if uc.config.RefreshTokenRotation {
		uc.tokenRepo.RevokeRefreshToken(ctx, tokenHash)
	}

	uc.tokenRepo.CreateRefreshToken(ctx, &entity.RefreshToken{
		ID:        uuid.New(),
		TokenHash: newRefreshTokenHash,
		ClientID:  refreshToken.ClientID,
		UserID:    user.ID,
		Scope:     refreshToken.Scope,
		ExpiresAt: time.Now().Add(time.Duration(uc.config.RefreshTokenTTLDur) * time.Second),
		CreatedAt: time.Now(),
	})

	resp := &TokenResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   int(uc.config.AccessTokenTTLDur),
		Scope:       refreshToken.Scope,
	}

	if includesScope(refreshToken.Scope, "openid") {
		resp.IDToken = accessToken
		resp.RefreshToken = newRefreshToken
	}

	return resp, nil
}

func (uc *TokenUseCase) RevokeToken(ctx context.Context, token string, tokenTypeHint string) error {
	if tokenTypeHint == "refresh_token" || tokenTypeHint == "" {
		tokenHash := uc.jwtService.HashToken(token)
		uc.tokenRepo.RevokeRefreshToken(ctx, tokenHash)
	}
	return nil
}

func (uc *TokenUseCase) IntrospectToken(ctx context.Context, token string) (*entity.AccessToken, error) {
	_, err := uc.jwtService.ValidateToken(token)
	if err != nil {
		return nil, ErrInvalidToken
	}

	tokenHash := uc.jwtService.HashToken(token)
	return uc.tokenRepo.GetAccessToken(ctx, tokenHash)
}

func includesScope(scope, target string) bool {
	for _, s := range splitScope(scope) {
		if s == target {
			return true
		}
	}
	return false
}

func splitScope(scope string) []string {
	if scope == "" {
		return nil
	}
	var result []string
	for _, s := range split(scope, ' ') {
		if s != "" {
			result = append(result, s)
		}
	}
	return result
}

func split(s string, sep byte) []string {
	result := []string{}
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == sep {
			result = append(result, s[start:i])
			start = i + 1
		}
	}
	result = append(result, s[start:])
	return result
}
