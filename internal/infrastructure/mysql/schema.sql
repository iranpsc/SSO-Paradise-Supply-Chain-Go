-- Canonical schema after version 2. Apply via cmd/migrate, not directly.
CREATE TABLE IF NOT EXISTS users (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  name VARCHAR(255) NOT NULL,
  username VARCHAR(30) NULL UNIQUE,
  email VARCHAR(255) NULL UNIQUE,
  password_hash VARCHAR(255) NULL,
  code VARCHAR(32) UNIQUE,
  referral VARCHAR(32) NOT NULL DEFAULT '',
  email_verified_at DATETIME(6),
  created_at DATETIME(6) NOT NULL,
  updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  mobile VARCHAR(11) CHARACTER SET ascii NULL,
  KEY users_cleanup(email_verified_at,created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
CREATE TABLE IF NOT EXISTS personal_infos (
  user_id BIGINT PRIMARY KEY,
  is_verified TINYINT NOT NULL DEFAULT 0,
  first_name VARCHAR(255) NOT NULL DEFAULT '',
  last_name VARCHAR(255) NOT NULL DEFAULT '',
  is_company BOOLEAN NULL,
  mobile VARCHAR(11) CHARACTER SET ascii NOT NULL DEFAULT '',
  telephone VARCHAR(11) CHARACTER SET ascii NOT NULL DEFAULT '',
  national_code VARCHAR(10) CHARACTER SET ascii NOT NULL DEFAULT '',
  address VARCHAR(255) NOT NULL DEFAULT '',
  company_name VARCHAR(255) NOT NULL DEFAULT '',
  company_address VARCHAR(255) NOT NULL DEFAULT '',
  company_registration_number VARCHAR(32) NOT NULL DEFAULT '',
  company_national_number VARCHAR(32) NOT NULL DEFAULT '',
  company_tax_number VARCHAR(32) NOT NULL DEFAULT '',
  company_executive_name VARCHAR(255) NOT NULL DEFAULT '',
  verification_messages TEXT NOT NULL DEFAULT (''),
  CONSTRAINT fk_personal_infos_user_id FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
CREATE TABLE IF NOT EXISTS sessions (
  token_hash CHAR(64) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
  user_id BIGINT NOT NULL,
  expires_at DATETIME(6) NOT NULL,
  KEY sessions_user (user_id),
  KEY sessions_expiry(expires_at),
  CONSTRAINT fk_sessions_user_id FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
CREATE TABLE IF NOT EXISTS actions (
  user_id BIGINT NOT NULL,
  kind VARCHAR(6) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  token_hash CHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  email VARCHAR(255) NOT NULL,
  expires_at DATETIME(6) NOT NULL,
  PRIMARY KEY(user_id, kind),
  CONSTRAINT chk_kind CHECK(kind IN ('verify','reset')),
  KEY actions_expiry(expires_at),
  KEY actions_reset_lookup(email,kind,token_hash),
  CONSTRAINT fk_actions_user_id FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
CREATE TABLE IF NOT EXISTS code_sequence (id INT PRIMARY KEY, value BIGINT NOT NULL, CONSTRAINT chk_id CHECK(id=1)) ENGINE=InnoDB;
INSERT IGNORE INTO code_sequence(id,value) VALUES(1,2000000);