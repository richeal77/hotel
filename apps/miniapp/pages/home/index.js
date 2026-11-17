const config = require("../../utils/config");
const { money } = require("../../utils/format");
const session = require("../../utils/session");

Page({
  data: {
    loading: true,
    apiMode: "http",
    apiModeText: "HTTP 接口（验收）",
    context: {},
    hotelName: config.hotelName,
    hotelAddress: "",
    hotelPhone: "",
    hotelTagsText: "",
    priceText: "¥0.00",
    walletText: "¥0.00",
    couponCount: 0,
    bookingCount: 0,
    referralCount: 0,
    rewardText: "¥0.00",
    isPhoneBound: false
  },

  onLoad(options) {
    getApp().capturePageQuery(options);
  },

  onShow() {
    return this.loadHome();
  },

  onPullDownRefresh() {
    this.loadHome().finally(() => wx.stopPullDownRefresh());
  },

  syncRuntimeState() {
    const app = getApp();
    const runtime = app.getRuntimeStatus();
    const authMember = session.getAuthMember();
    this.setData({
      apiMode: runtime.apiMode,
      apiModeText: runtime.apiModeText,
      context: app.globalData.launchContext,
      isPhoneBound: Boolean(authMember && authMember.is_phone_bound)
    });
  },

  async loadHome() {
    this.syncRuntimeState();
    this.setData({ loading: true });

    try {
      const app = getApp();
      await app.ensureLogin();
      const home = await app.globalData.api.getHome();
      const tags = home.hotel && home.hotel.tags ? home.hotel.tags : [];

      this.setData({
        hotelName: home.hotel ? home.hotel.name : config.hotelName,
        hotelAddress: home.hotel ? home.hotel.address : "",
        hotelPhone: home.hotel ? home.hotel.phone : "",
        hotelTagsText: tags.join(" / "),
        priceText: money(home.today_member_price_cent),
        walletText: money(home.wallet_balance_cent),
        couponCount: home.coupon_count || 0,
        bookingCount: home.booking_count || 0,
        referralCount: home.referral_summary ? home.referral_summary.referred_count : 0,
        rewardText: money(home.referral_summary ? home.referral_summary.available_reward_cent : 0)
      });
      this.syncRuntimeState();
    } catch (error) {
      wx.showToast({
        title: error.message || "首页加载失败",
        icon: "none"
      });
    } finally {
      this.setData({ loading: false });
    }
  },

  goPage(event) {
    const url = event.currentTarget.dataset.url;
    wx.navigateTo({ url });
  }
});
