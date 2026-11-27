import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { createRequire } from "node:module";
import test from "node:test";
import vm from "node:vm";

const pageFile = new URL("../pages/home/index.js", import.meta.url);
const require = createRequire(pageFile);

test("home reloads summary after returning from another page without double loading initially", async () => {
  let pageDefinition;
  let requests = 0;
  const app = {
    capturePageQuery() {},
    getRuntimeStatus: () => ({ apiMode: "http", apiModeText: "HTTP 接口（验收）" }),
    globalData: {
      launchContext: {},
      api: {
        async getCapabilities() { return { configured_booking: true }; },
        async getHome() {
          requests += 1;
          return {
            hotel: { name: "高碑店尚客优", address: "", phone: "", tags: [] },
            today_member_price_cent: 8000,
            wallet_balance_cent: requests * 100,
            coupon_count: requests,
            booking_count: requests,
            referral_summary: { referred_count: 0, available_reward_cent: 0 }
          };
        }
      }
    },
    async ensureLogin() {}
  };
  vm.runInNewContext(readFileSync(pageFile, "utf8"), {
    require,
    Page(definition) { pageDefinition = definition; },
    getApp: () => app,
    wx: { showToast() { assert.fail("unexpected load failure"); } }
  }, { filename: pageFile.pathname });

  const page = {
    ...pageDefinition,
    data: { ...pageDefinition.data },
    setData(changes) { Object.assign(this.data, changes); }
  };

  page.onLoad({ scene: "frontdesk_qr" });
  await new Promise(setImmediate);
  await page.onShow();
  assert.equal(requests, 1);
  assert.equal(page.data.couponCount, 1);

  await page.onShow();
  assert.equal(requests, 2);
  assert.equal(page.data.couponCount, 2);
  assert.equal(page.data.walletText, "¥2.00");
});

test("home clears stale summary and exposes an HTTP loading error", async () => {
  let pageDefinition;
  const app = {
    getRuntimeStatus: () => ({ apiMode: "live", apiModeText: "真实微信接口" }),
    globalData: {
      launchContext: {},
      api: { async getHome() { throw new Error("请求超时"); } }
    },
    async ensureLogin() {}
  };
  vm.runInNewContext(readFileSync(pageFile, "utf8"), {
    require,
    Page(definition) { pageDefinition = definition; },
    getApp: () => app,
    wx: { showToast() {} }
  }, { filename: pageFile.pathname });
  const page = {
    ...pageDefinition,
    data: { ...pageDefinition.data, walletText: "¥99.00", couponCount: 9 },
    setData(changes) { Object.assign(this.data, changes); }
  };
  await page.onShow();
  assert.equal(page.data.loadError, "请求超时");
  assert.equal(page.data.walletText, "--");
  assert.equal(page.data.couponCount, 0);
});
