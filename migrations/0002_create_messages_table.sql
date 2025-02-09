CREATE TABLE messages (
                          id UUID PRIMARY KEY,
                          sender_id UUID NOT NULL,
                          receiver_id UUID NOT NULL,
                          content TEXT NOT NULL,
                          status VARCHAR(50) DEFAULT 'sent', -- sent, received, read
                          created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
                          FOREIGN KEY (sender_id) REFERENCES users(id),
                          FOREIGN KEY (receiver_id) REFERENCES users(id)
);