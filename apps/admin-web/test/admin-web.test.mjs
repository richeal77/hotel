import assert from "node:assert/strict";
import { existsSync } from "node:fs";
import { rm } from "node:fs/promises";
import { join } from "node:path";
import test from "node:test";
import { fileURLToPath } from "node:url";

import { buildAdmin } from "../scripts/build.mjs";
import { DEFAULT_API_BASE_URL, nextBookingStatus, sanitizeApiBaseUrl } from "../src/app.js";

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

test("buildAdmin writes a static dist bundle", async () => {
  const rootDir = new URL("..", import.meta.url);
  const rootPath = fileURLToPath(rootDir);
  const distDir = join(rootPath, "dist");
  await rm(distDir, { recursive: true, force: true });

  await buildAdmin({ rootDir: rootPath });

  assert.equal(existsSync(join(distDir, "index.html")), true);
  assert.equal(existsSync(join(distDir, "app.js")), true);
  assert.equal(existsSync(join(distDir, "styles.css")), true);
});
