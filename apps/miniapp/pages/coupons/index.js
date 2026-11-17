const { shortDate } = require("../../utils/format");

const STATUS_TEXT = {
  available: "可用",
  used: "已使用",
  expired: "已过期",
  voided: "已作废"
};

Page({
  data: {
    loading: true,
    apiModeText: "HTTP 接口（验收）",
    activeStatus: "",
    filters: [
      { value: "", label: "全部" },
      { value: "available", label: "可用" },
      { value: "used", label: "已使用" },
      { value: "expired", label: "已过期" }
    ],
    coupons: []
  },

  onLoad() {
    this.loadCoupons();
  },

  onShow() {
    this.setData({
      apiModeText: getApp().getRuntimeStatus().apiModeText
    });
  },

  onPullDownRefresh() {
    this.loadCoupons().finally(() => wx.stopPullDownRefresh());
  },

  selectStatus(event) {
    this.setData({
      activeStatus: event.currentTarget.dataset.status
    });
    this.loadCoupons();
  },

  async loadCoupons() {
    this.setData({ loading: true });

    try {
      const app = getApp();
      await app.ensureLogin();
      const data = await app.globalData.api.getCoupons(this.data.activeStatus);
      const coupons = (data.items || []).map((coupon) => ({
        ...coupon,
        statusText: STATUS_TEXT[coupon.status] || coupon.status,
        expireText: shortDate(coupon.expire_at),
        receivedText: shortDate(coupon.received_at)
      }));

      this.setData({
        coupons,
        apiModeText: app.getRuntimeStatus().apiModeText
      });
    } catch (error) {
      wx.showToast({
        title: error.message || "优惠券加载失败",
        icon: "none"
      });
    } finally {
      this.setData({ loading: false });
    }
  },

  goJoin() {
    wx.navigateTo({
      url: "/pages/join/index"
    });
  }
});
