CREATE TABLE IF NOT EXISTS legacy_remember_tokens (
 user_id BIGINT PRIMARY KEY,
 token_hash CHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 expires_at DATETIME(6) NOT NULL,
 KEY legacy_remember_expiry(expires_at),
 CONSTRAINT fk_legacy_remember_user FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB;
