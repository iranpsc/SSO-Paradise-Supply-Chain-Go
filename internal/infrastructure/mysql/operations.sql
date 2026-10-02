CREATE TABLE IF NOT EXISTS mail_outbox (
 id BIGINT AUTO_INCREMENT PRIMARY KEY,
 recipient VARCHAR(255) NOT NULL,
 subject VARCHAR(255) NOT NULL,
 text_body MEDIUMTEXT NOT NULL,
 html_body MEDIUMTEXT NOT NULL,
 state VARCHAR(10) CHARACTER SET ascii COLLATE ascii_bin NOT NULL DEFAULT 'pending',
 attempts SMALLINT UNSIGNED NOT NULL DEFAULT 0,
 available_at DATETIME(6) NOT NULL,
 created_at DATETIME(6) NOT NULL,
 sent_at DATETIME(6) NULL,
 KEY outbox_due(state,available_at,id),
 CONSTRAINT chk_outbox_state CHECK(state IN ('pending','processing','sent','dead'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE TABLE IF NOT EXISTS rate_limits (
 peer_hash CHAR(64) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
 window_start DATETIME(6) NOT NULL,
 blocked_until DATETIME(6) NULL,
 request_count SMALLINT UNSIGNED NOT NULL DEFAULT 0,
 strikes TINYINT UNSIGNED NOT NULL DEFAULT 0
) ENGINE=InnoDB;
CREATE TABLE IF NOT EXISTS legacy_password_resets (
 user_id BIGINT PRIMARY KEY,
 token_hash VARCHAR(255) NOT NULL,
 expires_at DATETIME(6) NOT NULL,
 KEY legacy_resets_expiry(expires_at),
 CONSTRAINT fk_legacy_reset_user FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
