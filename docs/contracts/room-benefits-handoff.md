# Room Pricing and Benefits Handoff

Status: **M0 frozen for M1 implementation on 2026-11-26; not approved for production**. The frozen identifier is `0.4.0-draft` plus the two SHA-256 hashes below; the `-draft` suffix remains to distinguish this development contract from a production release. Backend, admin-web and miniapp use the same `openapi.yaml` and `room-benefits-examples.json`. Any change to either frozen file requires a new contract version and renewed three-team acknowledgement. No production rollout is authorized by this document.

| Frozen artifact | SHA-256 |
| --- | --- |
| `docs/contracts/openapi.yaml` | `ddcfb4aca96059ff725a47facf4acb4defdceff43ab84de50b61f3cdc67912b5` |
| `docs/contracts/room-benefits-examples.json` | `2d9f710b71e22f663e8abeb7ad86ad96dd793e5dae631c08cafe8f5779b7a8a0` |

| M0 recipient task | Same-version response | Result |
| --- | --- | --- |
| `3.2 你是后端 AI` | Confirmed `0.4.0-draft` and `room-benefits-examples.json`; reran 60-path/12-example check | Can develop M1, no blocker |
| `3.3 你是后台/管理端 AI。` | Confirmed `0.4.0-draft` and `room-benefits-examples.json`; reran 60-path/12-example check | Can develop M1, no blocker |
| `3.4 你是小程序 AI` | Confirmed `0.4.0-draft` and `room-benefits-examples.json` | Can develop M1, no blocker |

M0 validation run (2026-11-26, repository root): `ruby docs/contracts/validate-room-benefits.rb` -> `0.4.0-draft: 60 paths, references and 12 examples valid`. The validator checks local references, path parameters, every planned write's 400/401/403/409 declarations, and all shared example shapes. It is a focused contract check, not an end-to-end implementation test. `git diff --check` also passed.

Backend feasibility feedback (2026-11-26): backend reviewed the candidate contract and examples and reported the mixed-list candidate, status filters, D7 isolated 503, internal no-show and empty ledger feasible. It requested explicit closure for legacy `/bookings` and `/members/join` writes. The current revision closes those writes to local-demo only and removes the mixed-list production assumption. After that revision, backend confirmed `0.4.0-draft` and `room-benefits-examples.json` are usable for M1 with no blocker; backend reran validation (60 paths, 12 examples). Admin-web and miniapp confirmation remain pending.

| Recipient | Use the same contract/examples for | Acceptance handoff |
| --- | --- | --- |
| Backend AI | Store/API, migrations, transactions, idempotency, audit, capability switches | Return commit, migration/test results, schema deviations and JSON/MySQL parity |
| Admin-web AI | Room/rate/inventory, campaigns, check-in, exports, operation logs | Return build, screenshot/flow evidence, conflict handling and contract mismatches |
| Miniapp AI | Capabilities, room list/quote/submit, gift claim, benefit list | Return API mapping, repeat-submit and expired quote tests, WeChat preview evidence |
| Acceptance AI | PRD R1-R3, A1-A6 and D1-D7, including two storage modes | Record environment/build/contract version, evidence, failures and retest owner |
| Release AI | Switches, approval, compatible rollback | Wait for M3 sign-off and production authorization; no deployment yet |

## Release Switch Matrix

| Switch | Default in all environments | When enabled | When disabled after use | Approval |
| --- | --- | --- | --- | --- |
| `CONFIGURED_BOOKING_ENABLED` | off | New quotes and configured bookings | Block new quotes/submissions; keep existing order queries, check-in, cancellation and no-show release | Store owner confirms room/price/capacity; acceptance signs R1-R3 |
| `CONFIGURED_JOIN_ENABLED` | off | New registration and referral-code binding | Block new registration; preserve existing member and entitlement reads | Store owner confirms registration/referral policy; acceptance signs A1-A2 |
| `CAMPAIGN_ISSUE_ENABLED` | off | Claim and new benefit issuance | Block new issuance; keep issued benefit reads, existing holds/redemption and in-flight order processing | Store owner confirms campaign amounts/limits; acceptance signs A3-A6 |

`GET /mini/capabilities` exposes the three booleans as `configured_booking`, `configured_join`, `campaign_issue`. Production switch changes require recorded time, environment, operator, approval scope and before/after values. Never replace live data with demo seed data.

## Release Gate and Rollback

Current code baseline: `98321ed` (recorded 2026-11-26). **This is not a compatible rollback build for new order/benefit data.** Before M4, Release AI must identify and test a build that understands the incremental tables and in-flight orders, then write its exact commit here. Keep migrations in place unless a separately approved data-recovery plan exists. Rollback order: turn off new booking/registration/issuance, reconcile orders and ledgers, process in-flight check-in/cancellation/no-show, then deploy only the tested compatible build.

M0 may be frozen only after OpenAPI validation, example/schema validation, and explicit same-version acknowledgement from backend, admin-web and miniapp. M3 needs isolated MySQL and JSON demo evidence, WeChat preview checks, store-approved real configuration and acceptance sign-off. M4 additionally needs production change approval, backup/restore rehearsal and a compatible rollback commit.

The old write routes `POST /bookings`, `POST /members/join` and `PATCH /admin/bookings/{order_id}/status` are local JSON demo only. Integration and production return 409 `FEATURE_DISABLED` from these routes, including while a new feature switch is off; switches must never fall back to fixed-price bookings or old gift issuance. The shared list routes return legacy orders only in local-demo and configured orders only in integration/production. Demo seed orders are not production history, so there is no production dual-track migration. Only `configured-status` can check in a formal order and trigger gifts/credit. No-show processing is a server-internal scheduled/recoverable job, not a public API. First initialization in empty JSON/MySQL creates rules and an empty new ledger; it does not import demo orders or issue benefits. These are draft implementation boundaries to be confirmed by backend tests before M0 freeze.

For isolated D7 testing only, the candidate audit-failure behavior is 503 `AUDIT_UNAVAILABLE`: roll back successful business writes, refuse reads, and independently alarm; rejected/failed requests are appended separately. The store has **not** approved this for production. Retention, export and miniapp-side logging remain open decisions.

| Milestone | Current status | Blocker |
| --- | --- | --- |
| M0 contract | Frozen for M1; validation and all three acknowledgements recorded | Production release remains separately gated |
| M1 room booking | Independent acceptance complete; **not signed off** | Three-end live flow, isolated MySQL test, delivery snapshots and contract-gap retest; see `docs/room-benefits-m1-independent-acceptance-2026-11-26.md` |
| M2 benefits | Not accepted | Issuance/ledger/referral implementation and tests |
| M3 integrated acceptance | Not scheduled | M1-M2 evidence, store configuration and two storage modes |
| M4 production | Not authorized | M3 sign-off, approvals, backup/restore and compatible rollback commit |

## M1 Backend Delivery Note

Backend reported an M1 implementation on 2026-11-26 under the unchanged frozen OpenAPI/example hashes above. This is a delivery claim, **not independent acceptance**. It covers room types, default/daily inventory, quotes, two-night booking, cancellation/release, no-show recovery, and local API/concurrency tests. MySQL first-initialization tests are written but were not run against an isolated empty database because `API_TEST_MYSQL_DSN` is unavailable. M2 benefit issuance and full admin operation audit are outside this M1 delivery.

On 2026-11-26, the project controller sent the M1 independent acceptance brief to `3.5 你是验收/联调 AI` (task `01a0981f-3da3-7382-b1f0-719afac8ca94`). Assignment is not a pass result. The reviewer must report evidence for the two-night booking/cancel/release flow, repeat cancellation, concurrency, switches, no-show, JSON/MySQL parity and contract deviations before M1 sign-off.

The reviewer completed `docs/room-benefits-m1-independent-acceptance-2026-11-26.md` and **declined M1 sign-off**. Uncached Go tests and a focused race test passed; admin-web 11/11 and miniapp 28/28 tests passed. JSON API tests cover two-night holds/release, duplicate cancellation, last-room concurrency, closure, quote expiry and no-show. The MySQL initialization test skipped without `API_TEST_MYSQL_DSN`; no actual admin-web → miniapp → admin-web integrated flow or WeChat device evidence was available. M2 remains gated on M1 retest, not on local unit tests alone.

Next-version contract clarifications, without editing the frozen M0 artifacts: add explicit 404 `NOT_FOUND` to applicable resource operations; make inventory GET `from` and `to` required while keeping optional filters optional elsewhere; state quote TTL (implementation currently 10 minutes) and the one-year maximum date range. Reconfirm these with all three clients before a new freeze.

Architecture review required before M2/production: migration `002_configured_booking.sql` stores M1 state as an independent JSON row under the existing global `store_revision` transaction lock, instead of the plan's separate room/rate/inventory/order tables. Review migration safety, query scale, consistency and rollback compatibility; either accept this deviation with limits or request a normalized migration. Do not infer approval from passing local tests.
