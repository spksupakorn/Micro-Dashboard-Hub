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