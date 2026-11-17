# Phase 2 API Changes

The phase 1 implementation remains the runnable baseline. `openapi.yaml` version 0.2.0 records the following phase 2 contract; operations marked `x-implementation-status: planned` are not implemented by this commit.

| Operation | Change for integration/production | Local demo |
| --- | --- | --- |
| `POST /auth/wx-login` | New. Accepts `wx_login_code` from `wx.login` and optional `scene` / `referrer_member_id`; returns the existing `member_token` and `member` shape. The server exchanges the code with WeChat. | Use `POST /auth/mock-login`. |
| `POST /auth/wx-bind-phone` | New. Requires member bearer token and `phone_code` from `wx.getPhoneNumber`; returns the existing bound `member` shape. | Use `POST /auth/bind-phone`. |
| `POST /auth/send-code` | Retains request fields. Sends a real SMS; response `data` has `expire_seconds` and never `mock_code`. | Returns `mock_code: "666666"` and `expire_seconds`. |
| `POST /auth/bind-phone` | Retains request/response fields. Verifies the issued SMS code; the code is no longer restricted to `666666` by the schema. | Continues to accept `666666`. |
| `POST /admin/auth/login` | Retains request/response fields. Password is provisioned per environment; the schema no longer fixes it to `666666`. | Existing demo accounts still use `666666`. |
| `POST /auth/mock-login`, `GET /admin/demo/state`, `POST /admin/demo/reset` | Disabled outside `local-demo`. | Remain available. |

The existing member, booking, coupon, wallet, referral, and admin resource paths do not change. MySQL replaces the backend JSON repository without a client API change. Real subscription notifications use backend integration behind existing business actions; a client subscription operation needs a separate contract before implementation. Payment and PMS integration are outside this phase 2 contract.

Before enabling a live environment, implement the planned routes, enforce environment gating on demo endpoints, replace mock tokens and demo credentials, configure real SMS/WeChat credentials, and test both live and demo response variants. The current server and clients still implement the local demo only.
