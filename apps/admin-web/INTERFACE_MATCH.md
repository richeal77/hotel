# Admin P1 interface match (2026-11-26)

Source: `docs/contracts/openapi.yaml` version `0.4.0-draft` and `docs/contracts/room-benefits-examples.json`. All paths below are relative to `/api/v1` and use the existing staff Bearer session. JSON APIs expect `{code,message,data}`; CSV exports return `text/csv`. Writes send a UUID `Idempotency-Key`; versioned updates send `version`. HTTP 409 displays `data.reason` and a refresh action.

| UI entry | Requests wired | Expected data |
| --- | --- | --- |
| 房型 | GET/POST `/admin/room-types`; PATCH `/admin/room-types/{id}`; GET/POST `/admin/room-types/{id}/rates`; PATCH `/admin/room-rates/{id}` | Paginated rooms, `{room}`, rate items / `{rate}` |
| 日库存 | GET `/admin/room-types`; GET/PUT `/admin/room-types/{id}/inventory?from&to` | `{items,version}` with daily quantity, reserved quantity and closed flag |
| 活动 | GET/POST `/admin/campaigns`; GET/PATCH `/admin/campaigns/{id}`; POST `/admin/campaigns/{id}/state`; GET `/admin/campaigns/{id}/versions` | Paginated campaigns, `{campaign}`, paginated versions |
| 订单人工发礼/核销 | GET `/admin/bookings`; GET `/admin/bookings/{id}`; GET `/admin/bookings/{id}/benefit-options`; PATCH `/admin/bookings/{id}/configured-status`; GET `/admin/bookings/export` | Order/status history, eligible gifts/locked deductions, atomic check-in result containing issued/redeemed/skipped benefits, CSV |
| 权益明细 | GET `/admin/benefits`; GET `/admin/benefit-events`; GET `/admin/benefit-events/export` | Paginated grants and events, CSV |
| 操作日志 | GET `/admin/operation-audit-logs` | Paginated audit records with details and request ID |

Every tab entry and refresh issues new GET requests, including log queries. The activity preview explicitly re-GETs the selected campaign. Order redemption is part of the configured check-in transaction; no separate new-benefit redeem request is invented.

**Backend integration status:** OpenAPI marks these new routes `x-implementation-status: planned`. The UI reports the backend response, not simulated success. Existing `/admin/bookings` is available, but configured booking detail, gift options and status depend on the M1 backend. This file records the mapping only; backend and contract are outside this task's `apps/admin-web/` edit boundary.
