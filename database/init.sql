-- Create users table
CREATE TABLE
    IF NOT EXISTS users (
        id SERIAL PRIMARY KEY,
        email VARCHAR(255) UNIQUE NOT NULL,
        password_hash VARCHAR(255) NOT NULL,
        full_name VARCHAR(255) NOT NULL,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );

-- Create index on email for faster lookups
CREATE INDEX IF NOT EXISTS idx_users_email ON users (email);

-- Create refresh_tokens table for managing refresh tokens
CREATE TABLE
    IF NOT EXISTS refresh_tokens (
        id SERIAL PRIMARY KEY,
        user_id INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
        token VARCHAR(255) UNIQUE NOT NULL,
        expires_at TIMESTAMP NOT NULL,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        revoked BOOLEAN DEFAULT FALSE
    );

-- Create index on refresh tokens
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_token ON refresh_tokens (token);

CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id ON refresh_tokens (user_id);

-- Create active_sessions table for global logout functionality
CREATE TABLE
    IF NOT EXISTS active_sessions (
        id SERIAL PRIMARY KEY,
        user_id INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
        session_id VARCHAR(255) UNIQUE NOT NULL,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        expires_at TIMESTAMP NOT NULL,
        invalidated BOOLEAN DEFAULT FALSE
    );

-- Create index on active sessions
CREATE INDEX IF NOT EXISTS idx_active_sessions_session_id ON active_sessions (session_id);

CREATE INDEX IF NOT EXISTS idx_active_sessions_user_id ON active_sessions (user_id);

-- Insert test users (password: password123)
-- Valid bcrypt hash generated with cost 10
INSERT INTO
    users (email, password_hash, full_name)
VALUES
    (
        'test@example.com',
        '$2a$10$nDFlBfQVF3C4wk7bOxJlfugwXv.ExjlwPNO4s8nmQTeUt9q6A6OdO',
        'Test User'
    ),
    (
        'admin@example.com',
        '$2a$10$nDFlBfQVF3C4wk7bOxJlfugwXv.ExjlwPNO4s8nmQTeUt9q6A6OdO',
        'Admin User'
    ),
    (
        'demo@example.com',
        '$2a$10$nDFlBfQVF3C4wk7bOxJlfugwXv.ExjlwPNO4s8nmQTeUt9q6A6OdO',
        'Demo User'
    ) ON CONFLICT (email) DO NOTHING;