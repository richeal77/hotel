import assert from "node:assert/strict";
import { createRequire } from "node:module";
import test from "node:test";

const require = createRequire(import.meta.url);
const { parseLaunchContext } = require("../utils/scene.js");

test("direct launch params preserve scene and referrer", () => {
  const context = parseLaunchContext({
    scene: 1047,
    query: {
      scene: "frontdesk_qr",
      referrer: "mbr_001"
    }
  });

  assert.deepEqual(context, {
    launchScene: 1047,
    source: "frontdesk",
    scene: "frontdesk_qr",
    referrerMemberId: "mbr_001"
  });
});

test("encoded QR scene params are decoded into contract fields", () => {
  const context = parseLaunchContext({
    scene: 1047,
    query: {
      scene: encodeURIComponent("source=member_share&scene=referral_qr&referrer_member_id=mbr_888")
    }
  });

  assert.deepEqual(context, {
    launchScene: 1047,
    source: "member_share",
    scene: "referral_qr",
    referrerMemberId: "mbr_888"
  });
});

test("empty launch params fall back to front desk demo defaults", () => {
  const context = parseLaunchContext({});

  assert.deepEqual(context, {
    launchScene: undefined,
    source: "frontdesk",
    scene: "frontdesk_qr",
    referrerMemberId: ""
  });
});
