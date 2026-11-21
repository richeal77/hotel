# Gaobeidian Shangkeyou Miniapp Demo

Phase 1 remains a local JSON demo. Phase 2 MySQL and WeChat authentication setup is documented in `docs/architecture/mysql-phase2.md`.

## Repository Root

Use this `hotel/` directory as the code repository root.

## Project Layout

```text
apps/
  api/        # Go/Gin backend, mock services, local JSON storage
  admin-web/  # Node-served static admin console
  miniapp/    # WeChat native miniapp user client
docs/
  contracts/  # Phase 1 API and phase 2 planned changes
```

## API Contract

The canonical contract is `docs/contracts/openapi.yaml`; see `docs/contracts/phase-2-changes.md` for the next phase.

All AI workers and developers must follow it:

1. Backend implements these routes and schemas.
2. Admin-web and miniapp call these routes and schemas.
3. Any API change must update `docs/contracts/openapi.yaml` first.
4. Phase 1 never calls real WeChat, SMS, payment, notification, MySQL, or PMS services.

## Local Defaults

- API base URL: `http://localhost:8080/api/v1`
- Phone preview base URL: `http://{computer-lan-ip}:8080/api/v1`
- Mock SMS code: `666666`
- Staff demo password: `666666`

## Workspace Setup

Use Node.js 22 or newer and pnpm from the repository root:

```bash
pnpm install
pnpm run check
```

The API, admin web and miniapp code is present. Without `APP_ENV`, the API uses local JSON and mock routes. With `APP_ENV=integration` or `production`, it uses MySQL and real WeChat authentication. See `docs/architecture/mysql-phase2.md` for migrations, staff provisioning, and environment settings.

## Development Commands

Backend API:

```bash
pnpm run dev:api
```

Equivalent direct command:

```bash
cd apps/api
go run ./cmd/server
```

Backend tests and demo reset:

```bash
pnpm run test:api
pnpm run test:admin
pnpm run test:miniapp
pnpm run seed:reset
```

Admin web:

```bash
pnpm run dev:admin
```

Equivalent direct command:

```bash
cd apps/admin-web
pnpm install
pnpm dev
```

Miniapp:

```bash
pnpm run install:miniapp
pnpm run dev:miniapp
```

For local preview, open `apps/miniapp` in WeChat Developer Tools and point the API base URL at either `http://localhost:8080/api/v1` or `http://{computer-lan-ip}:8080/api/v1` for phone testing.
