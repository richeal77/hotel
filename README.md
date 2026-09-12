# Gaobeidian Shangkeyou Miniapp Demo

Phase 1 goal: build a local demo for the hotel membership miniapp. All external services are mocked.

## Repository Root

Use this `hotel/` directory as the code repository root.

## Project Layout

```text
apps/
  api/        # Go/Gin backend, mock services, local JSON storage
  admin-web/  # Vue 3 + Vite + Element Plus admin console
  miniapp/    # WeChat native miniapp user client
docs/
  contracts/  # Frozen API contract
```

## Frozen API Contract

The phase 1 contract is `docs/contracts/openapi.yaml`.

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
