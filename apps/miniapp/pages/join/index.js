const config = require("../../utils/config");
const { guestPhoneForContext } = require("../../utils/defaults");
const { validateJoinResult } = require("../../utils/join");
const session = require("../../utils/session");

Page({
  data: {
    apiModeText: "HTTP 接口（验收）",
    phone: "13800138901",
    phoneHint: "前台验收手机号：13800138901",
    code: "666666",
    nickname: "张三",
    source: "frontdesk",
    scene: "frontdesk_qr",
    referrerMemberId: "",
    acceptedPrivacy: true,
    acceptedMemberRule: true,
    giftChoice: "laundry",
    gifts: [
      { value: "laundry", label: "洗衣券 1 次" },
      { value: "late_checkout", label: "延迟退房券" },
      { value: "next_stay_5", label: "下次入住立减 5 元" }
    ],
    sentCode: false,
    joined: false,
    issuedCoupons: []
  },

  onLoad(options) {
    const app = getApp();
    app.capturePageQuery(options);
    const context = app.globalData.launchContext;
    const phone = guestPhoneForContext(context);
    this.setData({
      apiModeText: app.getRuntimeStatus().apiModeText,
      phone,
      phoneHint: phone === "13800138902" ? "朋友推荐示例手机号：13800138902" : "前台验收手机号：13800138901",
      source: context.source,
      scene: context.scene,
      referrerMemberId: context.referrerMemberId
    });
  },

  onShow() {
    this.setData({
      apiModeText: getApp().getRuntimeStatus().apiModeText
    });
  },

  onInput(event) {
    const field = event.currentTarget.dataset.field;
    this.setData({
      [field]: event.detail.value
    });
  },

  onSwitchChange(event) {
    const field = event.currentTarget.dataset.field;
    this.setData({
      [field]: event.detail.value
    });
  },

  onGiftChange(event) {
    this.setData({
      giftChoice: event.detail.value
    });
  },

  validatePhone() {
    if (!/^1\d{10}$/.test(this.data.phone)) {
      wx.showToast({
        title: "请输入 11 位手机号",
        icon: "none"
      });
      return false;
    }

    return true;
  },

  async sendCode() {
    if (!this.validatePhone()) {
      return;
    }

    try {
      const app = getApp();
      await app.ensureLogin();
      await app.globalData.api.sendCode(this.data.phone);
      this.setData({
        sentCode: true,
        code: config.demoCode
      });
      wx.showToast({
        title: "验证码 666666",
        icon: "none"
      });
    } catch (error) {
      wx.showToast({
        title: error.message || "验证码发送失败",
        icon: "none"
      });
    }
  },

  async submitJoin() {
    if (!this.validatePhone()) {
      return;
    }

    if (this.data.code !== config.demoCode) {
      wx.showToast({
        title: "验证码固定 666666",
        icon: "none"
      });
      return;
    }

    if (!this.data.acceptedPrivacy || !this.data.acceptedMemberRule) {
      wx.showToast({
        title: "请确认隐私和会员规则",
        icon: "none"
      });
      return;
    }

    try {
      const app = getApp();
      await app.ensureLogin();
      const bindResult = await app.globalData.api.bindPhone(this.data.phone, this.data.code);
      session.setAuthMember(bindResult.member);
      const joinResult = await app.globalData.api.joinMember({
        nickname: this.data.nickname,
        phone: this.data.phone,
        source: this.data.source,
        scene: this.data.scene,
        referrerMemberId: this.data.referrerMemberId,
        accepted_privacy: true,
        accepted_member_rule: true,
        gift_choice: this.data.giftChoice
      });
      const checkedResult = validateJoinResult({
        bindMember: bindResult.member,
        joinResult,
        expectedPhone: this.data.phone
      });

      this.setData({
        joined: true,
        issuedCoupons: checkedResult.issuedCoupons
      });
      wx.showToast({
        title: "入会成功",
        icon: "success"
      });
    } catch (error) {
      wx.showToast({
        title: error.message || "入会失败",
        icon: "none"
      });
    }
  },

  goHome() {
    wx.reLaunch({
      url: "/pages/home/index"
    });
  }
});
