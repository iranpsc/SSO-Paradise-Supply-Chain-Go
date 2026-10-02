CREATE TABLE IF NOT EXISTS oauth_clients (
 id BIGINT AUTO_INCREMENT PRIMARY KEY,
 name VARCHAR(255) NOT NULL,
 purpose VARCHAR(16) CHARACTER SET ascii COLLATE ascii_bin NULL UNIQUE,
 secret_hash VARCHAR(255) NULL,
 revoked BOOLEAN NOT NULL DEFAULT 0,
 created_at DATETIME(6) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE TABLE IF NOT EXISTS oauth_client_redirects (
 client_id BIGINT NOT NULL,
 uri_hash CHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 uri VARCHAR(2048) COLLATE utf8mb4_bin NOT NULL,
 PRIMARY KEY(client_id,uri_hash),
 CONSTRAINT fk_oauth_redirect_client FOREIGN KEY(client_id) REFERENCES oauth_clients(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE TABLE IF NOT EXISTS oauth_grants (
 id BIGINT AUTO_INCREMENT PRIMARY KEY,
 user_id BIGINT NOT NULL,
 client_id BIGINT NOT NULL,
 revoked_at DATETIME(6) NULL,
 KEY oauth_grant_user(user_id), KEY oauth_grant_client(client_id),
 CONSTRAINT fk_oauth_grant_user FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE,
 CONSTRAINT fk_oauth_grant_client FOREIGN KEY(client_id) REFERENCES oauth_clients(id) ON DELETE CASCADE
) ENGINE=InnoDB;
CREATE TABLE IF NOT EXISTS oauth_grant_scopes (
 grant_id BIGINT NOT NULL,
 scope VARCHAR(32) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 PRIMARY KEY(grant_id,scope),
 CONSTRAINT fk_oauth_scope_grant FOREIGN KEY(grant_id) REFERENCES oauth_grants(id) ON DELETE CASCADE
) ENGINE=InnoDB;
CREATE TABLE IF NOT EXISTS oauth_codes (
 token_hash CHAR(64) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
 grant_id BIGINT NOT NULL,
 redirect_uri VARCHAR(2048) COLLATE utf8mb4_bin NOT NULL,
 challenge CHAR(43) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 expires_at DATETIME(6) NOT NULL,
 consumed_at DATETIME(6) NULL,
 KEY oauth_codes_grant(grant_id), KEY oauth_codes_expiry(expires_at),
 CONSTRAINT fk_oauth_code_grant FOREIGN KEY(grant_id) REFERENCES oauth_grants(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE TABLE IF NOT EXISTS oauth_tokens (
 access_hash CHAR(64) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
 refresh_hash CHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL UNIQUE,
 grant_id BIGINT NOT NULL,
 access_expires_at DATETIME(6) NOT NULL,
 refresh_expires_at DATETIME(6) NOT NULL,
 consumed_at DATETIME(6) NULL,
 KEY oauth_tokens_grant(grant_id), KEY oauth_tokens_expiry(refresh_expires_at),
 CONSTRAINT fk_oauth_token_grant FOREIGN KEY(grant_id) REFERENCES oauth_grants(id) ON DELETE CASCADE
) ENGINE=InnoDB;
