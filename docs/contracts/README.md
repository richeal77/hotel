# Interface Contract

This directory is the frozen API contract for phase 1 demo development.

Canonical contract:

- `openapi.yaml`

Rules for all AI workers and developers:

1. Backend, admin-web, and miniapp work from this contract first.
2. Do not change request fields, response fields, route paths, or enum values only in code.
3. Any API change must update `openapi.yaml` in the same change.
4. Phase 1 uses mock external services only: mock WeChat, mock SMS code `666666`, mock payment, mock notification logs, and local JSON storage.
5. Backend is the only writer of demo data. Frontend clients must call HTTP APIs and must not read or write JSON data files directly.
