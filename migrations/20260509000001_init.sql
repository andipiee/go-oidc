-- +goose Up

-- Create users table
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255),
    name VARCHAR(255),
    email_verified BOOLEAN DEFAULT FALSE,
    picture VARCHAR(512),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_created_at ON users(created_at DESC);

-- Create clients table
CREATE TABLE IF NOT EXISTS clients (
    id UUID PRIMARY KEY,
    client_id VARCHAR(255) UNIQUE NOT NULL,
    client_secret_hash VARCHAR(255),
    name VARCHAR(255) NOT NULL,
    redirect_uris JSONB NOT NULL DEFAULT '[]',
    grant_types JSONB NOT NULL DEFAULT '["authorization_code"]',
    response_types JSONB NOT NULL DEFAULT '["code"]',
    token_endpoint_auth_method VARCHAR(50) NOT NULL DEFAULT 'client_secret_basic',
    scopes JSONB NOT NULL DEFAULT '["openid","profile","email"]',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_clients_created_at ON clients(created_at DESC);

-- Create authorization_codes table
CREATE TABLE IF NOT EXISTS authorization_codes (
    id UUID PRIMARY KEY,
    code VARCHAR(255) UNIQUE NOT NULL,
    client_id VARCHAR(255) NOT NULL,
    user_id UUID NOT NULL,
    redirect_uri VARCHAR(512) NOT NULL,
    scope TEXT,
    nonce VARCHAR(255),
    code_challenge VARCHAR(255),
    code_challenge_method VARCHAR(10),
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_authorization_codes_expires_at ON authorization_codes(expires_at);

-- Create access_tokens table
CREATE TABLE IF NOT EXISTS access_tokens (
    id UUID PRIMARY KEY,
    token_hash VARCHAR(255) UNIQUE NOT NULL,
    client_id VARCHAR(255) NOT NULL,
    user_id UUID NOT NULL,
    scope TEXT,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_access_tokens_expires_at ON access_tokens(expires_at);

-- Create refresh_tokens table
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id UUID PRIMARY KEY,
    token_hash VARCHAR(255) UNIQUE NOT NULL,
    client_id VARCHAR(255) NOT NULL,
    user_id UUID NOT NULL,
    scope TEXT,
    expires_at TIMESTAMP NOT NULL,
    revoked_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_refresh_tokens_expires_at ON refresh_tokens(expires_at);

-- Create user_sessions table
CREATE TABLE IF NOT EXISTS user_sessions (
    id UUID PRIMARY KEY,
    session_id VARCHAR(255) UNIQUE NOT NULL,
    user_id UUID NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_user_sessions_expires_at ON user_sessions(expires_at);

-- Create external_accounts table
CREATE TABLE IF NOT EXISTS external_accounts (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    provider VARCHAR(50) NOT NULL,
    provider_user_id VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(provider, provider_user_id)
);

CREATE INDEX idx_external_accounts_user_id ON external_accounts(user_id);

-- Create device_codes table
CREATE TABLE IF NOT EXISTS device_codes (
    id UUID PRIMARY KEY,
    device_code VARCHAR(255) UNIQUE NOT NULL,
    user_code VARCHAR(255) UNIQUE NOT NULL,
    client_id VARCHAR(255) NOT NULL,
    scope TEXT,
    expires_at TIMESTAMP NOT NULL,
    polling_interval INTEGER NOT NULL DEFAULT 5,
    verified BOOLEAN NOT NULL DEFAULT FALSE,
    user_id UUID,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_device_codes_expires_at ON device_codes(expires_at);

-- Insert default admin client
INSERT INTO clients (id, client_id, client_secret_hash, name, redirect_uris, grant_types, token_endpoint_auth_method, scopes, created_at, updated_at)
VALUES (
    'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11',
    'default-client',
    '$2a$10$USnAIkkd3NRD2G09.s0UuukINRHCJr6VmHmjng8oZLntCO3bzD29q', -- password: default
    'Default Client',
    '["http://localhost:3000/callback","http://localhost:8080/callback"]',
    '["authorization_code","refresh_token","urn:ietf:params:oauth:grant-type:device_code"]',
    'client_secret_post',
    '["openid","profile","email"]',
    NOW(),
    NOW()
) ON CONFLICT (client_id) DO NOTHING;

-- Insert demo user
INSERT INTO users (id, email, password_hash, name, email_verified, created_at, updated_at)
VALUES (
    'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12',
    'demo@example.com',
    '$2a$10$USnAIkkd3NRD2G09.s0UuukINRHCJr6VmHmjng8oZLntCO3bzD29q', -- password: default
    'Demo User',
    TRUE,
    NOW(),
    NOW()
) ON CONFLICT (email) DO NOTHING;

-- +goose Down
DROP TABLE IF EXISTS device_codes;
DROP TABLE IF EXISTS external_accounts;
DROP TABLE IF EXISTS user_sessions;
DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS access_tokens;
DROP TABLE IF EXISTS authorization_codes;
DROP TABLE IF EXISTS clients;
DROP TABLE IF EXISTS users;
