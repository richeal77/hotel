const newTabs = [
  ["rooms", "房型"], ["inventory", "日库存"], ["campaigns", "活动"],
  ["configuredOrders", "订单发礼核销"], ["benefits", "权益明细"], ["operationLogs", "操作日志"]
];

const esc = (value) => String(value ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;")
  .replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#39;");
const query = (values) => {
  const params = new URLSearchParams();
  Object.entries(values).forEach(([key, value]) => { if (value !== "" && value != null) params.set(key, value); });
  return params.toString();
};
const field = (name, label, value = "", type = "text") =>
  `<label><span>${esc(label)}</span><input name="${esc(name)}" type="${type}" value="${esc(value)}"></label>`;
const select = (name, label, options, value = "") =>
  `<label><span>${esc(label)}</span><select name="${esc(name)}">${options.map(([key, title]) =>
    `<option value="${esc(key)}" ${String(key) === String(value) ? "selected" : ""}>${esc(title)}</option>`).join("")}</select></label>`;
const button = (label, action, extra = "") => `<button type="button" data-op="${action}" ${extra}>${label}</button>`;
const json = (value) => `<pre class="op-json">${esc(JSON.stringify(value, null, 2))}</pre>`;
const list = (items, columns) => `<div class="op-table-wrap"><table class="op-table"><thead><tr>${columns.map(([, label]) => `<th>${esc(label)}</th>`).join("")}</tr></thead><tbody>${(items || []).map((item) => `<tr>${columns.map(([key]) => `<td>${esc(item[key] ?? "-")}</td>`).join("")}</tr>`).join("")}</tbody></table></div>`;
const paged = (data) => `<p class="op-muted">第 ${esc(data?.page || 1)} 页 / 共 ${esc(data?.total || 0)} 条</p>`;
const date = () => new Date().toISOString().slice(0, 10);
const defaultRule = () => ({ name: "", description: "", kind: "physical_gift", trigger: "checkin_gift", group_key: "", issue_start_at: "", issue_end_at: "", validity: { mode: "relative_days", days: 30 }, room_type_ids: [], payload: { item_name: "", quantity: 1, unit: "件", instructions: "" } });

export function operationLogPath(filters = {}, page = 1) {
  return `/admin/operation-audit-logs?${query({ ...filters, page, page_size: 20 })}`;
}

export function exportPath(kind, filters) {
  if (!filters.from || !filters.to || filters.from > filters.to) throw new Error("请选择有效的导出起止日期");
  const days = (Date.parse(`${filters.to}T00:00:00Z`) - Date.parse(`${filters.from}T00:00:00Z`)) / 86400000;
  if (!Number.isFinite(days) || days > 31) throw new Error("导出日期范围不能超过 31 天");
  const keys = kind === "bookings" ? ["from", "to", "status", "room_type_id", "order_id"] : ["from", "to"];
  return `/admin/${kind === "bookings" ? "bookings" : "benefit-events"}/export?${query(Object.fromEntries(keys.map((key) => [key, filters[key]])))}`;
}

export function createOperations({ api, downloadCsv, notify, render }) {
  const state = {
    rooms: {}, inventory: { from: date(), to: date() }, campaigns: {},
    configuredOrders: {}, benefits: {}, operationLogs: { filters: {}, page: 1 }
  };
  const request = async (path, options) => {
    const write = options && options.method && options.method !== "GET";
    return api(path, write ? { ...options, headers: { ...options.headers, "Idempotency-Key": crypto.randomUUID() } } : options);
  };
  const fail = (s, error) => {
    s.error = error.status === 409 ? `冲突：${error.reason || error.message}。请刷新后重试。` : error.message;
    notify(s.error, "error");
    render();
  };
  const run = async (tab, task) => {
    const s = state[tab];
    s.error = "";
    s.loading = true;
    render();
    try { await task(s); } catch (error) { fail(s, error); }
    finally { s.loading = false; render(); }
  };
  const selectedRoom = (s) => s.items?.some((item) => item.id === s.roomId) ? s.roomId : s.items?.[0]?.id;
  const selectedCampaign = (s) => s.items?.some((item) => item.id === s.campaignId) ? s.campaignId : s.items?.[0]?.id;
  const loadRooms = async (s) => {
    const data = await request(`/admin/room-types?${query({ q: s.q, page: 1, page_size: 20 })}`);
    s.items = data.items || [];
    s.page = data;
    s.roomId = selectedRoom(s);
    s.room = s.items.find((item) => item.id === s.roomId);
    s.rates = s.roomId ? (await request(`/admin/room-types/${encodeURIComponent(s.roomId)}/rates`)).items || [] : [];
    s.rate = s.rates.find((item) => item.id === s.rateId) || null;
  };
  const loadInventory = async (s) => {
    const rooms = await request("/admin/room-types?page=1&page_size=100");
    s.rooms = rooms.items || [];
    s.roomId = selectedRoom({ ...s, items: s.rooms });
    s.data = s.roomId ? await request(`/admin/room-types/${encodeURIComponent(s.roomId)}/inventory?${query({ from: s.from, to: s.to })}`) : null;
  };
  const loadCampaigns = async (s) => {
    const data = await request(`/admin/campaigns?${query({ q: s.q, page: 1, page_size: 20 })}`);
    s.items = data.items || [];
    s.page = data;
    s.campaignId = selectedCampaign(s);
    if (s.campaignId) {
      s.campaign = (await request(`/admin/campaigns/${encodeURIComponent(s.campaignId)}`)).campaign;
      s.versions = await request(`/admin/campaigns/${encodeURIComponent(s.campaignId)}/versions?page=1&page_size=20`);
    } else { s.campaign = null; s.versions = null; }
  };
  const loadConfiguredOrder = async (s) => {
    s.orders = await request("/admin/bookings?page=1&page_size=20");
    if (!s.orderId) return;
    s.detail = await request(`/admin/bookings/${encodeURIComponent(s.orderId)}`);
    s.options = await request(`/admin/bookings/${encodeURIComponent(s.orderId)}/benefit-options`);
  };
  const loadBenefits = async (s) => {
    const filters = { ...s.filters, page: s.page || 1, page_size: 20 };
    const eventFilters = { ...filters };
    delete eventFilters.state;
    const [benefits, events] = await Promise.all([
      request(`/admin/benefits?${query(filters)}`),
      request(`/admin/benefit-events?${query(eventFilters)}`)
    ]);
    s.benefits = benefits; s.events = events;
  };
  const loadLogs = async (s) => { s.data = await request(operationLogPath(s.filters, s.page)); };
  const loaders = { rooms: loadRooms, inventory: loadInventory, campaigns: loadCampaigns,
    configuredOrders: loadConfiguredOrder, benefits: loadBenefits, operationLogs: loadLogs };
  const refresh = (tab) => run(tab, loaders[tab]);
  const notice = (message) => { notify(message); render(); };
  const save = (tab, task, message) => run(tab, async (s) => { await task(s); notice(message); await loaders[tab](s); });

  function html(tab) {
    const s = state[tab];
    const heading = newTabs.find(([key]) => key === tab)?.[1];
    const common = `<div class="section-heading"><h2>${heading}</h2>${button("刷新状态", "refresh")}</div>${s.error ? `<div class="op-error" role="alert">${esc(s.error)} ${button("刷新", "refresh")}</div>` : ""}`;
    let body = "";
    if (tab === "rooms") body = `
      <form data-op-form="room-search" class="op-inline">${field("q", "搜索房型", s.q)}<button type="submit">查询</button></form>
      <div class="op-layout"><section><h3>房型列表</h3>${(s.items || []).map((r) => button(`${esc(r.name)} · ${esc(r.status)}`, "room-select", `data-id="${esc(r.id)}"`)).join(" ")}${paged(s.page)}</section>
      <section><h3>房型配置</h3><form data-op-form="room-save" class="form-grid">
      ${field("name", "名称", s.room?.name)}${field("sort_order", "排序", s.room?.sort_order ?? 0, "number")}${field("default_sellable_quantity", "默认可售间数", s.room?.default_sellable_quantity ?? 0, "number")}
      ${select("status", "状态", [["draft", "草稿"], ["active", "开售"], ["paused", "暂停"]], s.room?.status || "draft")}
      <label><span>房型详情 JSON</span><textarea name="details" rows="7">${esc(JSON.stringify(s.room?.details || { photo_urls: [], beds: [], amenities: [] }, null, 2))}</textarea></label>
      <button type="submit">${s.room ? "保存房型" : "新增房型"}</button></form>
      <h3>价格方案</h3>${list(s.rates, [["name", "名称"], ["status", "状态"], ["base_nightly_cent", "每晚分"], ["version", "版本"]])}
      ${(s.rates || []).map((r) => button(`编辑 ${esc(r.name)}`, "rate-select", `data-id="${esc(r.id)}"`)).join(" ")}
      ${s.room ? `<form data-op-form="rate-save" class="op-inline">${field("name", "名称", s.rate?.name)}${field("base_nightly_cent", "每晚分", s.rate?.base_nightly_cent ?? 0, "number")}${field("breakfast_count", "早餐份数", s.rate?.breakfast_count ?? 0, "number")}${field("cancellation_text", "退订规则", s.rate?.cancellation_text)}${s.rate ? select("status", "状态", [["draft", "草稿"], ["active", "开售"], ["paused", "暂停"]], s.rate.status) : ""}<button type="submit">${s.rate ? "保存价格方案" : "新增价格方案"}</button>${s.rate ? button("取消编辑", "rate-clear") : ""}</form>` : ""}</section></div>`;
    if (tab === "inventory") body = `<form data-op-form="inventory-query" class="op-inline">${select("roomId", "房型", (s.rooms || []).map((r) => [r.id, r.name]), s.roomId)}${field("from", "开始日期", s.from, "date")}${field("to", "结束日期", s.to, "date")}<button type="submit">查询日库存</button></form>
      ${s.data ? `<form data-op-form="inventory-save"><div class="op-table-wrap"><table class="op-table"><thead><tr><th>住宿日</th><th>可售覆盖量</th><th>有效量</th><th>已占</th><th>关闭</th></tr></thead><tbody>${(s.data.items || []).map((item) => `<tr><td>${esc(item.stay_date)}</td><td><input type="number" min="0" name="quantity:${esc(item.stay_date)}" value="${esc(item.sellable_quantity ?? "")}" placeholder="继承默认"></td><td>${esc(item.effective_quantity)}</td><td>${esc(item.reserved_quantity)}</td><td><input type="checkbox" name="closed:${esc(item.stay_date)}" ${item.closed ? "checked" : ""}></td></tr>`).join("")}</tbody></table></div><button type="submit">保存日库存</button></form>` : ""}`;
    if (tab === "campaigns") body = `<form data-op-form="campaign-search" class="op-inline">${field("q", "搜索活动", s.q)}<button type="submit">查询</button></form>
      <div class="op-layout"><section><h3>活动列表</h3>${(s.items || []).map((c) => button(`${esc(c.rule?.name || c.name)} · ${esc(c.status)}`, "campaign-select", `data-id="${esc(c.id)}"`)).join(" ")}${paged(s.page)}</section>
      <section><h3>${s.campaign ? "活动配置与预览" : "新建活动"}</h3><form data-op-form="campaign-save"><label><span>活动规则 JSON</span><textarea name="rule" rows="19">${esc(JSON.stringify(s.campaign?.rule || defaultRule(), null, 2))}</textarea></label><button type="submit">${s.campaign ? "保存新版本" : "创建草稿"}</button></form>
      ${s.campaign ? `<p>版本 ${esc(s.campaign.version)} · ${esc(s.campaign.status)} · 已发 ${esc(s.campaign.issued_count)}</p><div class="op-inline">${button("预览当前版本", "campaign-preview")}${button("启用", "campaign-active")}${button("暂停", "campaign-paused")}</div><h3>历史版本</h3>${list(s.versions?.items, [["version", "版本"], ["created_at", "创建时间"], ["status", "状态"]])}</section></div>` : ""}`;
    if (tab === "configuredOrders") body = `<form data-op-form="order-query" class="op-inline">${field("orderId", "订单 ID", s.orderId)}<button type="submit">查看订单</button></form><form data-op-form="booking-export" class="op-inline">${field("from", "导出开始日", s.exportFilters?.from, "date")}${field("to", "导出结束日", s.exportFilters?.to, "date")}${field("status", "状态", s.exportFilters?.status)}${field("room_type_id", "房型 ID", s.exportFilters?.room_type_id)}${field("order_id", "订单 ID", s.exportFilters?.order_id)}<button type="submit">导出订单 CSV</button></form><h3>近期订单</h3>${list(s.orders?.items, [["id", "订单 ID"], ["status", "状态"], ["checkin_date", "入住日"], ["checkout_date", "离店日"]])}${paged(s.orders)}${s.detail ? `<h3>订单详情</h3>${json(s.detail)}<h3>可发入住礼</h3><form data-op-form="order-checkin">${(s.options?.eligible || []).map((item) => `<label class="op-choice"><input type="checkbox" name="gift" value="${esc(item.campaign_id)}:${esc(item.campaign_version)}">${esc(item.name)} · ${esc(item.display_value)}</label>`).join("")}<p>应付 ${esc(s.options?.amount_due_cent)} 分</p><p>抵扣与实物交付</p>${json({ deductions: s.options?.deductions, fulfillment_items: s.options?.fulfillment_items })}<button type="submit" ${s.detail.order?.status !== "booked" ? "disabled" : ""}>确认入住、核销并发礼</button></form>${s.result ? `<h3>本次处理结果</h3>${json(s.result)}` : ""}` : ""}`;
    if (tab === "benefits") body = `<form data-op-form="benefit-query" class="op-inline">${["member_id", "order_id", "campaign_id", "kind", "state", "from", "to"].map((key) => field(key, key, s.filters?.[key], key === "from" || key === "to" ? "date" : "text")).join("")}<button type="submit">查询明细</button></form><form data-op-form="event-export" class="op-inline">${field("from", "导出开始日", s.exportFilters?.from, "date")}${field("to", "导出结束日", s.exportFilters?.to, "date")}<button type="submit">导出权益流水 CSV</button></form><h3>权益</h3>${list(s.benefits?.items, [["id", "ID"], ["member_id", "会员"], ["campaign_id", "活动"], ["kind", "类型"], ["state", "状态"], ["remaining_cent", "剩余分"], ["remaining_quantity", "剩余件"]])}${paged(s.benefits)}<h3>核销与权益流水</h3>${list(s.events?.items, [["id", "ID"], ["grant_id", "权益"], ["order_id", "订单"], ["action", "动作"], ["delta_cent", "金额分"], ["delta_quantity", "数量"], ["created_at", "时间"]])}${paged(s.events)}`;
    if (tab === "operationLogs") body = `<form data-op-form="log-query" class="op-inline">${["from", "to", "username", "client_ip", "action", "outcome", "target_type", "target_id", "request_id"].map((key) => field(key, key, s.filters?.[key], key === "from" || key === "to" ? "date" : "text")).join("")}<button type="submit">查询日志</button></form>${list(s.data?.items, [["occurred_at", "时间"], ["username", "员工"], ["action", "操作"], ["target_type", "对象"], ["target_id", "对象 ID"], ["outcome", "结果"], ["http_status", "HTTP"], ["reason_code", "原因"]])}${(s.data?.items || []).map((item) => `<details><summary>${esc(item.occurred_at)} · ${esc(item.action)} · ${esc(item.request_id)}</summary>${json(item)}</details>`).join("")}<div class="op-inline">${button("上一页", "log-prev", s.page <= 1 ? "disabled" : "")}${paged(s.data)}${button("下一页", "log-next", s.page * 20 >= (s.data?.total || 0) ? "disabled" : "")}</div>`;
    return `<section class="operations">${common}${s.loading ? `<p class="op-muted">正在读取...</p>` : ""}${body}</section>`;
  }

  function bind(tab) {
    if (!state[tab]) return;
    const root = document.querySelector(".operations");
    root?.addEventListener("click", async (event) => {
      const target = event.target.closest("[data-op]");
      if (!target) return;
      const s = state[tab];
      const action = target.dataset.op;
      if (action === "refresh") return refresh(tab);
      if (action === "room-select") { s.roomId = target.dataset.id; return refresh(tab); }
      if (action === "rate-select") { s.rateId = target.dataset.id; return refresh(tab); }
      if (action === "rate-clear") { s.rateId = null; s.rate = null; return render(); }
      if (action === "campaign-select") { s.campaignId = target.dataset.id; return refresh(tab); }
      if (action === "campaign-preview") return run(tab, async () => { s.campaign = (await request(`/admin/campaigns/${encodeURIComponent(s.campaignId)}`)).campaign; notice("已读取最新活动版本"); });
      if (action === "campaign-active" || action === "campaign-paused") {
        if (!confirm(`确认${action === "campaign-active" ? "启用" : "暂停"}活动？`)) return;
        return save(tab, () => request(`/admin/campaigns/${encodeURIComponent(s.campaignId)}/state`, { method: "POST", body: { version: s.campaign.version, status: action === "campaign-active" ? "active" : "paused" } }), "活动状态已更新");
      }
      if (action === "log-prev" || action === "log-next") { s.page += action === "log-next" ? 1 : -1; return refresh(tab); }
    });
    root?.addEventListener("submit", async (event) => {
      const form = event.target.closest("[data-op-form]");
      if (!form) return;
      event.preventDefault();
      const action = form.dataset.opForm;
      const s = state[tab];
      const values = Object.fromEntries(new FormData(form));
      if (action === "room-search" || action === "campaign-search") { s.q = values.q; return refresh(tab); }
      if (action === "inventory-query") { Object.assign(s, values); return refresh(tab); }
      if (action === "order-query") { s.orderId = values.orderId.trim(); s.detail = null; s.options = null; return refresh(tab); }
      if (action === "benefit-query" || action === "log-query") { s.filters = values; s.page = 1; return refresh(tab); }
      if (action === "booking-export" || action === "event-export") {
        s.exportFilters = values;
        return run(tab, async () => {
          const kind = action === "booking-export" ? "bookings" : "benefit-events";
          await downloadCsv(exportPath(kind, values), `${kind}-${values.from}-${values.to}.csv`);
          notice("CSV 已导出");
        });
      }
      if (action === "room-save") return save(tab, () => {
        const body = { name: values.name, sort_order: Number(values.sort_order), default_sellable_quantity: Number(values.default_sellable_quantity), details: JSON.parse(values.details) };
        if (s.room) return request(`/admin/room-types/${encodeURIComponent(s.room.id)}`, { method: "PATCH", body: { ...body, status: values.status, version: s.room.version } });
        return request("/admin/room-types", { method: "POST", body });
      }, "房型已保存");
      if (action === "rate-save") return save(tab, () => {
        const body = { name: values.name, base_nightly_cent: Number(values.base_nightly_cent), breakfast_count: Number(values.breakfast_count), cancellation_text: values.cancellation_text };
        if (s.rate) return request(`/admin/room-rates/${encodeURIComponent(s.rate.id)}`, { method: "PATCH", body: { ...body, version: s.rate.version, status: values.status } });
        return request(`/admin/room-types/${encodeURIComponent(s.roomId)}/rates`, { method: "POST", body });
      }, "价格方案已保存");
      if (action === "inventory-save") {
        const items = (s.data?.items || []).map((item) => ({ stay_date: item.stay_date, sellable_quantity: values[`quantity:${item.stay_date}`] === "" ? null : Number(values[`quantity:${item.stay_date}`]), closed: values[`closed:${item.stay_date}`] === "on" }));
        if (!confirm(`确认保存 ${items.length} 天的库存？`)) return;
        return save(tab, () => request(`/admin/room-types/${encodeURIComponent(s.roomId)}/inventory`, { method: "PUT", body: { version: s.data.version, from: s.from, to: s.to, items } }), "日库存已保存");
      }
      if (action === "campaign-save") return save(tab, () => {
        const rule = JSON.parse(values.rule);
        return s.campaign ? request(`/admin/campaigns/${encodeURIComponent(s.campaignId)}`, { method: "PATCH", body: { version: s.campaign.version, rule } }) : request("/admin/campaigns", { method: "POST", body: rule });
      }, "活动配置已保存");
      if (action === "order-checkin") {
        if (!confirm("确认办理入住？将同时核销锁定权益并发放所选入住礼。")) return;
        const selected_campaigns = new FormData(form).getAll("gift").map((value) => { const [campaign_id, campaign_version] = value.split(":"); return { campaign_id, campaign_version: Number(campaign_version) }; });
        return run(tab, async () => { s.result = await request(`/admin/bookings/${encodeURIComponent(s.orderId)}/configured-status`, { method: "PATCH", body: { expected_status: "booked", status: "checked_in", selected_campaigns } }); notice("入住已确认，核销及发礼结果已返回"); await loadConfiguredOrder(s); });
      }
    });
  }
  return { tabs: newTabs, hasTab: (tab) => Boolean(state[tab]), view: refresh, html, bind };
}
