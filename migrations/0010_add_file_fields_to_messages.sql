-- Add file-related columns to the messages table
ALTER TABLE messages
    ADD COLUMN file_name VARCHAR(255),
ADD COLUMN file_path VARCHAR(255),
ADD COLUMN file_type VARCHAR(100),
ADD COLUMN file_size BIGINT;