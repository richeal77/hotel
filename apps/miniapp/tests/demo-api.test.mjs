import assert from "node:assert/strict";
import { createRequire } from "node:module";
import test from "node:test";

const require = createRequire(import.meta.url);
const { createDemoResponder } = require("../utils/demo-api.js");

function createMemoryStorage() {
  const state = new Map();

  return {
    get(key) {
      return state.get(key);
    },
    set(key, value) {
      state.set(key, value);
    }
  };
}

test("local demo responder follows the guest membership and booking contract flow", async () => {
  const demo = createDemoResponder({
    now: () => new Date("2026-09-12T10:00:00.000Z"),
    storage: createMemoryStorage()
  });

  const login = await demo({
    method: "POST",
    path: "/auth/mock-login",
    data: {
      scene: "frontdesk_qr",
      referrer_member_id: "mbr_900"
    }
  });
  assert.equal(login.member_token, "mock_member_token");
  assert.equal(login.member.is_phone_bound, false);

  const code = await demo({
    method: "POST",
    path: "/auth/send-code",
    data: {
      phone: "13800138901",
      purpose: "bind_phone"
    }
  });
  assert.deepEqual(code, {
    mock_code: "666666",
    expire_seconds: 300
  });

  const bind = await demo({
    method: "POST",
    path: "/auth/bind-phone",
    data: {
      phone: "13800138901",
      code: "666666"
    }
  });
  assert.equal(bind.member.phone, "13800138901");
  assert.equal(bind.member.is_phone_bound, true);

  const joined = await demo({
    method: "POST",
    path: "/members/join",
    data: {
      nickname: "张三",
      phone: "13800138901",
      source: "frontdesk",
      scene: "frontdesk_qr",
      referrer_member_id: "mbr_900",
      accepted_privacy: true,
      accepted_member_rule: true,
      gift_choice: "laundry"
    }
  });
  assert.equal(joined.member.referrer_member_id, "mbr_900");
  assert.equal(joined.issued_coupons[0].name, "洗衣券 1 次");

  const booking = await demo({
    method: "POST",
    path: "/bookings",
    data: {
      checkin_date: "2026-09-13",
      checkout_date: "2026-09-14",
      room_type: "标准大床房",
      arrival_time: "20:00",
      contact_name: "张三",
      phone: "13800138901",
      remark: "想要安静房间"
    }
  });
  assert.equal(booking.order.status, "pending_confirm");
  assert.equal(booking.order.status_text, "待前台确认");

  const orders = await demo({
    method: "GET",
    path: "/bookings/my"
  });
  assert.equal(orders.items.length, 1);
  assert.equal(orders.items[0].room_type, "标准大床房");

  const coupons = await demo({
    method: "GET",
    path: "/coupons/my",
    query: {
      status: "available"
    }
  });
  assert.equal(coupons.items.length, 1);
  assert.equal(coupons.items[0].status, "available");
});
