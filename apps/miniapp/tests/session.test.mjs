import assert from "node:assert/strict";
import { createRequire } from "node:module";
import test from "node:test";

const require = createRequire(import.meta.url);

test("live mode does not read a mock mode token or phone profile", () => {
  const values = new Map([["hotel_api_mode", "http"]]);
  global.wx = {
    getStorageSync(key) { return values.get(key); },
    setStorageSync(key, value) { values.set(key, value); }
  };
  const session = require("../utils/session.js");
  session.setToken("mock_member_token");
  session.setAuthMember({ id: "mbr_demo", phone: "13800138901" });
  values.set("hotel_live_api_base_url", "https://api.example.com/api/v1");
  session.setApiMode("live");
  assert.equal(session.getToken(), "");
  assert.equal(session.getAuthMember(), null);
  session.setToken("old-live-token");
  session.setAuthMember({ id: "old-live-member" });
  session.setLiveApiBaseUrl("https://another.example.com/api/v1");
  assert.equal(session.getToken(), "");
  assert.equal(session.getAuthMember(), null);
  delete global.wx;
});
