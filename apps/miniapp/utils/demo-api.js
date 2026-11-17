const config = require("./config");

const STORAGE_KEY = "hotel_demo_api_state";
const MEMBER_ID = "mbr_demo_001";
const TOKEN = "mock_member_token";

function toIso(now) {
  return now().toISOString();
}

function toDate(now) {
  return toIso(now).slice(0, 10);
}

function addDays(dateText, days) {
  const date = new Date(`${dateText}T00:00:00.000Z`);
  date.setUTCDate(date.getUTCDate() + days);
  return date.toISOString().slice(0, 10);
}

function addDaysIso(now, days) {
  const date = now();
  date.setUTCDate(date.getUTCDate() + days);
  return date.toISOString();
}

function createInitialState(now) {
  return {
    member_token: TOKEN,
    auth_member: {
      id: MEMBER_ID,
      openid: "mock_openid_demo_001",
      phone: "",
      is_phone_bound: false
    },
    member: null,
    coupons: [],
    wallet: {
      balance_cent: 0,
      total_granted_cent: 0,
      total_used_cent: 0,
      ledgers: []
    },
    bookings: [],
    referral_relation: null,
    referral_rewards: [],
    login_context: {
      scene: "frontdesk_qr",
      referrer_member_id: ""
    },
    created_at: toIso(now)
  };
}

function clone(value) {
  return JSON.parse(JSON.stringify(value));
}

function loadState(storage, now) {
  const saved = storage.get(STORAGE_KEY);
  return saved ? clone(saved) : createInitialState(now);
}

function saveState(storage, state) {
  storage.set(STORAGE_KEY, clone(state));
}

function ensureMember(state, now) {
  if (state.member) {
    return state.member;
  }

  return {
    id: MEMBER_ID,
    openid: "mock_openid_demo_001",
    phone: state.auth_member.phone || "",
    nickname: "演示住客",
    source: "frontdesk",
    source_scene: state.login_context.scene || "frontdesk_qr",
    referrer_member_id: state.login_context.referrer_member_id || "",
    status: "active",
    join_time: toIso(now),
    stay_count: 0,
    created_at: toIso(now),
    updated_at: toIso(now)
  };
}

function giftName(giftChoice) {
  const names = {
    laundry: "洗衣券 1 次",
    late_checkout: "延迟退房券",
    next_stay_5: "下次入住立减 5 元"
  };

  return names[giftChoice] || "洗衣券 1 次";
}

function giftType(giftChoice) {
  const types = {
    laundry: "service_coupon",
    late_checkout: "late_checkout",
    next_stay_5: "cash_discount"
  };

  return types[giftChoice] || "service_coupon";
}

function issueGiftCoupon(state, request, now) {
  if (state.coupons.some((coupon) => coupon.template_id === request.gift_choice)) {
    return [];
  }

  const coupon = {
    id: `cpn_${String(state.coupons.length + 1).padStart(3, "0")}`,
    template_id: request.gift_choice,
    member_id: MEMBER_ID,
    name: giftName(request.gift_choice),
    type: giftType(request.gift_choice),
    value_cent: request.gift_choice === "next_stay_5" ? 500 : 0,
    status: "available",
    received_at: toIso(now),
    expire_at: addDaysIso(now, 30),
    used_at: null,
    used_order_id: null,
    use_rule: "演示版权益，到店后由前台核销"
  };

  state.coupons.unshift(coupon);
  return [{
    id: coupon.id,
    name: coupon.name,
    status: coupon.status
  }];
}

function createBooking(state, request, now) {
  const id = `ord_${String(state.bookings.length + 1).padStart(3, "0")}`;
  const member = ensureMember(state, now);
  const order = {
    id,
    status: "pending_confirm",
    status_text: "待前台确认",
    member_id: member.id,
    phone: request.phone,
    contact_name: request.contact_name,
    room_type: request.room_type,
    checkin_date: request.checkin_date,
    checkout_date: request.checkout_date,
    arrival_time: request.arrival_time,
    channel: "miniapp",
    source: member.source || "frontdesk",
    remark: request.remark || "",
    estimated_amount_cent: 8000,
    actual_amount_cent: 8000,
    paid_status: "pay_at_hotel",
    operator_staff_id: "",
    created_at: toIso(now),
    updated_at: toIso(now)
  };

  state.bookings.unshift(order);

  return {
    id: order.id,
    status: order.status,
    status_text: order.status_text
  };
}

function referralSummary(state) {
  return {
    referred_count: state.referral_relation ? 1 : 0,
    referred_members: state.referral_relation ? 1 : 0,
    referred_orders: state.referral_rewards.length,
    pending_reward_cent: 0,
    available_reward_cent: state.referral_rewards
      .filter((reward) => reward.status === "available")
      .reduce((total, reward) => total + reward.reward_amount_cent, 0)
  };
}

function homeData(state) {
  return {
    hotel: {
      name: "高碑店尚客优",
      address: "河北保定高碑店核心商街，近老火车站",
      phone: "0312-0000000",
      tags: ["停车", "洗衣", "发票", "棋牌"]
    },
    today_member_price_cent: 8000,
    wallet_balance_cent: state.wallet.balance_cent,
    coupon_count: state.coupons.filter((coupon) => coupon.status === "available").length,
    booking_count: state.bookings.length,
    referral_summary: referralSummary(state)
  };
}

function createDemoResponder(options) {
  const settings = options || {};
  const storage = settings.storage;
  const now = settings.now || (() => new Date());

  if (!storage || typeof storage.get !== "function" || typeof storage.set !== "function") {
    throw new Error("demo responder requires storage get/set");
  }

  return async function respond(descriptor) {
    const state = loadState(storage, now);
    const path = descriptor.path;
    const method = descriptor.method || "GET";
    const data = descriptor.data || {};
    const query = descriptor.query || {};
    let result;

    if (method === "POST" && path === "/auth/mock-login") {
      state.login_context = {
        scene: data.scene || "frontdesk_qr",
        referrer_member_id: data.referrer_member_id || ""
      };
      result = {
        member_token: state.member_token,
        member: state.auth_member
      };
    } else if (method === "POST" && path === "/auth/send-code") {
      result = {
        mock_code: config.demoCode,
        expire_seconds: 300
      };
    } else if (method === "POST" && path === "/auth/bind-phone") {
      if (data.code !== config.demoCode) {
        throw new Error("演示验证码固定为 666666");
      }
      state.auth_member.phone = data.phone;
      state.auth_member.is_phone_bound = true;
      result = {
        member: state.auth_member
      };
    } else if (method === "POST" && path === "/members/join") {
      state.member = {
        id: MEMBER_ID,
        openid: "mock_openid_demo_001",
        phone: data.phone,
        nickname: data.nickname,
        source: data.source,
        source_scene: data.scene,
        referrer_member_id: data.referrer_member_id || "",
        status: "active",
        join_time: toIso(now),
        stay_count: state.bookings.length,
        last_stay_date: state.bookings.length ? state.bookings[0].checkout_date : undefined,
        created_at: state.created_at,
        updated_at: toIso(now)
      };
      state.auth_member.phone = data.phone;
      state.auth_member.is_phone_bound = true;
      result = {
        member: state.member,
        issued_coupons: issueGiftCoupon(state, data, now)
      };
    } else if (method === "GET" && path === "/mini/home") {
      result = homeData(state);
    } else if (method === "GET" && path === "/members/me") {
      result = {
        member: ensureMember(state, now),
        member_code: MEMBER_ID
      };
    } else if (method === "GET" && path === "/coupons/my") {
      result = {
        items: query.status ? state.coupons.filter((coupon) => coupon.status === query.status) : state.coupons
      };
    } else if (method === "GET" && path === "/wallet/me") {
      result = state.wallet;
    } else if (method === "POST" && path === "/bookings") {
      result = {
        order: createBooking(state, data, now)
      };
    } else if (method === "GET" && path === "/bookings/my") {
      result = {
        items: state.bookings
      };
    } else if (method === "GET" && path === "/referrals/me") {
      const member = ensureMember(state, now);
      result = {
        share_scene: `referrer=${member.id}`,
        share_qr_text: `demo://miniapp?referrer=${member.id}`,
        summary: referralSummary(state),
        rewards: state.referral_rewards
      };
    } else if (method === "POST" && path === "/referrals/bind") {
      state.referral_relation = {
        id: "rfr_demo_001",
        referrer_member_id: data.referrer_member_id,
        referred_member_id: MEMBER_ID,
        created_at: toIso(now)
      };
      if (state.member) {
        state.member.referrer_member_id = data.referrer_member_id;
      }
      result = {
        relation: state.referral_relation
      };
    } else {
      throw new Error(`未实现的演示接口：${method} ${path}`);
    }

    saveState(storage, state);
    return clone(result);
  };
}

module.exports = {
  STORAGE_KEY,
  addDays,
  createDemoResponder
};
