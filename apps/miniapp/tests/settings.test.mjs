import assert from "node:assert/strict";
import { createRequire } from "node:module";
import test from "node:test";

const require = createRequire(import.meta.url);
const { createRuntimeSettings, normalizeApiBaseUrl } = require("../utils/settings.js");

function createMemoryStorage(initial) {
  const state = new Map(Object.entries(initial || {}));

  return {
    get(key) {
      return state.get(key);
    },
    set(key, value) {
      state.set(key, value);
    }
  };
}

test("runtime settings prefer a stored LAN API base URL", () => {
  const settings = createRuntimeSettings({
    storage: createMemoryStorage({
      hotel_api_base_url: "http://192.168.31.20:8080/api/v1/"
    })
  });

  assert.equal(settings.getApiBaseUrl(), "http://192.168.31.20:8080/api/v1");
});

test("invalid API base URLs are rejected with a clear message", () => {
  assert.throws(
    () => normalizeApiBaseUrl("192.168.31.20:8080/api/v1"),
    /必须以 http:\/\/ 或 https:\/\//
  );
});

test("local demo mode is disabled by default and must be enabled explicitly", () => {
  const settings = createRuntimeSettings({
    storage: createMemoryStorage()
  });

  assert.equal(settings.getApiMode(), "http");

  settings.setApiMode("local-demo");
  assert.equal(settings.getApiMode(), "local-demo");
});
