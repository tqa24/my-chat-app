ALTER TABLE messages ADD COLUMN group_id UUID;
ALTER TABLE messages ADD FOREIGN KEY (group_id) REFERENCES groups(id);