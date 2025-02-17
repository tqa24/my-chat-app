CREATE TABLE groups (
                        id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                        name VARCHAR(255) UNIQUE NOT NULL,
                        created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
                        updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create the join table for the many-to-many relationship
CREATE TABLE user_groups (
                             user_id UUID NOT NULL,
                             group_id UUID NOT NULL,
                             PRIMARY KEY (user_id, group_id),
                             FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
                             FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE
);