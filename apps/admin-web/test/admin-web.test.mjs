import assert from "node:assert/strict";
import { existsSync } from "node:fs";
import { rm } from "node:fs/promises";
import { join } from "node:path";
import test from "node:test";
import { fileURLToPath } from "node:url";

import { buildAdmin } from "../scripts/build.mjs";
import { adminConfig, proxyTarget } from "../scripts/config.mjs";
import { DEFAULT_API_BASE_URL, allowedTabs, nextBookingStatus, sanitizeApiBaseUrl } from "../src/app.js";

test("sanitizeApiBaseUrl falls back to the local API default and removes trailing slashes", () => {
  assert.equal(sanitizeApiBaseUrl(""), DEFAULT_API_BASE_URL);
  assert.equal(sanitizeApiBaseUrl("  http://localhost:8080/api/v1///  "), DEFAULT_API_BASE_URL);
});

test("nextBookingStatus supports the P0 front desk flow only", () => {
  assert.deepEqual(nextBookingStatus("pending_confirm"), { status: "confirmed", label: "确认订单" });
  assert.deepEqual(nextBookingStatus("confirmed"), { status: "checked_in", label: "标记入住" });
  assert.deepEqual(nextBookingStatus("checked_in"), { status: "checked_out", label: "标记离店" });
  assert.equal(nextBookingStatus("checked_out"), null);
  assert.equal(nextBookingStatus("cancelled"), null);
});

test("roles expose only the backend-authorized admin views", () => {
  assert.deepEqual(allowedTabs("frontdesk", "integration"), ["orders", "redemption"]);
  assert.deepEqual(allowedTabs("finance", "production"), ["audit"]);
  assert.deepEqual(allowedTabs("boss", "production"), ["dashboard", "orders", "redemption", "audit"]);
  assert.equal(allowedTabs("boss", "local-demo").includes("demo"), true);
});

test("proxy target is fixed to the configured origin and disabled in production", () => {
  assert.equal(proxyTarget("/__admin_proxy/api/v1/admin/bookings", "?page=1", "http://localhost:8080", "integration"), "http://localhost:8080/api/v1/admin/bookings?page=1");
  assert.equal(proxyTarget("/__admin_proxy", "?url=http://example.com", "http://localhost:8080", "integration"), null);
  assert.equal(proxyTarget("/__admin_proxy/api/v1/%2e%2e/admin", "", "http://localhost:8080", "integration"), null);
  assert.equal(proxyTarget("/__admin_proxy/api/v1/admin/bookings", "", "http://localhost:8080", "production"), null);
});

test("production build uses same-origin API and has no development proxy", () => {
  assert.deepEqual(adminConfig({ APP_ENV: "production" }), { appEnv: "production", apiBaseUrl: "/api/v1", proxyEnabled: false });
  assert.throws(() => adminConfig({ APP_ENV: "production", ADMIN_API_BASE_URL: "http://localhost:8080/api/v1" }), /Production ADMIN_API_BASE_URL/);
});

test("buildAdmin writes a static dist bundle", async () => {
  const rootDir = new URL("..", import.meta.url);
  const rootPath = fileURLToPath(rootDir);
  const distDir = join(rootPath, "dist");
  await rm(distDir, { recursive: true, force: true });

  await buildAdmin({ rootDir: rootPath });

  assert.equal(existsSync(join(distDir, "index.html")), true);
  assert.equal(existsSync(join(distDir, "app.js")), true);
  assert.equal(existsSync(join(distDir, "styles.css")), true);
  assert.equal(existsSync(join(distDir, "config.js")), true);
});
