ALTER TABLE messages
    ADD COLUMN file_checksum VARCHAR(64); -- SHA-256 checksum is 64 characters (hex-encoded)
CREATE INDEX idx_file_checksum ON messages (file_checksum);