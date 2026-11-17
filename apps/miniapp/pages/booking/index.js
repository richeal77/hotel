const { dayAfterTomorrow, tomorrow } = require("../../utils/format");
const { phoneForBooking } = require("../../utils/defaults");
const session = require("../../utils/session");

Page({
  data: {
    checkinDate: tomorrow(),
    checkoutDate: dayAfterTomorrow(),
    roomTypes: ["标准大床房", "舒适双床房", "家庭房", "棋牌房"],
    roomIndex: 0,
    arrivalTimes: ["18:00", "20:00", "22:00", "次日 00:00"],
    arrivalIndex: 1,
    contactName: "张三",
    phone: "",
    remark: "想要安静房间",
    createdOrder: null,
    apiModeText: "HTTP 接口（验收）"
  },

  onLoad() {
    this.syncBoundPhone();
  },

  onShow() {
    this.syncBoundPhone();
  },

  syncBoundPhone() {
    const authMember = session.getAuthMember();
    this.setData({
      phone: phoneForBooking(authMember),
      apiModeText: getApp().getRuntimeStatus().apiModeText
    });
  },

  onInput(event) {
    const field = event.currentTarget.dataset.field;
    this.setData({
      [field]: event.detail.value
    });
  },

  onCheckinChange(event) {
    this.setData({
      checkinDate: event.detail.value
    });
  },

  onCheckoutChange(event) {
    this.setData({
      checkoutDate: event.detail.value
    });
  },

  onRoomChange(event) {
    this.setData({
      roomIndex: Number(event.detail.value)
    });
  },

  onArrivalChange(event) {
    this.setData({
      arrivalIndex: Number(event.detail.value)
    });
  },

  validate() {
    if (!this.data.phone) {
      wx.showToast({
        title: "请先绑定手机号",
        icon: "none"
      });
      return false;
    }

    if (!this.data.contactName || !/^1\d{10}$/.test(this.data.phone)) {
      wx.showToast({
        title: "请填写联系人和手机号",
        icon: "none"
      });
      return false;
    }

    if (this.data.checkoutDate <= this.data.checkinDate) {
      wx.showToast({
        title: "离店日期需晚于入住日期",
        icon: "none"
      });
      return false;
    }

    return true;
  },

  async submitBooking() {
    if (!this.validate()) {
      return;
    }

    try {
      const app = getApp();
      await app.ensureLogin();
      const result = await app.globalData.api.createBooking({
        checkin_date: this.data.checkinDate,
        checkout_date: this.data.checkoutDate,
        room_type: this.data.roomTypes[this.data.roomIndex],
        arrival_time: this.data.arrivalTimes[this.data.arrivalIndex],
        contact_name: this.data.contactName,
        phone: this.data.phone,
        remark: this.data.remark
      });

      this.setData({
        createdOrder: result.order
      });
      wx.showToast({
        title: result.order.status_text,
        icon: "success"
      });
    } catch (error) {
      wx.showToast({
        title: error.message || "提交失败",
        icon: "none"
      });
    }
  },

  goOrders() {
    wx.navigateTo({
      url: "/pages/orders/index"
    });
  }
});
