
CREATE TABLE departments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    head_user_id UUID,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);