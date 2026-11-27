export const DEFAULT_API_BASE_URL = "http://localhost:8080/api/v1";
import { createOperations } from "./operations.js";
const appConfig = typeof window === "undefined"
  ? { appEnv: "local-demo", apiBaseUrl: DEFAULT_API_BASE_URL, proxyEnabled: false }
  : (window.ADMIN_CONFIG || { appEnv: "production", apiBaseUrl: "/api/v1", proxyEnabled: false });

const storageKeys = {
  staffToken: "admin_staff_token",
  staffUser: "admin_staff_user"
};

const roleTabs = {
  boss: ["dashboard", "orders", "redemption", "audit", "demo"],
  manager: ["dashboard", "orders", "redemption"],
  frontdesk: ["orders", "redemption"],
  finance: ["audit"]
};

export function allowedTabs(role, appEnv) {
  if (!roleTabs[role]) return [];
  return [...roleTabs[role].filter((tab) => tab !== "demo" || appEnv === "local-demo"),
    "rooms", "inventory", "campaigns", "configuredOrders", "benefits", "operationLogs"];
}

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
  apiBaseUrl: appConfig.apiBaseUrl,
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
  auditPage: 1,
  auditData: null,
  toast: null,
  loading: false
};

export function sanitizeApiBaseUrl(value) {
  const trimmed = String(value || "").trim().replace(/\/+$/, "");
  return trimmed || DEFAULT_API_BASE_URL;
}

export function nonJsonApiError(path, status) {
  const error = new Error(status === 404
    ? `接口未部署：${path}（HTTP 404）。请更新并重启 API 服务。`
    : `接口返回不是 JSON：${path}（HTTP ${status}）`);
  error.status = status;
  return error;
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
  state.token = sessionStorage.getItem(storageKeys.staffToken) || "";
  localStorage.removeItem(storageKeys.staffToken);
  try {
    state.staff = JSON.parse(sessionStorage.getItem(storageKeys.staffUser) || "null");
  } catch {
    state.staff = null;
  }
  localStorage.removeItem("admin_api_base_url");
  localStorage.removeItem(storageKeys.staffUser);
  if (!state.staff || !allowedTabs(state.staff.role, appConfig.appEnv).length) {
    clearStaffSession();
  } else {
    state.activeTab = allowedTabs(state.staff.role, appConfig.appEnv)[0];
  }
}

function saveStaffSession(token, staff) {
  state.token = token;
  state.staff = staff;
  sessionStorage.setItem(storageKeys.staffToken, token);
  sessionStorage.setItem(storageKeys.staffUser, JSON.stringify(staff));
}

function clearStaffSession() {
  state.token = "";
  state.staff = null;
  sessionStorage.removeItem(storageKeys.staffToken);
  sessionStorage.removeItem(storageKeys.staffUser);
  localStorage.removeItem(storageKeys.staffToken);
  localStorage.removeItem(storageKeys.staffUser);
}

function apiUrl(path) {
  return appConfig.proxyEnabled ? `/__admin_proxy/api/v1${path}` : `${state.apiBaseUrl}${path}`;
}

async function api(path, options = {}) {
  const headers = { Accept: "application/json", ...(options.headers || {}) };
  if (options.body !== undefined) {
    headers["Content-Type"] = "application/json";
  }
  if (state.token) {
    headers.Authorization = `Bearer ${state.token}`;
  }

  const response = await fetch(apiUrl(path), {
    method: options.method || "GET",
    headers,
    body: options.body === undefined ? undefined : JSON.stringify(options.body)
  });

  if (response.status === 401 && state.token) {
    clearStaffSession();
    state.dashboard = null;
    state.bookings = [];
    state.redemption = null;
    state.auditData = null;
    state.demoState = null;
    throw new Error("登录已失效，请重新登录");
  }

  let payload = null;
  try {
    payload = await response.json();
  } catch {
    throw nonJsonApiError(path, response.status);
  }

  if (!response.ok || payload.code !== 0) {
    const reason = payload.data?.reason;
    const error = new Error([payload.message || `请求失败：HTTP ${response.status}`, reason].filter(Boolean).join(" · "));
    error.status = response.status;
    error.reason = reason;
    throw error;
  }
  return payload.data;
}

async function downloadCsv(path, filename) {
  const response = await fetch(apiUrl(path), {
    headers: { Accept: "text/csv", Authorization: `Bearer ${state.token}` }
  });
  if (response.status === 401) {
    clearStaffSession();
    throw new Error("登录已失效，请重新登录");
  }
  if (!response.ok) {
    let payload = null;
    try { payload = await response.json(); } catch { /* Non-JSON gateway errors have no structured reason. */ }
    const error = new Error(payload?.message || `导出失败：HTTP ${response.status}`);
    error.status = response.status;
    error.reason = payload?.data?.reason;
    throw error;
  }
  const blob = await response.blob();
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = filename;
  document.body.append(link);
  link.click();
  link.remove();
  URL.revokeObjectURL(url);
}

const operations = createOperations({ api, downloadCsv, notify: setToast, render: () => render() });

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
  const tabs = allowedTabs(data.staff?.role, appConfig.appEnv);
  if (!tabs.length) {
    setToast("当前账号没有可用的后台页面", "error");
    return;
  }
  saveStaffSession(data.access_token, data.staff);
  state.activeTab = tabs[0];
  render();
  await loadActiveTab();
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
  if (!window.confirm(`确认对订单 ${order.id} 执行“${next.label}”？`)) {
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
  const coupon = state.redemption?.coupons?.find((item) => item.id === couponId);
  if (!window.confirm(`确认将“${coupon?.name || couponId}”用于订单 ${orderId}？核销后不可撤销。`)) {
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

async function loadAuditLogs(page = state.auditPage) {
  const data = await runAction(() => api(`/admin/audit-logs?page=${page}&page_size=20`));
  if (data) {
    state.auditPage = data.page;
    state.auditData = data;
    render();
  }
}

async function loadActiveTab() {
  if (operations.hasTab(state.activeTab)) return operations.view(state.activeTab);
  if (state.activeTab === "dashboard") await loadDashboard();
  if (state.activeTab === "orders") await loadBookings();
  if (state.activeTab === "audit") await loadAuditLogs();
  if (state.activeTab === "demo") await loadDemoState();
}

async function resetDemo(event) {
  event.preventDefault();
  const form = new FormData(event.currentTarget);
  const confirmText = String(form.get("confirmText") || "").trim();
  if (confirmText !== "RESET") {
    setToast("请输入 RESET 后再重置", "error");
    return;
  }
  if (!window.confirm("确认重置所有演示数据？此操作不可撤销。")) {
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
          <p class="eyebrow">管理后台</p>
          <h1>高碑店尚客优</h1>
        </div>
        <form id="loginForm" class="form-grid">
          <label>
            <span>账号</span>
            <input name="username" autocomplete="username" required />
          </label>
          <label>
            <span>密码</span>
            <input name="password" type="password" autocomplete="current-password" required />
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
    ["audit", "审计日志"],
    ["demo", "Demo 控制台"],
    ...operations.tabs
  ].filter(([key]) => allowedTabs(state.staff?.role, appConfig.appEnv).includes(key));
  return `
    <header class="topbar">
      <div>
        <p class="eyebrow">高碑店尚客优</p>
        <h1>管理后台</h1>
      </div>
      <div class="staff-chip">
        <span>${escapeHtml(state.staff?.name || "员工")} · ${escapeHtml(state.staff?.role || "")}</span>
        <button id="logoutBtn" class="ghost" type="button">退出</button>
      </div>
    </header>
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
  if (operations.hasTab(state.activeTab)) return operations.html(state.activeTab);
  if (state.activeTab === "orders") {
    return renderOrders();
  }
  if (state.activeTab === "redemption") {
    return renderRedemption();
  }
  if (state.activeTab === "demo") {
    return renderDemo();
  }
  if (state.activeTab === "audit") {
    return renderAudit();
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

function renderAudit() {
  const result = state.auditData;
  const items = result?.items || [];
  const lastPage = result ? Math.max(1, Math.ceil(result.total / result.page_size)) : 1;
  return `
    <section class="section-head">
      <h2>审计日志</h2>
      <button id="refreshAuditBtn" class="secondary" type="button">刷新</button>
    </section>
    <section class="panel table-panel">
      ${items.length ? `
        <table>
          <thead><tr><th>时间</th><th>操作人</th><th>动作</th><th>对象</th><th>摘要</th><th>详情</th></tr></thead>
          <tbody>${items.map((item) => `
            <tr>
              <td>${escapeHtml(item.created_at)}</td>
              <td>${escapeHtml(item.operator_name || item.operator_staff_id)}</td>
              <td>${escapeHtml(item.action)}</td>
              <td>${escapeHtml(item.target_type)}<small>${escapeHtml(item.target_id)}</small></td>
              <td>${escapeHtml(item.summary)}</td>
              <td>${item.before || item.after ? `<details><summary>查看</summary><pre>${escapeHtml(JSON.stringify({ before: item.before || null, after: item.after || null }, null, 2))}</pre></details>` : "-"}</td>
            </tr>`).join("")}</tbody>
        </table>` : `<div class="empty">暂无审计记录</div>`}
    </section>
    <div class="pagination">
      <button class="secondary" data-audit-page="${state.auditPage - 1}" ${state.auditPage <= 1 ? "disabled" : ""}>上一页</button>
      <span>第 ${state.auditPage} / ${lastPage} 页 · 共 ${result?.total || 0} 条</span>
      <button class="secondary" data-audit-page="${state.auditPage + 1}" ${state.auditPage >= lastPage ? "disabled" : ""}>下一页</button>
    </div>
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
  operations.bind(state.activeTab);
  document.querySelector("#loginForm")?.addEventListener("submit", login);
  document.querySelector("#logoutBtn")?.addEventListener("click", async () => {
    if (appConfig.appEnv !== "local-demo") {
      try {
        await api("/admin/auth/logout", { method: "POST" });
      } catch (error) {
        clearStaffSession();
        setToast(`已退出本机；服务端撤销失败：${error.message}`, "error");
        return;
      }
    }
    clearStaffSession();
    setToast("已退出", "success");
  });
  document.querySelectorAll("[data-tab]").forEach((button) => {
    button.addEventListener("click", async () => {
      state.activeTab = button.dataset.tab;
      if (state.activeTab === "demo") state.demoState = null;
      render();
      await loadActiveTab();
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
  document.querySelector("#refreshAuditBtn")?.addEventListener("click", () => loadAuditLogs());
  document.querySelectorAll("[data-audit-page]").forEach((button) => {
    button.addEventListener("click", () => loadAuditLogs(Number(button.dataset.auditPage)));
  });
}

if (typeof window !== "undefined" && typeof document !== "undefined") {
  loadState();
  render();
  if (state.token) {
    loadActiveTab();
  }
}
