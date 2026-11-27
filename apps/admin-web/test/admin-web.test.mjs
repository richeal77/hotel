import assert from "node:assert/strict";
import { existsSync } from "node:fs";
import { cp, mkdtemp, rm } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import test from "node:test";
import { fileURLToPath } from "node:url";

import { buildAdmin } from "../scripts/build.mjs";
import { adminConfig, proxyTarget } from "../scripts/config.mjs";
import { DEFAULT_API_BASE_URL, allowedTabs, nextBookingStatus, nonJsonApiError, sanitizeApiBaseUrl } from "../src/app.js";
import { createOperations, exportPath, operationLogPath } from "../src/operations.js";

test("sanitizeApiBaseUrl falls back to the local API default and removes trailing slashes", () => {
  assert.equal(sanitizeApiBaseUrl(""), DEFAULT_API_BASE_URL);
  assert.equal(sanitizeApiBaseUrl("  http://localhost:8080/api/v1///  "), DEFAULT_API_BASE_URL);
});

test("plain text 404 identifies the unavailable API route", () => {
  const error = nonJsonApiError("/admin/room-types?page=1", 404);
  assert.equal(error.status, 404);
  assert.match(error.message, /\/admin\/room-types\?page=1/);
  assert.match(error.message, /重启 API/);
});

test("nextBookingStatus supports the P0 front desk flow only", () => {
  assert.deepEqual(nextBookingStatus("pending_confirm"), { status: "confirmed", label: "确认订单" });
  assert.deepEqual(nextBookingStatus("confirmed"), { status: "checked_in", label: "标记入住" });
  assert.deepEqual(nextBookingStatus("checked_in"), { status: "checked_out", label: "标记离店" });
  assert.equal(nextBookingStatus("checked_out"), null);
  assert.equal(nextBookingStatus("cancelled"), null);
});

test("new business entries appear for every authenticated staff role", () => {
  const entries = ["rooms", "inventory", "campaigns", "configuredOrders", "benefits", "operationLogs"];
  for (const role of ["boss", "manager", "frontdesk", "finance"]) {
    assert.deepEqual(allowedTabs(role, "production").slice(-6), entries);
  }
  assert.deepEqual(allowedTabs("unknown", "production"), []);
  assert.equal(allowedTabs("boss", "local-demo").includes("demo"), true);
});

test("operation log query uses planned filters, paging, and fetches again on reentry", async () => {
  const paths = [];
  const operations = createOperations({
    api: async (path) => { paths.push(path); return { items: [], page: 1, total: 0 }; },
    notify: () => {}, render: () => {}
  });
  assert.equal(operationLogPath({ username: "前台 A", outcome: "rejected" }, 2),
    "/admin/operation-audit-logs?username=%E5%89%8D%E5%8F%B0+A&outcome=rejected&page=2&page_size=20");
  await operations.view("operationLogs");
  await operations.view("operationLogs");
  assert.deepEqual(paths, [operationLogPath(), operationLogPath()]);
});

test("409 conflict shows server reason and a refresh control", async () => {
  const operations = createOperations({
    api: async () => { throw Object.assign(new Error("版本冲突"), { status: 409, reason: "STALE_VERSION" }); },
    notify: () => {}, render: () => {}
  });
  await operations.view("operationLogs");
  assert.match(operations.html("operationLogs"), /STALE_VERSION/);
  assert.match(operations.html("operationLogs"), /data-op="refresh"/);
});

test("CSV export paths match the contract and require bounded dates", () => {
  assert.equal(exportPath("bookings", { from: "2026-12-01", to: "2026-12-03", status: "booked", order_id: "ord 1" }),
    "/admin/bookings/export?from=2026-12-01&to=2026-12-03&status=booked&order_id=ord+1");
  assert.equal(exportPath("benefit-events", { from: "2026-12-01", to: "2026-12-03" }),
    "/admin/benefit-events/export?from=2026-12-01&to=2026-12-03");
  assert.throws(() => exportPath("bookings", { from: "2026-12-03", to: "2026-12-01" }), /有效/);
  assert.throws(() => exportPath("bookings", { from: "2026-12-01", to: "2027-02-01" }), /31 天/);
});

test("benefit list state filter does not leak to event query", async () => {
  const paths = [];
  const operations = createOperations({
    api: async (path) => { paths.push(path); return { items: [], page: 1, total: 0 }; },
    notify: () => {}, render: () => {}
  });
  await operations.view("benefits");
  assert.equal(paths.length, 2);
  assert.match(paths[0], /^\/admin\/benefits\?/);
  assert.match(paths[1], /^\/admin\/benefit-events\?/);
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
  const sourceRoot = fileURLToPath(new URL("..", import.meta.url));
  const rootDir = await mkdtemp(join(tmpdir(), "hotel-admin-build-test-"));
  try {
    await cp(join(sourceRoot, "src"), join(rootDir, "src"), { recursive: true });
    const { distDir } = await buildAdmin({ rootDir });
    assert.equal(existsSync(join(distDir, "index.html")), true);
    assert.equal(existsSync(join(distDir, "app.js")), true);
    assert.equal(existsSync(join(distDir, "operations.js")), true);
    assert.equal(existsSync(join(distDir, "styles.css")), true);
    assert.equal(existsSync(join(distDir, "config.js")), true);
  } finally {
    await rm(rootDir, { recursive: true, force: true });
  }
});
