ALTER TABLE messages ADD COLUMN reply_to_message_id UUID;
ALTER TABLE messages ADD FOREIGN KEY (reply_to_message_id) REFERENCES messages(id);