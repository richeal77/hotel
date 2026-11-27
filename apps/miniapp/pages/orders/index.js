const { money } = require("../../utils/format");

const STATUS_TEXT = {
  pending_confirm: "待前台确认",
  booked: "待前台确认",
  confirmed: "已确认",
  checked_in: "已入住",
  checked_out: "已离店",
  cancelled: "已取消"
};

Page({
  data: {
    loading: true,
    apiModeText: "HTTP 接口（验收）",
    orders: []
  },

  onLoad() {
    this.loadOrders();
  },

  onShow() {
    this.setData({
      apiModeText: getApp().getRuntimeStatus().apiModeText
    });
    this.loadOrders();
  },

  onPullDownRefresh() {
    this.loadOrders().finally(() => wx.stopPullDownRefresh());
  },

  async loadOrders() {
    this.setData({ loading: true });

    try {
      const app = getApp();
      await app.ensureLogin();
      const data = await app.globalData.api.getBookings();
      const orders = (data.items || []).map((order) => ({
        ...order,
        statusText: order.status_text || STATUS_TEXT[order.status] || order.status,
        amountText: money(order.amount_due_cent ?? order.actual_amount_cent ?? order.estimated_amount_cent ?? 0),
        roomName: order.booking_snapshot ? order.booking_snapshot.room_name : order.room_type
      }));

      this.setData({
        orders,
        apiModeText: app.getRuntimeStatus().apiModeText
      });
    } catch (error) {
      wx.showToast({
        title: error.message || "订单加载失败",
        icon: "none"
      });
    } finally {
      this.setData({ loading: false });
    }
  },

  goBooking() {
    wx.navigateTo({
      url: "/pages/booking/index"
    });
  }
});
