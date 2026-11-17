import assert from "node:assert/strict";
import { createRequire } from "node:module";
import test from "node:test";

const require = createRequire(import.meta.url);
const { createApiClient } = require("../utils/api.js");

function createRequestRecorder(response) {
  const calls = [];

  function request(options) {
    calls.push(options);
    options.success({
      statusCode: 200,
      data: response ?? {
        code: 0,
        message: "ok",
        data: {}
      }
    });
  }

  return { calls, request };
}

test("mock login uses the contract route and body without bearer auth", async () => {
  const recorder = createRequestRecorder({
    code: 0,
    message: "ok",
    data: {
      member_token: "mock_member_token",
      member: {
        id: "mbr_001",
        openid: "mock_openid_001",
        phone: "",
        is_phone_bound: false
      }
    }
  });
  const client = createApiClient({
    baseUrl: "http://localhost:8080/api/v1",
    request: recorder.request,
    getToken: () => "existing-token"
  });

  const data = await client.mockLogin({
    scene: "frontdesk_qr",
    referrerMemberId: "mbr_009"
  });

  assert.equal(data.member_token, "mock_member_token");
  assert.equal(recorder.calls.length, 1);
  assert.equal(recorder.calls[0].url, "http://localhost:8080/api/v1/auth/mock-login");
  assert.equal(recorder.calls[0].method, "POST");
  assert.deepEqual(recorder.calls[0].data, {
    scene: "frontdesk_qr",
    referrer_member_id: "mbr_009"
  });
  assert.equal(recorder.calls[0].header.Authorization, undefined);
});

test("member endpoints send bearer auth and unwrap API response data", async () => {
  const recorder = createRequestRecorder({
    code: 0,
    message: "ok",
    data: {
      hotel: {
        name: "Gaobeidian Shangkeyou",
        address: "Hebei Baoding Gaobeidian",
        phone: "0312-0000000",
        tags: ["parking"]
      },
      today_member_price_cent: 8000,
      wallet_balance_cent: 240,
      coupon_count: 2,
      booking_count: 1,
      referral_summary: {
        referred_count: 1,
        available_reward_cent: 400
      }
    }
  });
  const client = createApiClient({
    baseUrl: "http://localhost:8080/api/v1/",
    request: recorder.request,
    getToken: () => "member-token"
  });

  const data = await client.getHome();

  assert.equal(data.today_member_price_cent, 8000);
  assert.equal(recorder.calls[0].url, "http://localhost:8080/api/v1/mini/home");
  assert.equal(recorder.calls[0].method, "GET");
  assert.equal(recorder.calls[0].header.Authorization, "Bearer member-token");
});

test("coupon status is encoded as the documented status query", async () => {
  const recorder = createRequestRecorder({
    code: 0,
    message: "ok",
    data: {
      items: []
    }
  });
  const client = createApiClient({
    baseUrl: "http://localhost:8080/api/v1",
    request: recorder.request,
    getToken: () => "member-token"
  });

  await client.getCoupons("available");

  assert.equal(recorder.calls[0].url, "http://localhost:8080/api/v1/coupons/my?status=available");
});

test("non-zero API responses reject with the server message", async () => {
  const recorder = createRequestRecorder({
    code: 40100,
    message: "missing token",
    data: null
  });
  const client = createApiClient({
    baseUrl: "http://localhost:8080/api/v1",
    request: recorder.request,
    getToken: () => ""
  });

  await assert.rejects(
    () => client.getHome(),
    /missing token/
  );
});

test("network failure rejects by default even when a demo fallback exists", async () => {
  let fallbackCalled = false;
  const client = createApiClient({
    baseUrl: "http://192.168.1.10:8080/api/v1",
    request(options) {
      options.fail({ errMsg: "request:fail timeout" });
    },
    getToken: () => "member-token",
    fallback() {
      fallbackCalled = true;
      return { ok: true };
    }
  });

  await assert.rejects(
    () => client.getHome(),
    /request:fail timeout/
  );
  assert.equal(fallbackCalled, false);
});

test("network failure uses demo fallback only when local demo mode is explicit", async () => {
  let fallbackCalled = false;
  const client = createApiClient({
    baseUrl: "http://localhost:8080/api/v1",
    enableFallback: true,
    request(options) {
      options.fail({ errMsg: "request:fail timeout" });
    },
    getToken: () => "member-token",
    fallback(descriptor) {
      fallbackCalled = true;
      assert.equal(descriptor.path, "/mini/home");
      return { hotel: { name: "demo" } };
    }
  });

  const data = await client.getHome();

  assert.deepEqual(data, { hotel: { name: "demo" } });
  assert.equal(fallbackCalled, true);
});

test("local demo mode can bypass HTTP requests entirely", async () => {
  let requestCalled = false;
  const client = createApiClient({
    baseUrl: "http://localhost:8080/api/v1",
    demoOnly: true,
    request() {
      requestCalled = true;
    },
    fallback(descriptor) {
      assert.equal(descriptor.path, "/mini/home");
      return {
        hotel: {
          name: "本地演示"
        }
      };
    }
  });

  const data = await client.getHome();

  assert.equal(requestCalled, false);
  assert.equal(data.hotel.name, "本地演示");
});
