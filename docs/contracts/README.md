# Interface Contract

This directory contains the implemented phase 1 demo contract and planned phase 2 additions.

Canonical contract:

- `openapi.yaml`
- `phase-2-changes.md` lists contract changes and implementation status.

Rules for all AI workers and developers:

1. Backend, admin-web, and miniapp work from this contract first.
2. Do not change request fields, response fields, route paths, or enum values only in code.
3. Any API change must update `openapi.yaml` in the same change.
4. Phase 1 uses mock external services only: mock WeChat, mock SMS code `666666`, mock payment, mock notification logs, and local JSON storage. Operations marked `x-implementation-status: planned` require implementation before use.
5. Backend is the only writer of demo data. Frontend clients must call HTTP APIs and must not read or write JSON data files directly.
