import assert from "node:assert/strict";
import { createRequire } from "node:module";
import test from "node:test";

const require = createRequire(import.meta.url);
const { validateJoinResult } = require("../utils/join.js");

test("join success requires issued gift coupons from backend", () => {
  assert.throws(
    () => validateJoinResult({
      bindMember: {
        id: "mbr_901",
        phone: "13800138901"
      },
      joinResult: {
        member: {
          id: "mbr_901",
          phone: "13800138901"
        },
        issued_coupons: []
      },
      expectedPhone: "13800138901"
    }),
    /后端未发放入会礼/
  );
});

test("join success requires backend member to match current bound session", () => {
  assert.throws(
    () => validateJoinResult({
      bindMember: {
        id: "mbr_901",
        phone: "13800138901"
      },
      joinResult: {
        member: {
          id: "mbr_seed_001",
          phone: "13800138901"
        },
        issued_coupons: [
          {
            id: "cpn_001",
            name: "洗衣券 1 次",
            status: "available"
          }
        ]
      },
      expectedPhone: "13800138901"
    }),
    /会员与当前会话不一致/
  );
});

test("join validation returns the issued coupons when member and gift are consistent", () => {
  const result = validateJoinResult({
    bindMember: {
      id: "mbr_901",
      phone: "13800138901"
    },
    joinResult: {
      member: {
        id: "mbr_901",
        phone: "13800138901"
      },
      issued_coupons: [
        {
          id: "cpn_001",
          name: "洗衣券 1 次",
          status: "available"
        }
      ]
    },
    expectedPhone: "13800138901"
  });

  assert.equal(result.member.id, "mbr_901");
  assert.equal(result.issuedCoupons.length, 1);
});
