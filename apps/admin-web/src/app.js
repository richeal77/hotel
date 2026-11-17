export const DEFAULT_API_BASE_URL = "http://localhost:8080/api/v1";

const storageKeys = {
  apiBaseUrl: "admin_api_base_url",
  staffToken: "admin_staff_token",
  staffUser: "admin_staff_user"
};

const statusText = {
  pending_confirm: "待前台确认",
  confirmed: "已确认",
  checked_in: "已入住",
  checked_out: "已离店",
  cancelled: "已取消"
};

const couponStatusText = {
  available: "可用",
  used: "已使用",
  expired: "已过期",
  voided: "已作废"
};

const bookingStatusOptions = [
  ["", "全部订单"],
  ["pending_confirm", "待前台确认"],
  ["confirmed", "已确认"],
  ["checked_in", "已入住"],
  ["checked_out", "已离店"],
  ["cancelled", "已取消"]
];

const state = {
  activeTab: "dashboard",
  apiBaseUrl: DEFAULT_API_BASE_URL,
  token: "",
  staff: null,
  dashboardDate: today(),
  bookingStatus: "",
  bookings: [],
  bookingPage: null,
  redemptionKeyword: "",
  redemption: null,
  selectedCouponId: "",
  selectedOrderId: "",
  demoState: null,
  toast: null,
  loading: false
};

export function sanitizeApiBaseUrl(value) {
  const trimmed = String(value || "").trim().replace(/\/+$/, "");
  return trimmed || DEFAULT_API_BASE_URL;
}

export function nextBookingStatus(status) {
  const transitions = {
    pending_confirm: { status: "confirmed", label: "确认订单" },
    confirmed: { status: "checked_in", label: "标记入住" },
    checked_in: { status: "checked_out", label: "标记离店" }
  };
  return transitions[status] || null;
}

function today() {
  const date = new Date();
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");
  return `${year}-${month}-${day}`;
}

function yuan(value) {
  const amount = Number(value || 0) / 100;
  return `¥${amount.toFixed(2)}`;
}

function text(value, fallback = "-") {
  if (value === null || value === undefined || value === "") {
    return fallback;
  }
  return String(value);
}

function escapeHtml(value) {
  return text(value, "")
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#039;");
}

function loadState() {
  state.apiBaseUrl = sanitizeApiBaseUrl(localStorage.getItem(storageKeys.apiBaseUrl) || DEFAULT_API_BASE_URL);
  state.token = localStorage.getItem(storageKeys.staffToken) || "";
  try {
    state.staff = JSON.parse(localStorage.getItem(storageKeys.staffUser) || "null");
  } catch {
    state.staff = null;
  }
}

function saveApiBaseUrl(value) {
  state.apiBaseUrl = sanitizeApiBaseUrl(value);
  localStorage.setItem(storageKeys.apiBaseUrl, state.apiBaseUrl);
}

function saveStaffSession(token, staff) {
  state.token = token;
  state.staff = staff;
  localStorage.setItem(storageKeys.staffToken, token);
  localStorage.setItem(storageKeys.staffUser, JSON.stringify(staff));
}

function clearStaffSession() {
  state.token = "";
  state.staff = null;
  localStorage.removeItem(storageKeys.staffToken);
  localStorage.removeItem(storageKeys.staffUser);
}

function proxyUrl(fullUrl) {
  if (typeof window === "undefined" || window.location.protocol === "file:") {
    return fullUrl;
  }
  return `/__admin_proxy?url=${encodeURIComponent(fullUrl)}`;
}

async function api(path, options = {}) {
  const fullUrl = `${sanitizeApiBaseUrl(state.apiBaseUrl)}${path}`;
  const headers = { Accept: "application/json", ...(options.headers || {}) };
  if (options.body !== undefined) {
    headers["Content-Type"] = "application/json";
  }
  if (state.token) {
    headers.Authorization = `Bearer ${state.token}`;
  }

  const response = await fetch(proxyUrl(fullUrl), {
    method: options.method || "GET",
    headers,
    body: options.body === undefined ? undefined : JSON.stringify(options.body)
  });

  let payload = null;
  try {
    payload = await response.json();
  } catch {
    throw new Error(`接口返回不是 JSON：HTTP ${response.status}`);
  }

  if (!response.ok || payload.code !== 0) {
    throw new Error(payload.message || `请求失败：HTTP ${response.status}`);
  }
  return payload.data;
}

function setToast(message, type = "success") {
  state.toast = { message, type };
  render();
  window.clearTimeout(setToast.timer);
  setToast.timer = window.setTimeout(() => {
    state.toast = null;
    render();
  }, 2600);
}

async function runAction(action, successMessage) {
  state.loading = true;
  render();
  try {
    const result = await action();
    if (successMessage) {
      setToast(successMessage, "success");
    }
    return result;
  } catch (error) {
    setToast(error.message || "操作失败", "error");
    return null;
  } finally {
    state.loading = false;
    render();
  }
}

async function login(event) {
  event.preventDefault();
  const form = new FormData(event.currentTarget);
  saveApiBaseUrl(form.get("apiBaseUrl"));
  const data = await runAction(
    () =>
      api("/admin/auth/login", {
        method: "POST",
        body: {
          username: form.get("username"),
          password: form.get("password")
        }
      }),
    "登录成功"
  );
  if (!data) {
    return;
  }
  saveStaffSession(data.access_token, data.staff);
  await loadDashboard();
}

async function loadDashboard() {
  const data = await runAction(() => api(`/admin/dashboard/summary?date=${encodeURIComponent(state.dashboardDate)}`));
  if (data) {
    state.dashboard = data;
    render();
  }
}

async function loadBookings() {
  const params = new URLSearchParams({ page: "1", page_size: "50" });
  if (state.bookingStatus) {
    params.set("status", state.bookingStatus);
  }
  const data = await runAction(() => api(`/admin/bookings?${params.toString()}`));
  if (data) {
    state.bookingPage = data;
    state.bookings = data.items || [];
    render();
  }
}

async function updateOrderStatus(order) {
  const next = nextBookingStatus(order.status);
  if (!next) {
    return;
  }
  const amount = Number(order.actual_amount_cent || order.estimated_amount_cent || 0);
  const data = await runAction(
    () =>
      api(`/admin/bookings/${encodeURIComponent(order.id)}/status`, {
        method: "PATCH",
        body: {
          status: next.status,
          actual_amount_cent: amount,
          remark: next.label
        }
      }),
    `${next.label}成功`
  );
  if (data) {
    await loadBookings();
    if (state.activeTab === "dashboard") {
      await loadDashboard();
    }
  }
}

async function lookupRedemption(event) {
  event.preventDefault();
  const form = new FormData(event.currentTarget);
  state.redemptionKeyword = String(form.get("keyword") || "").trim();
  const data = await runAction(
    () => api(`/admin/redemption/lookup?keyword=${encodeURIComponent(state.redemptionKeyword)}`),
    "查询成功"
  );
  if (data) {
    state.redemption = data;
    state.selectedCouponId = data.coupons?.[0]?.id || "";
    state.selectedOrderId = data.recent_orders?.[0]?.id || "";
    render();
  }
}

async function redeemCoupon(event) {
  event.preventDefault();
  const form = new FormData(event.currentTarget);
  const couponId = String(form.get("couponId") || "");
  const orderId = String(form.get("orderId") || "");
  if (!couponId || !orderId) {
    setToast("请选择券和订单", "error");
    return;
  }
  const data = await runAction(
    () =>
      api(`/admin/coupons/${encodeURIComponent(couponId)}/redeem`, {
        method: "POST",
        body: {
          order_id: orderId,
          remark: "前台核销"
        }
      }),
    "核销成功"
  );
  if (data && state.redemptionKeyword) {
    const refreshed = await runAction(() => api(`/admin/redemption/lookup?keyword=${encodeURIComponent(state.redemptionKeyword)}`));
    if (refreshed) {
      state.redemption = refreshed;
      state.selectedCouponId = refreshed.coupons?.[0]?.id || "";
      state.selectedOrderId = refreshed.recent_orders?.[0]?.id || "";
      render();
    }
  }
}

async function loadDemoState() {
  const data = await runAction(() => api("/admin/demo/state"));
  if (data) {
    state.demoState = data;
    render();
  }
}

async function resetDemo(event) {
  event.preventDefault();
  const form = new FormData(event.currentTarget);
  const confirmText = String(form.get("confirmText") || "").trim();
  if (confirmText !== "RESET") {
    setToast("请输入 RESET 后再重置", "error");
    return;
  }
  const data = await runAction(
    () =>
      api("/admin/demo/reset", {
        method: "POST",
        body: { confirm_text: confirmText }
      }),
    "演示数据已重置"
  );
  if (data) {
    state.demoState = data;
    state.dashboard = null;
    state.bookings = [];
    state.redemption = null;
    render();
  }
}

function metricCard(label, value) {
  return `
    <article class="metric">
      <span>${label}</span>
      <strong>${escapeHtml(value)}</strong>
    </article>
  `;
}

function renderLogin() {
  return `
    <main class="login-shell">
      <section class="login-panel">
        <div>
          <p class="eyebrow">P0 管理后台</p>
          <h1>高碑店尚客优</h1>
        </div>
        <form id="loginForm" class="form-grid">
          <label>
            <span>API Base URL</span>
            <input name="apiBaseUrl" value="${escapeHtml(state.apiBaseUrl)}" autocomplete="off" />
          </label>
          <label>
            <span>账号</span>
            <input name="username" value="boss" autocomplete="username" />
          </label>
          <label>
            <span>密码</span>
            <input name="password" type="password" value="666666" autocomplete="current-password" />
          </label>
          <button class="primary" type="submit" ${state.loading ? "disabled" : ""}>登录</button>
        </form>
      </section>
    </main>
  `;
}

function renderShell() {
  const tabs = [
    ["dashboard", "老板看板"],
    ["orders", "订单管理"],
    ["redemption", "券核销"],
    ["demo", "Demo 控制台"]
  ];
  return `
    <header class="topbar">
      <div>
        <p class="eyebrow">高碑店尚客优</p>
        <h1>管理后台</h1>
      </div>
      <div class="staff-chip">
        <span>${escapeHtml(state.staff?.name || "Boss")}</span>
        <button id="logoutBtn" class="ghost" type="button">退出</button>
      </div>
    </header>
    <section class="config-row">
      <label>
        <span>API Base URL</span>
        <input id="apiBaseInput" value="${escapeHtml(state.apiBaseUrl)}" autocomplete="off" />
      </label>
      <button id="saveApiBaseBtn" class="secondary" type="button">保存</button>
    </section>
    <nav class="tabs">
      ${tabs
        .map(
          ([key, label]) => `
          <button class="${state.activeTab === key ? "active" : ""}" data-tab="${key}" type="button">${label}</button>
        `
        )
        .join("")}
    </nav>
    <main class="content">
      ${renderActiveTab()}
    </main>
  `;
}

function renderActiveTab() {
  if (state.activeTab === "orders") {
    return renderOrders();
  }
  if (state.activeTab === "redemption") {
    return renderRedemption();
  }
  if (state.activeTab === "demo") {
    return renderDemo();
  }
  return renderDashboard();
}

function renderDashboard() {
  const summary = state.dashboard;
  return `
    <section class="section-head">
      <div>
        <h2>老板看板</h2>
      </div>
      <form id="dashboardForm" class="inline-form">
        <input name="date" type="date" value="${escapeHtml(state.dashboardDate)}" />
        <button class="primary" type="submit" ${state.loading ? "disabled" : ""}>查询</button>
      </form>
    </section>
    ${
      summary
        ? `
      <section class="metrics-grid">
        ${metricCard("今日入会数", summary.today_members)}
        ${metricCard("今日订单数", summary.today_orders)}
        ${metricCard("今日核销券数", summary.today_coupon_used)}
        ${metricCard("本月复购间夜", summary.month_repeat_room_nights)}
        ${metricCard("本月推荐间夜", summary.month_referral_room_nights)}
        ${metricCard("返住金发放", yuan(summary.month_wallet_granted_cent))}
        ${metricCard("返住金核销", yuan(summary.month_wallet_used_cent))}
      </section>
      <section class="split-grid">
        <article class="panel">
          <h3>前台入会排行</h3>
          ${renderSimpleTable(summary.frontdesk_rank || [], ["staff_name", "joined_members"], ["前台", "入会数"])}
        </article>
        <article class="panel">
          <h3>来源占比</h3>
          ${renderSimpleTable(summary.source_ratio || [], ["source", "count"], ["来源", "人数"])}
        </article>
      </section>
    `
        : `<div class="empty">请选择日期查询看板。</div>`
    }
  `;
}

function renderSimpleTable(items, keys, labels) {
  if (!items.length) {
    return `<div class="empty slim">暂无数据</div>`;
  }
  return `
    <table>
      <thead>
        <tr>${labels.map((label) => `<th>${label}</th>`).join("")}</tr>
      </thead>
      <tbody>
        ${items
          .map(
            (item) => `
          <tr>${keys.map((key) => `<td>${escapeHtml(item[key])}</td>`).join("")}</tr>
        `
          )
          .join("")}
      </tbody>
    </table>
  `;
}

function renderOrders() {
  return `
    <section class="section-head">
      <div>
        <h2>订单管理</h2>
      </div>
      <form id="bookingsForm" class="inline-form">
        <select name="status">
          ${bookingStatusOptions
            .map(([value, label]) => `<option value="${value}" ${state.bookingStatus === value ? "selected" : ""}>${label}</option>`)
            .join("")}
        </select>
        <button class="primary" type="submit" ${state.loading ? "disabled" : ""}>刷新</button>
      </form>
    </section>
    <section class="panel table-panel">
      ${
        state.bookings.length
          ? `
        <table>
          <thead>
            <tr>
              <th>订单</th>
              <th>住客</th>
              <th>房型</th>
              <th>入住/离店</th>
              <th>金额</th>
              <th>状态</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            ${state.bookings.map(renderOrderRow).join("")}
          </tbody>
        </table>
      `
          : `<div class="empty">暂无订单，点击刷新获取最新数据。</div>`
      }
    </section>
  `;
}

function renderOrderRow(order) {
  const next = nextBookingStatus(order.status);
  return `
    <tr>
      <td>
        <strong>${escapeHtml(order.id)}</strong>
        <small>${escapeHtml(order.created_at)}</small>
      </td>
      <td>
        ${escapeHtml(order.contact_name)}
        <small>${escapeHtml(order.phone)}</small>
      </td>
      <td>${escapeHtml(order.room_type)}</td>
      <td>
        ${escapeHtml(order.checkin_date)}
        <small>${escapeHtml(order.checkout_date)} · ${escapeHtml(order.arrival_time)}</small>
      </td>
      <td>${yuan(order.actual_amount_cent || order.estimated_amount_cent)}</td>
      <td><span class="badge ${escapeHtml(order.status)}">${escapeHtml(order.status_text || statusText[order.status] || order.status)}</span></td>
      <td>
        ${
          next
            ? `<button class="secondary compact" data-order-action="${escapeHtml(order.id)}" type="button">${escapeHtml(next.label)}</button>`
            : `<span class="muted">无</span>`
        }
      </td>
    </tr>
  `;
}

function renderRedemption() {
  const lookup = state.redemption;
  return `
    <section class="section-head">
      <div>
        <h2>券核销</h2>
      </div>
      <form id="redemptionLookupForm" class="inline-form">
        <input name="keyword" value="${escapeHtml(state.redemptionKeyword)}" placeholder="手机号" />
        <button class="primary" type="submit" ${state.loading ? "disabled" : ""}>查询</button>
      </form>
    </section>
    ${
      lookup
        ? `
      <section class="split-grid">
        <article class="panel">
          <h3>${escapeHtml(lookup.member.nickname)} · ${escapeHtml(lookup.member.phone)}</h3>
          <div class="detail-list">
            <span>会员状态</span><strong>${escapeHtml(lookup.member.status)}</strong>
            <span>返住金余额</span><strong>${yuan(lookup.wallet_balance_cent)}</strong>
            <span>可用券数</span><strong>${lookup.coupons.length}</strong>
          </div>
        </article>
        <article class="panel">
          <h3>核销</h3>
          <form id="redeemForm" class="form-grid">
            <label>
              <span>选择券</span>
              <select name="couponId">
                ${lookup.coupons
                  .map(
                    (coupon) =>
                      `<option value="${escapeHtml(coupon.id)}" ${state.selectedCouponId === coupon.id ? "selected" : ""}>${escapeHtml(coupon.name)} · ${escapeHtml(couponStatusText[coupon.status] || coupon.status)}</option>`
                  )
                  .join("")}
              </select>
            </label>
            <label>
              <span>选择订单</span>
              <select name="orderId">
                ${lookup.recent_orders
                  .map(
                    (order) =>
                      `<option value="${escapeHtml(order.id)}" ${state.selectedOrderId === order.id ? "selected" : ""}>${escapeHtml(order.id)} · ${escapeHtml(order.room_type)} · ${escapeHtml(order.status_text || order.status)}</option>`
                  )
                  .join("")}
              </select>
            </label>
            <button class="primary" type="submit" ${state.loading || !lookup.coupons.length || !lookup.recent_orders.length ? "disabled" : ""}>核销</button>
          </form>
        </article>
      </section>
      <section class="panel table-panel">
        <h3>可用券</h3>
        ${renderCouponsTable(lookup.coupons)}
      </section>
    `
        : `<div class="empty">输入手机号查询可用券和最近订单。</div>`
    }
  `;
}

function renderCouponsTable(coupons) {
  if (!coupons.length) {
    return `<div class="empty slim">暂无可用券</div>`;
  }
  return `
    <table>
      <thead>
        <tr>
          <th>券</th>
          <th>类型</th>
          <th>有效期</th>
          <th>规则</th>
        </tr>
      </thead>
      <tbody>
        ${coupons
          .map(
            (coupon) => `
          <tr>
            <td><strong>${escapeHtml(coupon.name)}</strong><small>${escapeHtml(coupon.id)}</small></td>
            <td>${escapeHtml(coupon.type)}</td>
            <td>${escapeHtml(coupon.expire_at)}</td>
            <td>${escapeHtml(coupon.use_rule)}</td>
          </tr>
        `
          )
          .join("")}
      </tbody>
    </table>
  `;
}

function renderDemo() {
  return `
    <section class="section-head">
      <div>
        <h2>Demo 控制台</h2>
      </div>
      <button id="loadDemoStateBtn" class="secondary" type="button">刷新状态</button>
    </section>
    <section class="split-grid">
      <article class="panel">
        <h3>重置演示数据</h3>
        <form id="resetDemoForm" class="form-grid">
          <label>
            <span>确认文本</span>
            <input name="confirmText" placeholder="RESET" autocomplete="off" />
          </label>
          <button class="danger" type="submit" ${state.loading ? "disabled" : ""}>重置</button>
        </form>
      </article>
      <article class="panel">
        <h3>当前状态</h3>
        ${
          state.demoState
            ? `<div class="detail-list">
              <span>会员数</span><strong>${state.demoState.member_count}</strong>
              <span>订单数</span><strong>${state.demoState.order_count}</strong>
              <span>券数量</span><strong>${state.demoState.coupon_count}</strong>
              <span>返住金流水</span><strong>${state.demoState.wallet_ledger_count}</strong>
              <span>推荐奖励</span><strong>${state.demoState.referral_reward_count}</strong>
              <span>审计日志</span><strong>${state.demoState.audit_log_count}</strong>
              <span>模拟日期</span><strong>${escapeHtml(state.demoState.simulated_date)}</strong>
            </div>`
            : `<div class="empty slim">暂无状态</div>`
        }
      </article>
    </section>
  `;
}

function render() {
  const app = document.querySelector("#app");
  app.innerHTML = `
    ${state.token ? renderShell() : renderLogin()}
    ${state.toast ? `<div class="toast ${state.toast.type}">${escapeHtml(state.toast.message)}</div>` : ""}
  `;
  bindEvents();
}

function bindEvents() {
  document.querySelector("#loginForm")?.addEventListener("submit", login);
  document.querySelector("#logoutBtn")?.addEventListener("click", () => {
    clearStaffSession();
    setToast("已退出", "success");
  });
  document.querySelector("#saveApiBaseBtn")?.addEventListener("click", () => {
    saveApiBaseUrl(document.querySelector("#apiBaseInput")?.value);
    setToast("API 地址已保存", "success");
  });
  document.querySelectorAll("[data-tab]").forEach((button) => {
    button.addEventListener("click", async () => {
      state.activeTab = button.dataset.tab;
      render();
      if (state.activeTab === "dashboard" && !state.dashboard) {
        await loadDashboard();
      }
      if (state.activeTab === "orders" && !state.bookings.length) {
        await loadBookings();
      }
      if (state.activeTab === "demo") {
        state.demoState = null;
        await loadDemoState();
      }
    });
  });
  document.querySelector("#dashboardForm")?.addEventListener("submit", async (event) => {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    state.dashboardDate = String(form.get("date") || today());
    await loadDashboard();
  });
  document.querySelector("#bookingsForm")?.addEventListener("submit", async (event) => {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    state.bookingStatus = String(form.get("status") || "");
    await loadBookings();
  });
  document.querySelectorAll("[data-order-action]").forEach((button) => {
    button.addEventListener("click", async () => {
      const order = state.bookings.find((item) => item.id === button.dataset.orderAction);
      if (order) {
        await updateOrderStatus(order);
      }
    });
  });
  document.querySelector("#redemptionLookupForm")?.addEventListener("submit", lookupRedemption);
  document.querySelector("#redeemForm")?.addEventListener("submit", redeemCoupon);
  document.querySelector("#loadDemoStateBtn")?.addEventListener("click", loadDemoState);
  document.querySelector("#resetDemoForm")?.addEventListener("submit", resetDemo);
}

if (typeof window !== "undefined" && typeof document !== "undefined") {
  loadState();
  render();
  if (state.token) {
    loadDashboard();
  }
}
