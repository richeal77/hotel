# Phase 2 MySQL and Authentication

## Storage and migration

`apps/api/internal/app/migrations/001_init.sql` creates the InnoDB tables. Business tables are `members`, `staff_users`, `orders`, `coupon_templates`, `coupon_instances`, `wallet_accounts`, `wallet_ledgers`, `referral_relations`, `referral_rewards`, `audit_logs`, `notifications`, and `rule_config`. Each entity table stores its stable ID, indexed lookup/status columns, and a JSON payload matching the phase 1 domain model. `store_revision` serializes writes; `auth_sessions` stores only SHA-256 token hashes, principal type/ID, expiry, and revocation time. `schema_migrations` records schema version 1.

This is a compatibility repository for the existing `Repository.View/Update` API. Reads build a consistent in-memory snapshot. Writes lock `store_revision` with `SELECT ... FOR UPDATE`, run the business callback, write only changed rows, update the revision, and commit. A callback or SQL error rolls the transaction back. The global write lock keeps ID allocation and cross-entity state changes safe across API processes; it limits write throughput and should be replaced with entity-level repositories when the workload grows.

The following operations are each one database transaction: member join and gift coupon issuance; booking status change, checkout cashback, referral reward and audit; coupon redemption and audit; wallet adjustment, balance/ledger and audit; rule/member status changes and audit. Staff provisioning, password rotation, its audit event, and revocation of existing sessions also commit in one transaction. WeChat code exchange occurs before the database transaction. Session issuance occurs after member creation; if it fails, retrying the same OpenID reuses that member. Staff sessions are issued after bcrypt verification, and login/logout events are audited.

## Commands

Create an empty MySQL 8 database with `utf8mb4`, then provide `MYSQL_DSN` through your shell or a private environment file. The repository does not read `.env` automatically. The DSN must point to the existing database and use TLS for a remote host.

```bash
cd apps/api
MYSQL_DSN='...' go run ./cmd/migrate
MYSQL_DSN='...' go run ./cmd/migrate --from-json ./data
STAFF_BOOTSTRAP_PASSWORD='...' MYSQL_DSN='...' go run ./cmd/staff --username boss --name '老板' --role boss
APP_ENV=integration MYSQL_DSN='...' WECHAT_APP_ID='...' WECHAT_APP_SECRET='...' go run ./cmd/server
```

Run `--from-json` only once against an empty business database after backing up the source JSON and target database. It refuses to overwrite any existing business rows. The import omits demo staff accounts and masks historical audit snapshots; provision staff separately with a password of at least 12 characters containing uppercase, lowercase, a digit and a special character. Reprovisioning a staff account rotates its password and revokes its sessions. Running `cmd/migrate` again is safe for the current schema, including after a partially completed DDL run.

For the admin development server in local-demo or integration, set `ADMIN_API_ORIGIN` to the exact API origin. Its proxy forwards only `/api/v1/` paths to that fixed origin. Production builds serve static files and call the same-origin `/api/v1` through HTTPS; the development proxy is disabled. See `apps/admin-web/README.md` for build and deployment settings.

## Authentication and privacy

The Mini Program selects `真实微信接口` in its API settings. It sends `wx.login` code to `/auth/wx-login`, then a user-triggered `getPhoneNumber` code to `/auth/wx-bind-phone`. The server calls WeChat with its private AppSecret, stores only the resulting OpenID and phone, and returns a random opaque session token. Member sessions expire after seven days; staff sessions after 12 hours. `/auth/logout` and `/admin/auth/logout` revoke the current token. Mock and demo reset routes are absent in integration and production.

Staff routes enforce the documented role matrix. Staff API responses mask phone numbers as `138****8001`; member self-service responses retain the member's full phone. New audit snapshots are masked before storage, and imported historical snapshots are masked during import. Login and code endpoints are limited per process and client IP. At multi-instance scale, use a shared limiter before exposing the service publicly.

Live SMS delivery and verification remain disabled until a provider is implemented. WeChat phone authorization requires a Mini Program account entitled to that capability. Real payment, PMS, and subscription notification delivery are outside this change.
