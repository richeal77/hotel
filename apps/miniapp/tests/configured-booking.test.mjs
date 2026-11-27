import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { createRequire } from "node:module";
import test from "node:test";
import vm from "node:vm";

const require = createRequire(import.meta.url);
const { createApiClient } = require("../utils/api.js");
const { needsNewQuote } = require("../utils/booking-flow.js");
const examples = JSON.parse(readFileSync(new URL("../../../docs/contracts/room-benefits-examples.json", import.meta.url), "utf8"));

test("configured booking routes carry contract fields and idempotency keys", async () => {
  const calls = [];
  const api = createApiClient({ baseUrl: "https://api.example.com/api/v1", getToken: () => "member-token",
    request(options) { calls.push(options); options.success({ statusCode: 200, data: { code: 0, data: {} } }); } });
  await api.getCapabilities();
  await api.getRoomTypes("2026-12-01", "2026-12-03");
  await api.createQuote({ rate_id: "rate_1", checkin_date: "2026-12-01", checkout_date: "2026-12-03" }, "quote-key");
  await api.submitConfiguredBooking({ quote_id: "quote_1", arrival_time: "18:00", contact_name: "住客", phone: "13800138001" }, "submit-key");
  assert.equal(calls[0].url, "https://api.example.com/api/v1/mini/capabilities");
  assert.equal(calls[0].header.Authorization, undefined);
  assert.match(calls[1].url, /mini\/room-types\?checkin_date=2026-12-01&checkout_date=2026-12-03$/);
  assert.equal(calls[2].header["Idempotency-Key"], "quote-key");
  assert.equal(calls[3].header["Idempotency-Key"], "submit-key");
  assert.equal(calls[3].header.Authorization, "Bearer member-token");
  assert.deepEqual(calls[3].data, { quote_id: "quote_1", arrival_time: "18:00", contact_name: "住客", phone: "13800138001" });
});

test("quote conflict requires a fresh quote and explicit second confirmation", async () => {
  let definition;
  let submits = 0;
  const pageFile = new URL("../pages/booking/index.js", import.meta.url);
  const pageRequire = createRequire(pageFile);
  const app = { getRuntimeStatus: () => ({ apiMode: "live", apiModeText: "真实微信接口" }),
    async ensureLogin() {}, globalData: { api: {
      async getCapabilities() { return { configured_booking: true }; },
      async getRoomTypes() { return { configured_booking: true, items: [{ id: "room_1", name: "大床", rates: [{ id: "rate_1", name: "会员价", nightly_cent: 7900, max_available_rooms: 1 }] }] }; },
      async createQuote() { return { quote: { ...examples.cases.quote_success.data.quote, expires_at: "2099-01-01T00:00:00Z" } }; },
      async submitConfiguredBooking() { submits++; throw { message: "报价变化", detail: { data: { data: { reason: "QUOTE_CHANGED" } } } }; }
    } } };
  vm.runInNewContext(readFileSync(pageFile, "utf8"), { require(id) {
    if (id === "../../utils/session") return { getAuthMember: () => ({ phone: "13800138001", is_phone_bound: true }) };
    return pageRequire(id);
  }, Page(value) { definition = value; }, getApp: () => app, wx: { showToast() {} } });
  const page = { ...definition, data: { ...definition.data }, setData(changes) { Object.assign(this.data, changes); } };
  page.syncBoundPhone();
  await page.loadAvailability();
  page.selectRate({ currentTarget: { dataset: { rateId: "rate_1" } } });
  await page.requestQuote();
  page.setData({ contactName: "住客" });
  await page.submitBooking();
  assert.equal(submits, 1);
  assert.equal(page.data.quote, null);
  assert.equal(page.data.quoteChanged, true);
  await page.submitBooking();
  assert.equal(submits, 1);
  assert.equal(needsNewQuote({ detail: { data: { data: { reason: "QUOTE_EXPIRED" } } } }), true);
});
