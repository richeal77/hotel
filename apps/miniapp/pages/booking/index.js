const { tomorrow, dayAfterTomorrow, money } = require("../../utils/format");
const { phoneForBooking } = require("../../utils/defaults");
const { LOCAL_DEMO_MODE } = require("../../utils/settings");
const { newKey, needsNewQuote } = require("../../utils/booking-flow");
const session = require("../../utils/session");

Page({
  data: {
    checkinDate: tomorrow(), checkoutDate: dayAfterTomorrow(),
    roomTypes: ["标准大床房", "舒适双床房", "家庭房", "棋牌房"], roomIndex: 0,
    arrivalTimes: ["18:00", "20:00", "22:00", "次日 00:00"], arrivalIndex: 1,
    contactName: "", phone: "", remark: "", createdOrder: null,
    apiModeText: "", localDemo: false, bookingEnabled: false, loading: false,
    submitting: false, errorText: "", rooms: [], selectedRateId: "", quote: null,
    quoteAmountText: "", quoteChanged: false, submitKey: ""
  },
  onLoad() { this.syncBoundPhone(); },
  async onShow() {
    this.syncBoundPhone();
    const availability = this.loadAvailability();
    if (!this.data.localDemo) {
      try { await getApp().ensureLogin(); this.syncBoundPhone(); }
      catch (error) { this.setData({ errorText: error.message || "登录失败" }); }
    }
    return availability;
  },
  syncBoundPhone() {
    const member = session.getAuthMember();
    this.setData({ phone: phoneForBooking(member), apiModeText: getApp().getRuntimeStatus().apiModeText,
      localDemo: getApp().getRuntimeStatus().apiMode === LOCAL_DEMO_MODE });
  },
  onInput(event) { this.setData({ [event.currentTarget.dataset.field]: event.detail.value }); },
  onCheckinChange(event) { this.setData({ checkinDate: event.detail.value, quote: null }); this.loadAvailability(); },
  onCheckoutChange(event) { this.setData({ checkoutDate: event.detail.value, quote: null }); this.loadAvailability(); },
  onRoomChange(event) { this.setData({ roomIndex: Number(event.detail.value) }); },
  onArrivalChange(event) { this.setData({ arrivalIndex: Number(event.detail.value) }); },
  async loadAvailability() {
    if (this.data.localDemo) return;
    this.setData({ loading: true, errorText: "", quote: null, rooms: [], selectedRateId: "" });
    try {
      const api = getApp().globalData.api;
      const capabilities = await api.getCapabilities();
      if (!capabilities.configured_booking) {
        this.setData({ bookingEnabled: false, errorText: "当前暂不开放线上预订，请联系前台" });
        return;
      }
      if (this.data.checkoutDate <= this.data.checkinDate) throw new Error("离店日期需晚于入住日期");
      const data = await api.getRoomTypes(this.data.checkinDate, this.data.checkoutDate);
      this.setData({ bookingEnabled: Boolean(data.configured_booking),
        errorText: data.configured_booking ? "" : "当前暂不开放线上预订，请联系前台",
        rooms: (data.items || []).map((room) => ({
        ...room, rates: (room.rates || []).map((rate) => ({ ...rate, priceText: money(rate.nightly_cent) }))
      })) });
    } catch (error) { this.setData({ bookingEnabled: false, errorText: error.message || "房型加载失败" }); }
    finally { this.setData({ loading: false }); }
  },
  selectRate(event) {
    if (Number(event.currentTarget.dataset.available) <= 0) return;
    this.setData({ selectedRateId: event.currentTarget.dataset.rateId, quote: null, quoteChanged: false });
  },
  async requestQuote() {
    if (!this.data.selectedRateId || !this.data.bookingEnabled) return;
    this.setData({ loading: true, errorText: "", quote: null });
    const selection = `${this.data.selectedRateId}:${this.data.checkinDate}:${this.data.checkoutDate}`;
    try {
      const app = getApp();
      await app.ensureLogin();
      const data = await app.globalData.api.createQuote({ rate_id: this.data.selectedRateId,
        checkin_date: this.data.checkinDate, checkout_date: this.data.checkoutDate }, newKey());
      if (selection !== `${this.data.selectedRateId}:${this.data.checkinDate}:${this.data.checkoutDate}`) return;
      this.setData({ quote: data.quote, quoteAmountText: money(data.quote.amount_due_cent), quoteChanged: false, submitKey: newKey() });
    } catch (error) { this.setData({ errorText: error.message || "报价失败" }); }
    finally { this.setData({ loading: false }); }
  },
  validate() {
    if (!this.data.phone) { wx.showToast({ title: "请先绑定手机号", icon: "none" }); return false; }
    if (!this.data.contactName || !/^1\d{10}$/.test(this.data.phone)) {
      wx.showToast({ title: "请填写联系人和手机号", icon: "none" }); return false;
    }
    if (this.data.checkoutDate <= this.data.checkinDate) {
      wx.showToast({ title: "离店日期需晚于入住日期", icon: "none" }); return false;
    }
    return true;
  },
  async submitBooking() {
    if (this.data.submitting || !this.validate()) return;
    if (!this.data.localDemo && (!this.data.bookingEnabled || !this.data.quote)) return;
    if (!this.data.localDemo && Date.parse(this.data.quote.expires_at) <= Date.now()) {
      this.setData({ quote: null, errorText: "报价已过期，请重新获取并确认" });
      return;
    }
    this.setData({ submitting: true, errorText: "" });
    try {
      const app = getApp();
      await app.ensureLogin();
      const result = this.data.localDemo
        ? await app.globalData.api.createBooking({ checkin_date: this.data.checkinDate,
          checkout_date: this.data.checkoutDate, room_type: this.data.roomTypes[this.data.roomIndex],
          arrival_time: this.data.arrivalTimes[this.data.arrivalIndex], contact_name: this.data.contactName,
          phone: this.data.phone, remark: this.data.remark })
        : await app.globalData.api.submitConfiguredBooking({ quote_id: this.data.quote.id,
          arrival_time: this.data.arrivalTimes[this.data.arrivalIndex], contact_name: this.data.contactName,
          phone: this.data.phone, remark: this.data.remark }, this.data.submitKey);
      this.setData({ createdOrder: result.order, quote: null });
      wx.showToast({ title: this.data.localDemo ? result.order.status_text : "预订成功，待前台确认", icon: "success" });
    } catch (error) {
      if (!this.data.localDemo && needsNewQuote(error)) {
        await this.loadAvailability();
        this.setData({ quoteChanged: true, errorText: "报价或房量已变化，请重新选择房型、获取报价并确认" });
      } else this.setData({ errorText: error.message || "提交失败" });
    } finally { this.setData({ submitting: false }); }
  },
  goOrders() { wx.navigateTo({ url: "/pages/orders/index" }); },
  goJoin() { wx.navigateTo({ url: "/pages/join/index" }); }
});
