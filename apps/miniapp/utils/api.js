const DEFAULT_BASE_URL = "http://localhost:8080/api/v1";

function trimTrailingSlash(value) {
  return String(value || DEFAULT_BASE_URL).replace(/\/+$/, "");
}

function buildQuery(query) {
  const entries = Object.keys(query || {})
    .filter((key) => query[key] !== undefined && query[key] !== null && query[key] !== "")
    .map((key) => `${encodeURIComponent(key)}=${encodeURIComponent(query[key])}`);

  return entries.length ? `?${entries.join("&")}` : "";
}

function buildUrl(baseUrl, path, query) {
  return `${trimTrailingSlash(baseUrl)}${path}${buildQuery(query)}`;
}

function createApiError(message, detail) {
  const error = new Error(message || "请求失败");
  error.detail = detail;
  return error;
}

function unwrapApiResponse(response) {
  const body = response && response.data ? response.data : {};

  if (response && response.statusCode && (response.statusCode < 200 || response.statusCode >= 300)) {
    throw createApiError(body.message || `HTTP ${response.statusCode}`, response);
  }

  if (body.code !== 0) {
    throw createApiError(body.message || "接口返回错误", body);
  }

  return body.data;
}

function normalizeMockLoginRequest(payload) {
  return {
    scene: payload.scene || "",
    referrer_member_id: payload.referrerMemberId || payload.referrer_member_id || ""
  };
}

function normalizeJoinMemberRequest(payload) {
  return {
    nickname: payload.nickname,
    phone: payload.phone,
    source: payload.source,
    scene: payload.scene,
    referrer_member_id: payload.referrerMemberId || payload.referrer_member_id || "",
    accepted_privacy: Boolean(payload.accepted_privacy),
    accepted_member_rule: Boolean(payload.accepted_member_rule),
    gift_choice: payload.gift_choice
  };
}

function normalizeBookingRequest(payload) {
  return {
    checkin_date: payload.checkin_date,
    checkout_date: payload.checkout_date,
    room_type: payload.room_type,
    arrival_time: payload.arrival_time,
    contact_name: payload.contact_name,
    phone: payload.phone,
    remark: payload.remark || ""
  };
}

function createApiClient(options) {
  const settings = options || {};
  const baseUrl = settings.baseUrl || DEFAULT_BASE_URL;
  const getToken = settings.getToken || (() => "");
  const fallback = settings.fallback;
  const demoOnly = settings.demoOnly === true;
  const onFallback = settings.onFallback || (() => {});
  const request = settings.request || ((requestOptions) => wx.request(requestOptions));

  function runFallback(descriptor, reason) {
    if (!fallback || !demoOnly) {
      throw createApiError(reason && reason.errMsg ? reason.errMsg : "网络请求失败", reason);
    }

    onFallback(descriptor, reason);
    return fallback(descriptor);
  }

  function send(descriptor) {
    const requestDescriptor = {
      method: descriptor.method || "GET",
      path: descriptor.path,
      query: descriptor.query || {},
      data: descriptor.data || {},
      auth: descriptor.auth !== false,
      idempotencyKey: descriptor.idempotencyKey
    };

    if (demoOnly) {
      return Promise.resolve(runFallback(requestDescriptor, { errMsg: "local demo mode" }));
    }

    const header = {
      "content-type": "application/json"
    };
    const token = getToken();

    if (requestDescriptor.auth && token) {
      header.Authorization = `Bearer ${token}`;
    }
    if (requestDescriptor.idempotencyKey) header["Idempotency-Key"] = requestDescriptor.idempotencyKey;

    return new Promise((resolve, reject) => {
      const requestOptions = {
        url: buildUrl(baseUrl, requestDescriptor.path, requestDescriptor.query),
        method: requestDescriptor.method,
        data: requestDescriptor.method === "GET" ? undefined : requestDescriptor.data,
        header,
        success(response) {
          try {
            resolve(unwrapApiResponse(response));
          } catch (error) {
            reject(error);
          }
        },
        fail(error) {
          reject(createApiError(error && error.errMsg ? error.errMsg : "网络请求失败", error));
        }
      };

      request(requestOptions);
    });
  }

  return {
    getCapabilities() {
      return send({ path: "/mini/capabilities", auth: false });
    },
    getRoomTypes(checkinDate, checkoutDate) {
      return send({ path: "/mini/room-types", query: { checkin_date: checkinDate, checkout_date: checkoutDate }, auth: false });
    },
    createQuote(payload, idempotencyKey) {
      return send({ method: "POST", path: "/bookings/quotes", data: payload, idempotencyKey });
    },
    submitConfiguredBooking(payload, idempotencyKey) {
      return send({ method: "POST", path: "/bookings/configured", data: payload, idempotencyKey });
    },
    mockLogin(payload) {
      return send({
        method: "POST",
        path: "/auth/mock-login",
        data: normalizeMockLoginRequest(payload || {}),
        auth: false
      });
    },
    wxLogin(code, payload) {
      return send({ method: "POST", path: "/auth/wx-login", data: {
        wx_login_code: code,
        ...normalizeMockLoginRequest(payload || {})
      }, auth: false });
    },
    wxBindPhone(code) {
      return send({ method: "POST", path: "/auth/wx-bind-phone", data: { phone_code: code } });
    },
    logout() {
      return send({ method: "POST", path: "/auth/logout" });
    },
    sendCode(phone) {
      return send({
        method: "POST",
        path: "/auth/send-code",
        data: {
          phone,
          purpose: "bind_phone"
        },
        auth: false
      });
    },
    bindPhone(phone, code) {
      return send({
        method: "POST",
        path: "/auth/bind-phone",
        data: {
          phone,
          code
        }
      });
    },
    joinMember(payload) {
      return send({
        method: "POST",
        path: "/members/join",
        data: normalizeJoinMemberRequest(payload || {})
      });
    },
    getHome() {
      return send({
        method: "GET",
        path: "/mini/home"
      });
    },
    getMe() {
      return send({
        method: "GET",
        path: "/members/me"
      });
    },
    getCoupons(status) {
      return send({
        method: "GET",
        path: "/coupons/my",
        query: {
          status
        }
      });
    },
    getWallet() {
      return send({
        method: "GET",
        path: "/wallet/me"
      });
    },
    createBooking(payload) {
      return send({
        method: "POST",
        path: "/bookings",
        data: normalizeBookingRequest(payload || {})
      });
    },
    getBookings() {
      return send({
        method: "GET",
        path: "/bookings/my"
      });
    },
    getReferral() {
      return send({
        method: "GET",
        path: "/referrals/me"
      });
    },
    bindReferral(referrerMemberId) {
      return send({
        method: "POST",
        path: "/referrals/bind",
        data: {
          referrer_member_id: referrerMemberId
        }
      });
    }
  };
}

module.exports = {
  DEFAULT_BASE_URL,
  buildQuery,
  buildUrl,
  createApiClient
};
