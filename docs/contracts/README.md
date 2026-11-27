# Interface Contract

This directory contains the implemented demo/live-auth contract and the room-pricing/benefits contract. `0.4.0-draft` is frozen for M1 implementation by the checksums in `room-benefits-handoff.md`, but its new operations are not deployed or approved for production.

Canonical contract:

- `openapi.yaml`
- `phase-2-changes.md` lists contract changes and implementation status.
- `room-benefits-examples.json` is the shared backend, admin-web and miniapp fixture set for `0.4.0-draft`.
- `room-benefits-decisions.md` records D1-D7 store answers and acceptance rules.
- `room-benefits-handoff.md` records release gates, switches and rollback prerequisites.

Rules for all AI workers and developers:

1. Backend, admin-web, and miniapp work from this contract first.
2. Do not change request fields, response fields, route paths, or enum values only in code.
3. Any API change must update `openapi.yaml` in the same change.
4. Phase 1 uses mock external services only: mock WeChat, mock SMS code `666666`, mock payment, mock notification logs, and local JSON storage. `x-planned-environments` and `x-implementation-status` identify unavailable operation variants.
5. Backend is the only writer of demo data. Frontend clients must call HTTP APIs and must not read or write JSON data files directly.
