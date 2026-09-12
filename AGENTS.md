# AI Development Instructions

This repository root is `hotel/`.

## Source Of Truth

- API contract: `docs/contracts/openapi.yaml`
- Contract rules: `docs/contracts/README.md`
- Original technical design is in the parent workspace under `../docs/04-技术方案设计/`.

## Directory Ownership

- `apps/api`: backend AI only unless a task explicitly says otherwise.
- `apps/admin-web`: admin-web AI only unless a task explicitly says otherwise.
- `apps/miniapp`: miniapp AI only unless a task explicitly says otherwise.
- `docs/contracts`: contract-owner or coordinator AI only.

## Contract Discipline

Do not change route paths, request fields, response fields, enum values, or auth assumptions only in app code.

If an API change is required:

1. Update `docs/contracts/openapi.yaml`.
2. Update backend implementation.
3. Update any affected frontend callers.
4. Mention the contract change in the task report.

## Phase 1 Constraints

- Mock WeChat login and phone authorization.
- Mock SMS code is always `666666`.
- Mock payment and notifications.
- Use local JSON storage in the backend.
- Do not add MySQL, Redis, real SMS, real payment, real PMS, or production deployment work.
- All data writes must go through backend APIs.

