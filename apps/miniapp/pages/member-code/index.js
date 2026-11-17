const session = require("../../utils/session");

function maskPhone(phone) {
  if (!phone || phone.length < 7) {
    return phone || "未绑定";
  }

  return `${phone.slice(0, 3)}****${phone.slice(-4)}`;
}

Page({
  data: {
    loading: true,
    member: {},
    memberCode: "",
    maskedPhone: "未绑定",
    sceneText: "",
    apiModeText: "HTTP 接口（验收）"
  },

  onLoad() {
    this.loadMember();
  },

  onShow() {
    this.setData({
      apiModeText: getApp().getRuntimeStatus().apiModeText
    });
  },

  async loadMember() {
    this.setData({ loading: true });

    try {
      const app = getApp();
      await app.ensureLogin();
      const data = await app.globalData.api.getMe();
      const member = data.member || {};
      const context = session.getLaunchContext();

      this.setData({
        member,
        memberCode: data.member_code || member.id || "",
        maskedPhone: maskPhone(member.phone),
        sceneText: `${context.source} / ${context.scene}`,
        apiModeText: getApp().getRuntimeStatus().apiModeText
      });
    } catch (error) {
      wx.showToast({
        title: error.message || "会员码加载失败",
        icon: "none"
      });
    } finally {
      this.setData({ loading: false });
    }
  },

  copyCode() {
    wx.setClipboardData({
      data: this.data.memberCode
    });
  },

  goJoin() {
    wx.navigateTo({
      url: "/pages/join/index"
    });
  }
});
