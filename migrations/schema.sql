-- Migrations Schema Initialization: PostgreSQL Auth Sessions
-- Designed in compliance with ADR-001 transient state management rules

CREATE TABLE IF NOT EXISTS auth_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    state VARCHAR(255) NOT NULL UNIQUE,
    code_verifier VARCHAR(128) NOT NULL,
    redirect_uri TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL
);

-- Optimized index for background scheduler batch purge execution
CREATE INDEX IF NOT EXISTS idx_auth_sessions_expires_at ON auth_sessions(expires_at);

-- Migrations Schema Initialization: PostgreSQL Secure Enclave Public Keys
-- Persists public device keys tied to users for non-repudiation verification
CREATE TABLE IF NOT EXISTS device_enclave_keys (
    id VARCHAR(255) PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL,
    public_key_pem TEXT NOT NULL,
    algorithm VARCHAR(64) NOT NULL,
    registered_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL
);

-- Index user IDs to optimize device public key resolution queries
CREATE INDEX IF NOT EXISTS idx_device_enclave_keys_user_id ON device_enclave_keys(user_id);
