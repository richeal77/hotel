import assert from "node:assert/strict";
import { createRequire } from "node:module";
import test from "node:test";

const require = createRequire(import.meta.url);
const { guestPhoneForContext, phoneForBooking } = require("../utils/defaults.js");

test("front desk join defaults use the non-seed owner验收手机号", () => {
  assert.equal(
    guestPhoneForContext({
      source: "frontdesk",
      scene: "frontdesk_qr",
      referrerMemberId: ""
    }),
    "13800138901"
  );
});

test("referral join examples use a separate friend phone", () => {
  assert.equal(
    guestPhoneForContext({
      source: "member_share",
      scene: "referral_qr",
      referrerMemberId: "mbr_901"
    }),
    "13800138902"
  );
});

test("booking phone follows the bound member and otherwise stays empty", () => {
  assert.equal(
    phoneForBooking({
      phone: "13800138901",
      is_phone_bound: true
    }),
    "13800138901"
  );
  assert.equal(phoneForBooking(null), "");
});
