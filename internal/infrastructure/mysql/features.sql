-- Canonical feature schema after version 2. profile_details is a read-only legacy archive.
CREATE TABLE IF NOT EXISTS profile_details (
  user_id BIGINT PRIMARY KEY,
  payload TEXT NOT NULL,
  CONSTRAINT fk_profile_details_user_id FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
CREATE TABLE IF NOT EXISTS media (
  user_id BIGINT NOT NULL,
  kind VARCHAR(20) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  content_type VARCHAR(32) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  data MEDIUMBLOB NOT NULL,
  PRIMARY KEY(user_id,kind),
  CONSTRAINT fk_media_user_id FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
CREATE TABLE IF NOT EXISTS wallets (
  user_id BIGINT PRIMARY KEY,
  address CHAR(42) CHARACTER SET ascii COLLATE ascii_bin NOT NULL UNIQUE,
  CONSTRAINT fk_wallets_user_id FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
CREATE TABLE IF NOT EXISTS challenges (
  `key` VARCHAR(100) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
  message TEXT NOT NULL,
  expires_at DATETIME(6) NOT NULL,
  KEY challenges_expiry(expires_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
CREATE TABLE IF NOT EXISTS session_attributes (
  token_hash CHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  name VARCHAR(32) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  value TEXT NOT NULL,
  expires_at DATETIME(6) NOT NULL,
  PRIMARY KEY(token_hash,name),
  CONSTRAINT fk_session_attributes_token_hash FOREIGN KEY(token_hash) REFERENCES sessions(token_hash) ON DELETE CASCADE,
  KEY session_attributes_expiry(expires_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
CREATE TABLE IF NOT EXISTS registration_callbacks (
  user_id BIGINT PRIMARY KEY,
  url VARCHAR(2048) NOT NULL,
  expires_at DATETIME(6) NOT NULL,
  CONSTRAINT fk_registration_callbacks_user_id FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE,
  KEY registration_callbacks_expiry(expires_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;