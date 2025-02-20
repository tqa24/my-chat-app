INSERT INTO users (
    id,
    username,
    password,
    email,
    created_at,
    updated_at
) VALUES (
             '00000000-0000-0000-0000-000000000000',
             'AI_Assistant',
             'not_applicable',  -- Since AI won't login traditionally
             'ai@system.internal',
             CURRENT_TIMESTAMP,
             CURRENT_TIMESTAMP
         ) ON CONFLICT (id) DO NOTHING;  -- Prevent duplicate insertion

-- Add an index to optimize queries involving the AI user
CREATE INDEX idx_ai_user ON users (id) WHERE id = '00000000-0000-0000-0000-000000000000';
