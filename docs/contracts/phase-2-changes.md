# Phase 2 API Changes

The phase 1 JSON demo remains runnable. `openapi.yaml` version 0.3.0 records the integration contract and its availability by environment.

| Operation | Change for integration/production | Local demo |
| --- | --- | --- |
| `POST /auth/wx-login` | New. Accepts `wx_login_code` from `wx.login` and optional `scene` / `referrer_member_id`; returns the existing `member_token` and `member` shape. The server exchanges the code with WeChat. | Use `POST /auth/mock-login`. |
| `POST /auth/wx-bind-phone` | New. Requires member bearer token and `phone_code` from `wx.getPhoneNumber`; returns the existing bound `member` shape. | Use `POST /auth/bind-phone`. |
| `POST /auth/send-code`, `POST /auth/bind-phone` | Live SMS variant remains reserved in OpenAPI but is disabled until an SMS provider is configured and implemented. | Continues to use `666666`. |
| `POST /auth/logout`, `POST /admin/auth/logout` | New. Revoke the current opaque session. | Demo mock tokens have no logout endpoint. |
| `POST /admin/auth/login` | Retains request/response fields. Password is provisioned per environment; the schema no longer fixes it to `666666`. | Existing demo accounts still use `666666`. |
| `POST /auth/mock-login`, `GET /admin/demo/state`, `POST /admin/demo/reset` | Disabled outside `local-demo`. | Remain available. |

The existing member, booking, coupon, wallet, referral, and admin resource paths do not change. MySQL replaces the backend JSON repository without a client API change. Real subscription notifications use backend integration behind existing business actions; a client subscription operation needs a separate contract before implementation. Payment and PMS integration are outside this phase 2 contract.

The MySQL-backed server implements WeChat login and phone binding, bcrypt staff passwords, revocable sessions, role checks, rate limits, audit, and masked staff responses. Live SMS, subscription notifications, payment and PMS integration remain separate work. WeChat phone authorization requires an eligible Mini Program account and configured credentials; live provider calls and MySQL behavior require a reachable environment for end-to-end verification.
