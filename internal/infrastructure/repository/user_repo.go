package repository

import (
	"context"
	"encoding/json"

	"github.com/andipiee/go-oidc/internal/domain/entity"
	"github.com/andipiee/go-oidc/internal/infrastructure/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type UserRepository struct {
	db *database.PostgresDB
}

func NewUserRepository(db *database.PostgresDB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, user *entity.User) error {
	query := `
		INSERT INTO users (id, email, password_hash, name, email_verified, picture, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.db.Pool().Exec(ctx, query,
		user.ID, user.Email, user.PasswordHash, user.Name,
		user.EmailVerified, user.Picture, user.CreatedAt, user.UpdatedAt,
	)
	return err
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	query := `
		SELECT id, email, password_hash, name, email_verified, COALESCE(picture, ''), created_at, updated_at
		FROM users WHERE id = $1
	`
	var user entity.User
	err := r.db.Pool().QueryRow(ctx, query, id).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.Name,
		&user.EmailVerified, &user.Picture, &user.CreatedAt, &user.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &user, err
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*entity.User, error) {
	query := `
		SELECT id, email, password_hash, name, email_verified, COALESCE(picture, ''), created_at, updated_at
		FROM users WHERE email = $1
	`
	var user entity.User
	err := r.db.Pool().QueryRow(ctx, query, email).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.Name,
		&user.EmailVerified, &user.Picture, &user.CreatedAt, &user.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &user, err
}

func (r *UserRepository) Update(ctx context.Context, user *entity.User) error {
	query := `
		UPDATE users SET email = $2, password_hash = $3, name = $4, email_verified = $5,
		picture = $6, updated_at = $7 WHERE id = $1
	`
	_, err := r.db.Pool().Exec(ctx, query,
		user.ID, user.Email, user.PasswordHash, user.Name,
		user.EmailVerified, user.Picture, user.UpdatedAt,
	)
	return err
}

func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM users WHERE id = $1`
	_, err := r.db.Pool().Exec(ctx, query, id)
	return err
}

func (r *UserRepository) List(ctx context.Context, limit, offset int) ([]*entity.User, error) {
	query := `
		SELECT id, email, password_hash, name, email_verified, COALESCE(picture, ''), created_at, updated_at
		FROM users ORDER BY created_at DESC LIMIT $1 OFFSET $2
	`
	rows, err := r.db.Pool().Query(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*entity.User
	for rows.Next() {
		var user entity.User
		if err := rows.Scan(
			&user.ID, &user.Email, &user.PasswordHash, &user.Name,
			&user.EmailVerified, &user.Picture, &user.CreatedAt, &user.UpdatedAt,
		); err != nil {
			return nil, err
		}
		users = append(users, &user)
	}
	return users, nil
}

func (r *UserRepository) GetByExternalAccount(ctx context.Context, provider, providerUserID string) (*entity.User, error) {
	query := `
		SELECT u.id, u.email, u.password_hash, u.name, u.email_verified, COALESCE(u.picture, ''), u.created_at, u.updated_at
		FROM users u
		JOIN external_accounts ea ON u.id = ea.user_id
		WHERE ea.provider = $1 AND ea.provider_user_id = $2
	`
	var user entity.User
	err := r.db.Pool().QueryRow(ctx, query, provider, providerUserID).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.Name,
		&user.EmailVerified, &user.Picture, &user.CreatedAt, &user.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &user, err
}

func (r *UserRepository) CreateExternalAccount(ctx context.Context, account *entity.ExternalAccount) error {
	query := `
		INSERT INTO external_accounts (id, user_id, provider, provider_user_id, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := r.db.Pool().Exec(ctx, query,
		account.ID, account.UserID, account.Provider, account.ProviderUserID, account.CreatedAt,
	)
	return err
}

type ClientRepository struct {
	db *database.PostgresDB
}

func NewClientRepository(db *database.PostgresDB) *ClientRepository {
	return &ClientRepository{db: db}
}

func (r *ClientRepository) Create(ctx context.Context, client *entity.Client) error {
	redirectURIs, _ := json.Marshal(client.RedirectURIs)
	grantTypes, _ := json.Marshal(client.GrantTypes)
	responseTypes, _ := json.Marshal(client.ResponseTypes)
	scopes, _ := json.Marshal(client.Scopes)

	query := `
		INSERT INTO clients (id, client_id, client_secret_hash, name, redirect_uris, grant_types, 
		response_types, token_endpoint_auth_method, scopes, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	_, err := r.db.Pool().Exec(ctx, query,
		client.ID, client.ClientID, client.ClientSecretHash, client.Name,
		redirectURIs, grantTypes, responseTypes, client.TokenEndpointAuthMethod,
		scopes, client.CreatedAt, client.UpdatedAt,
	)
	return err
}

func (r *ClientRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.Client, error) {
	query := `
		SELECT id, client_id, client_secret_hash, name, redirect_uris, grant_types, 
		response_types, token_endpoint_auth_method, scopes, created_at, updated_at
		FROM clients WHERE id = $1
	`
	var client entity.Client
	var redirectURIs, grantTypes, responseTypes, scopes []byte
	err := r.db.Pool().QueryRow(ctx, query, id).Scan(
		&client.ID, &client.ClientID, &client.ClientSecretHash, &client.Name,
		&redirectURIs, &grantTypes, &responseTypes, &client.TokenEndpointAuthMethod,
		&scopes, &client.CreatedAt, &client.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	json.Unmarshal(redirectURIs, &client.RedirectURIs)
	json.Unmarshal(grantTypes, &client.GrantTypes)
	json.Unmarshal(responseTypes, &client.ResponseTypes)
	json.Unmarshal(scopes, &client.Scopes)
	return &client, err
}

func (r *ClientRepository) GetByClientID(ctx context.Context, clientID string) (*entity.Client, error) {
	query := `
		SELECT id, client_id, client_secret_hash, name, redirect_uris, grant_types, 
		response_types, token_endpoint_auth_method, scopes, created_at, updated_at
		FROM clients WHERE client_id = $1
	`
	var client entity.Client
	var redirectURIs, grantTypes, responseTypes, scopes []byte
	err := r.db.Pool().QueryRow(ctx, query, clientID).Scan(
		&client.ID, &client.ClientID, &client.ClientSecretHash, &client.Name,
		&redirectURIs, &grantTypes, &responseTypes, &client.TokenEndpointAuthMethod,
		&scopes, &client.CreatedAt, &client.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	json.Unmarshal(redirectURIs, &client.RedirectURIs)
	json.Unmarshal(grantTypes, &client.GrantTypes)
	json.Unmarshal(responseTypes, &client.ResponseTypes)
	json.Unmarshal(scopes, &client.Scopes)
	return &client, err
}

func (r *ClientRepository) Update(ctx context.Context, client *entity.Client) error {
	redirectURIs, _ := json.Marshal(client.RedirectURIs)
	grantTypes, _ := json.Marshal(client.GrantTypes)
	responseTypes, _ := json.Marshal(client.ResponseTypes)
	scopes, _ := json.Marshal(client.Scopes)

	query := `
		UPDATE clients SET client_id = $2, client_secret_hash = $3, name = $4, redirect_uris = $5,
		grant_types = $6, response_types = $7, token_endpoint_auth_method = $8, scopes = $9,
		updated_at = $10 WHERE id = $1
	`
	_, err := r.db.Pool().Exec(ctx, query,
		client.ID, client.ClientID, client.ClientSecretHash, client.Name,
		redirectURIs, grantTypes, responseTypes, client.TokenEndpointAuthMethod,
		scopes, client.UpdatedAt,
	)
	return err
}

func (r *ClientRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM clients WHERE id = $1`
	_, err := r.db.Pool().Exec(ctx, query, id)
	return err
}

func (r *ClientRepository) List(ctx context.Context, limit, offset int) ([]*entity.Client, error) {
	query := `
		SELECT id, client_id, client_secret_hash, name, redirect_uris, grant_types, 
		response_types, token_endpoint_auth_method, scopes, created_at, updated_at
		FROM clients ORDER BY created_at DESC LIMIT $1 OFFSET $2
	`
	rows, err := r.db.Pool().Query(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var clients []*entity.Client
	for rows.Next() {
		var client entity.Client
		var redirectURIs, grantTypes, responseTypes, scopes []byte
		if err := rows.Scan(
			&client.ID, &client.ClientID, &client.ClientSecretHash, &client.Name,
			&redirectURIs, &grantTypes, &responseTypes, &client.TokenEndpointAuthMethod,
			&scopes, &client.CreatedAt, &client.UpdatedAt,
		); err != nil {
			return nil, err
		}
		json.Unmarshal(redirectURIs, &client.RedirectURIs)
		json.Unmarshal(grantTypes, &client.GrantTypes)
		json.Unmarshal(responseTypes, &client.ResponseTypes)
		json.Unmarshal(scopes, &client.Scopes)
		clients = append(clients, &client)
	}
	return clients, nil
}
