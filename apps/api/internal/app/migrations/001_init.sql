CREATE TABLE IF NOT EXISTS schema_migrations (version INT PRIMARY KEY, applied_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6)) ENGINE=InnoDB;
CREATE TABLE IF NOT EXISTS store_revision (id TINYINT PRIMARY KEY, revision BIGINT NOT NULL) ENGINE=InnoDB;
INSERT IGNORE INTO store_revision (id, revision) VALUES (1, 0);

CREATE TABLE IF NOT EXISTS members (id VARCHAR(64) PRIMARY KEY, openid VARCHAR(128) NOT NULL, phone VARCHAR(32) NULL, payload JSON NOT NULL, UNIQUE KEY uq_member_openid (openid), UNIQUE KEY uq_member_phone (phone)) ENGINE=InnoDB;
CREATE TABLE IF NOT EXISTS staff_users (id VARCHAR(64) PRIMARY KEY, username VARCHAR(64) NOT NULL, password_hash VARCHAR(255) NOT NULL, payload JSON NOT NULL, UNIQUE KEY uq_staff_username (username)) ENGINE=InnoDB;
CREATE TABLE IF NOT EXISTS orders (id VARCHAR(64) PRIMARY KEY, member_id VARCHAR(64) NOT NULL, status VARCHAR(32) NOT NULL, payload JSON NOT NULL, KEY ix_order_member (member_id), KEY ix_order_status (status)) ENGINE=InnoDB;
CREATE TABLE IF NOT EXISTS coupon_templates (id VARCHAR(64) PRIMARY KEY, payload JSON NOT NULL) ENGINE=InnoDB;
CREATE TABLE IF NOT EXISTS coupon_instances (id VARCHAR(64) PRIMARY KEY, member_id VARCHAR(64) NOT NULL, status VARCHAR(32) NOT NULL, payload JSON NOT NULL, KEY ix_coupon_member_status (member_id, status)) ENGINE=InnoDB;
CREATE TABLE IF NOT EXISTS wallet_accounts (member_id VARCHAR(64) PRIMARY KEY, payload JSON NOT NULL) ENGINE=InnoDB;
CREATE TABLE IF NOT EXISTS wallet_ledgers (id VARCHAR(64) PRIMARY KEY, member_id VARCHAR(64) NOT NULL, order_id VARCHAR(64) NULL, payload JSON NOT NULL, KEY ix_wallet_member (member_id), KEY ix_wallet_order (order_id)) ENGINE=InnoDB;
CREATE TABLE IF NOT EXISTS referral_relations (id VARCHAR(64) PRIMARY KEY, referrer_member_id VARCHAR(64) NOT NULL, referred_member_id VARCHAR(64) NOT NULL, payload JSON NOT NULL, KEY ix_referrer (referrer_member_id), UNIQUE KEY uq_referred (referred_member_id)) ENGINE=InnoDB;
CREATE TABLE IF NOT EXISTS referral_rewards (id VARCHAR(64) PRIMARY KEY, order_id VARCHAR(64) NOT NULL, payload JSON NOT NULL, KEY ix_reward_order (order_id)) ENGINE=InnoDB;
CREATE TABLE IF NOT EXISTS audit_logs (id VARCHAR(64) PRIMARY KEY, operator_staff_id VARCHAR(64) NOT NULL, action VARCHAR(80) NOT NULL, created_at VARCHAR(40) NOT NULL, payload JSON NOT NULL, KEY ix_audit_created (created_at), KEY ix_audit_operator (operator_staff_id)) ENGINE=InnoDB;
CREATE TABLE IF NOT EXISTS notifications (id VARCHAR(64) PRIMARY KEY, member_id VARCHAR(64) NOT NULL, payload JSON NOT NULL, KEY ix_notification_member (member_id)) ENGINE=InnoDB;
CREATE TABLE IF NOT EXISTS rule_config (id TINYINT PRIMARY KEY, payload JSON NOT NULL) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS auth_sessions (token_hash CHAR(64) PRIMARY KEY, principal_type VARCHAR(16) NOT NULL, principal_id VARCHAR(64) NOT NULL, expires_at DATETIME(6) NOT NULL, revoked_at DATETIME(6) NULL, created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6), KEY ix_session_principal (principal_type, principal_id), KEY ix_session_expiry (expires_at)) ENGINE=InnoDB;
INSERT IGNORE INTO schema_migrations (version) VALUES (1);
