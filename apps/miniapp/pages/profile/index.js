const { money } = require("../../utils/format");
const session = require("../../utils/session");
const { HTTP_MODE, LOCAL_DEMO_MODE, LIVE_MODE } = require("../../utils/settings");

Page({
  data: {
    loading: true,
    member: {},
    memberCode: "",
    context: {},
    apiModeText: "HTTP 接口（验收）",
    apiMode: HTTP_MODE,
    apiBaseUrl: "",
    apiBaseInput: "",
    liveApiBaseInput: "",
    localDemoEnabled: false,
    liveEnabled: false,
    walletText: "¥0.00",
    couponCount: 0,
    bookingCount: 0
  },

  onLoad() {
    this.syncRuntimeSettings();
    this.loadProfile();
  },

  onShow() {
    this.syncRuntimeSettings();
  },

  syncRuntimeSettings() {
    const runtime = getApp().getRuntimeStatus();
    this.setData({
      context: session.getLaunchContext(),
      apiMode: runtime.apiMode,
      apiModeText: runtime.apiModeText,
      apiBaseUrl: runtime.apiBaseUrl,
      apiBaseInput: session.getHttpApiBaseUrl(),
      liveApiBaseInput: session.getLiveApiBaseUrl(),
      localDemoEnabled: runtime.apiMode === LOCAL_DEMO_MODE,
      liveEnabled: runtime.apiMode === LIVE_MODE
    });
  },

  onPullDownRefresh() {
    this.loadProfile().finally(() => wx.stopPullDownRefresh());
  },

  async loadProfile() {
    this.setData({ loading: true });

    try {
      const app = getApp();
      await app.ensureLogin();
      const me = await app.globalData.api.getMe();
      const home = await app.globalData.api.getHome();

      this.setData({
        member: me.member,
        memberCode: me.member_code,
        context: session.getLaunchContext(),
        apiMode: app.getRuntimeStatus().apiMode,
        apiModeText: app.getRuntimeStatus().apiModeText,
        apiBaseUrl: app.getRuntimeStatus().apiBaseUrl,
        apiBaseInput: session.getHttpApiBaseUrl(),
        liveApiBaseInput: session.getLiveApiBaseUrl(),
        localDemoEnabled: app.getRuntimeStatus().apiMode === LOCAL_DEMO_MODE,
        liveEnabled: app.getRuntimeStatus().apiMode === LIVE_MODE,
        walletText: money(home.wallet_balance_cent),
        couponCount: home.coupon_count,
        bookingCount: home.booking_count
      });
    } catch (error) {
      this.setData({ member: {}, memberCode: "", walletText: "--", couponCount: 0, bookingCount: 0 });
      wx.showToast({
        title: error.message || "我的页面加载失败",
        icon: "none"
      });
    } finally {
      this.setData({ loading: false });
    }
  },

  refreshLogin() {
    getApp().refreshLogin()
      .then(() => this.loadProfile())
      .catch((error) => {
        wx.showToast({
          title: error.message || "登录失败",
          icon: "none"
        });
      });
  },

  onInput(event) {
    const field = event.currentTarget.dataset.field;
    this.setData({
      [field]: event.detail.value
    });
  },

  saveApiBaseUrl() {
    try {
      const normalized = session.setApiBaseUrl(this.data.apiBaseInput);
      getApp().applyRuntimeSettings();
      this.syncRuntimeSettings();
      wx.showToast({
        title: "API 地址已保存",
        icon: "success"
      });
      this.setData({
        apiBaseInput: normalized
      });
    } catch (error) {
      wx.showToast({
        title: error.message || "API 地址无效",
        icon: "none"
      });
    }
  },

  saveLiveApiBaseUrl() {
    try {
      session.setLiveApiBaseUrl(this.data.liveApiBaseInput);
      getApp().applyRuntimeSettings();
      this.syncRuntimeSettings();
      if (this.data.liveEnabled) this.loadProfile();
      wx.showToast({ title: "真实接口域名已保存", icon: "success" });
    } catch (error) {
      wx.showToast({ title: error.message || "域名无效", icon: "none" });
    }
  },

  toggleLocalDemo(event) {
    const nextMode = event.detail.value ? LOCAL_DEMO_MODE : HTTP_MODE;
    session.setApiMode(nextMode);
    getApp().applyRuntimeSettings();
    this.syncRuntimeSettings();
    this.loadProfile();
    wx.showToast({
      title: nextMode === LOCAL_DEMO_MODE ? "已开启本地演示" : "已切回 HTTP",
      icon: "none"
    });
  },

  toggleLive(event) {
    try {
      session.setApiMode(event.detail.value ? LIVE_MODE : HTTP_MODE);
      getApp().applyRuntimeSettings();
      this.syncRuntimeSettings();
      this.loadProfile();
    } catch (error) {
      this.syncRuntimeSettings();
      wx.showToast({ title: error.message || "切换失败", icon: "none" });
    }
  },

  goPage(event) {
    wx.navigateTo({
      url: event.currentTarget.dataset.url
    });
  }
});
