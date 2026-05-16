package usecase

import (
	"context"
	"time"

	"github.com/andipiee/go-oidc/internal/domain/entity"
	"github.com/andipiee/go-oidc/internal/domain/repository"
	"github.com/google/uuid"
)

type UserUseCase struct {
	userRepo    repository.UserRepository
	sessionRepo repository.SessionRepository
}

func NewUserUseCase(userRepo repository.UserRepository, sessionRepo repository.SessionRepository) *UserUseCase {
	return &UserUseCase{
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
	}
}

func (uc *UserUseCase) GetUserByID(ctx context.Context, userID string) (*entity.User, error) {
	id, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}
	return uc.userRepo.GetByID(ctx, id)
}

func (uc *UserUseCase) GetUserByEmail(ctx context.Context, email string) (*entity.User, error) {
	return uc.userRepo.GetByEmail(ctx, email)
}

func (uc *UserUseCase) CreateSession(ctx context.Context, userID uuid.UUID) (*entity.UserSession, error) {
	session := &entity.UserSession{
		ID:        uuid.Must(uuid.NewV7()),
		SessionID: uuid.Must(uuid.NewV7()).String(),
		UserID:    userID,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
	}
	if err := uc.sessionRepo.Create(ctx, session); err != nil {
		return nil, err
	}
	return session, nil
}

func (uc *UserUseCase) ValidateSession(ctx context.Context, sessionID string) (*entity.UserSession, error) {
	return uc.sessionRepo.GetBySessionID(ctx, sessionID)
}

func (uc *UserUseCase) DeleteSession(ctx context.Context, sessionID string) error {
	return uc.sessionRepo.DeleteBySessionID(ctx, sessionID)
}

type DeviceUseCase struct {
	clientRepo repository.ClientRepository
	userRepo   repository.UserRepository
	tokenRepo  repository.TokenRepository
	jwtService interface {
		GenerateAccessToken(userID, clientID, scope, email, name, picture string) (string, error)
		GenerateRefreshToken(userID, clientID, scope string) (string, error)
		HashToken(token string) string
	}
	codeTTL int64
}

func NewDeviceUseCase(
	clientRepo repository.ClientRepository,
	userRepo repository.UserRepository,
	tokenRepo repository.TokenRepository,
	jwtService interface {
		GenerateAccessToken(userID, clientID, scope, email, name, picture string) (string, error)
		GenerateRefreshToken(userID, clientID, scope string) (string, error)
		HashToken(token string) string
	},
	codeTTL int64,
) *DeviceUseCase {
	return &DeviceUseCase{
		clientRepo: clientRepo,
		userRepo:   userRepo,
		tokenRepo:  tokenRepo,
		jwtService: jwtService,
		codeTTL:    codeTTL,
	}
}

func (uc *DeviceUseCase) CreateDeviceCode(ctx context.Context, clientID, scope string) (*entity.DeviceCode, error) {
	client, err := uc.clientRepo.GetByClientID(ctx, clientID)
	if err != nil || client == nil {
		return nil, err
	}

	deviceCode := &entity.DeviceCode{
		ID:              uuid.Must(uuid.NewV7()),
		DeviceCode:      generateRandomString(64),
		UserCode:        generateUserCode(),
		ClientID:        clientID,
		Scope:           scope,
		ExpiresAt:       time.Now().Add(time.Duration(uc.codeTTL) * time.Second),
		PollingInterval: 5,
		Verified:        false,
		CreatedAt:       time.Now(),
	}

	if err := uc.tokenRepo.CreateDeviceCode(ctx, deviceCode); err != nil {
		return nil, err
	}

	return deviceCode, nil
}

func (uc *DeviceUseCase) GetDeviceCode(ctx context.Context, deviceCode string) (*entity.DeviceCode, error) {
	return uc.tokenRepo.GetDeviceCode(ctx, deviceCode)
}

func (uc *DeviceUseCase) GetDeviceCodeByUserCode(ctx context.Context, userCode string) (*entity.DeviceCode, error) {
	return uc.tokenRepo.GetDeviceCodeByUserCode(ctx, userCode)
}

func (uc *DeviceUseCase) VerifyDeviceCode(ctx context.Context, userCode string, userID uuid.UUID) error {
	deviceCode, err := uc.tokenRepo.GetDeviceCodeByUserCode(ctx, userCode)
	if err != nil || deviceCode == nil {
		return err
	}

	deviceCode.Verified = true
	deviceCode.UserID = &userID

	return uc.tokenRepo.UpdateDeviceCode(ctx, deviceCode)
}

func generateRandomString(length int) string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[i%len(charset)]
	}
	return string(b)
}

func generateUserCode() string {
	const charset = "BCDFGHJKLMNPQRSTVWXYZ23456789"
	b := make([]byte, 8)
	for i := range b {
		b[i] = charset[i%len(charset)]
	}
	return string(b)
}
