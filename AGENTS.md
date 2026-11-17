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

## Environment Rules

- `local-demo` (phase 1): use mock WeChat login and phone authorization, SMS code `666666`, mock payment and notifications, and backend-owned local JSON storage. Demo-only routes and credentials must remain confined to this environment.
- `integration` (phase 2): real WeChat login and phone authorization, real SMS and subscription notifications, and MySQL may be integrated with test credentials. Never return a verification code in an API response or log a credential. Keep payment and PMS mocked until separately specified.
- `production`: use production credentials and HTTPS, disable mock login, demo reset/state, fixed SMS codes and demo staff credentials. Real payment and PMS require separate design and approval; they are not implied by phase 2.
- In every environment, all data writes go through backend APIs. Clients do not write storage directly.
- Secrets, runtime data, build output and developer-specific configuration stay out of Git. Commit safe examples only.
- Planned phase 2 operations in OpenAPI are contract definitions, not claims of implementation. Implement server and callers together when enabling them.
