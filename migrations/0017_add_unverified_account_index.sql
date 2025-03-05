-- Add an index to improve performance of queries for unverified accounts with expired OTPs
CREATE INDEX idx_users_verification_status ON users (is_verified, otp_expiry, deleted_at)
WHERE is_verified = false AND deleted_at IS NULL;
