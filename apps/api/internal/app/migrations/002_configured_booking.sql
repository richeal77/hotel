CREATE TABLE IF NOT EXISTS configured_booking_store (
  id TINYINT UNSIGNED NOT NULL PRIMARY KEY,
  payload JSON NOT NULL
) ENGINE=InnoDB;

INSERT IGNORE INTO configured_booking_store (id, payload) VALUES (1, JSON_OBJECT());

CREATE TABLE IF NOT EXISTS configured_booking_quotes (
  id VARCHAR(64) NOT NULL PRIMARY KEY,
  expires_at DATETIME(6) NOT NULL,
  payload JSON NOT NULL,
  KEY ix_configured_quote_expiry (expires_at)
) ENGINE=InnoDB;

INSERT IGNORE INTO schema_migrations (version) VALUES (2);
