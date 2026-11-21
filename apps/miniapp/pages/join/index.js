const config = require("../../utils/config");
const { guestPhoneForContext } = require("../../utils/defaults");
const { validateJoinResult } = require("../../utils/join");
const session = require("../../utils/session");
const { LIVE_MODE } = require("../../utils/settings");

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
    issuedCoupons: [],
    liveEnabled: false
  },

  onLoad(options) {
    const app = getApp();
    app.capturePageQuery(options);
    const context = app.globalData.launchContext;
    const phone = guestPhoneForContext(context);
    const liveEnabled = app.getRuntimeStatus().apiMode === LIVE_MODE;
    this.setData({
      apiModeText: app.getRuntimeStatus().apiModeText,
      phone: liveEnabled ? "" : phone,
      code: liveEnabled ? "" : config.demoCode,
      nickname: liveEnabled ? "" : "张三",
      acceptedPrivacy: !liveEnabled,
      acceptedMemberRule: !liveEnabled,
      phoneHint: phone === "13800138902" ? "朋友推荐示例手机号：13800138902" : "前台验收手机号：13800138901",
      source: context.source,
      scene: context.scene,
      referrerMemberId: context.referrerMemberId,
      liveEnabled
    });
  },

  onShow() {
    const liveEnabled = getApp().getRuntimeStatus().apiMode === LIVE_MODE;
    const modeChanged = liveEnabled !== this.data.liveEnabled;
    this.setData({
      apiModeText: getApp().getRuntimeStatus().apiModeText,
      liveEnabled,
      ...(modeChanged && liveEnabled ? {
        phone: "", code: "", sentCode: false, acceptedPrivacy: false, acceptedMemberRule: false
      } : {})
    });
  },

  onInput(event) {
    const field = event.currentTarget.dataset.field;
    this.setData({
      [field]: event.detail.value
    });
  },

  async onSwitchChange(event) {
    const field = event.currentTarget.dataset.field;
    if (field === "acceptedPrivacy" && this.data.liveEnabled && event.detail.value) {
      try {
        await new Promise((resolve, reject) => {
          if (typeof wx.requirePrivacyAuthorize !== "function") {
            reject(new Error("当前微信版本不支持隐私授权，请升级后重试"));
            return;
          }
          wx.requirePrivacyAuthorize({ success: resolve, fail: reject });
        });
      } catch (error) {
        this.setData({ acceptedPrivacy: false });
        wx.showToast({ title: error.errMsg || error.message || "请先同意隐私指引", icon: "none" });
        return;
      }
    }
    this.setData({
      [field]: event.detail.value
    });
  },

  openPrivacyContract() {
    if (!this.data.liveEnabled) return;
    if (typeof wx.openPrivacyContract !== "function") {
      wx.showToast({ title: "当前微信版本不支持查看隐私指引", icon: "none" });
      return;
    }
    wx.openPrivacyContract({ fail: (error) => wx.showToast({ title: error.errMsg || "打开隐私指引失败", icon: "none" }) });
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
    if (this.data.liveEnabled) return;
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
    if (this.data.liveEnabled) {
      wx.showToast({ title: "请点击微信手机号授权", icon: "none" });
      return;
    }
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
      await this.finishJoin(bindResult);
    } catch (error) {
      wx.showToast({ title: error.message || "入会失败", icon: "none" });
    }
  },

  async bindWeChatPhone(event) {
    if (!this.data.liveEnabled) return;
    if (!event.detail || !event.detail.code) {
      wx.showToast({ title: "未获得手机号授权", icon: "none" });
      return;
    }
    if (!this.data.acceptedPrivacy || !this.data.acceptedMemberRule) {
      wx.showToast({ title: "请确认隐私和会员规则", icon: "none" });
      return;
    }
    try {
      const app = getApp();
      await app.ensureLogin();
      const bindResult = await app.globalData.api.wxBindPhone(event.detail.code);
      session.setAuthMember(bindResult.member);
      this.setData({ phone: bindResult.member.phone });
      await this.finishJoin(bindResult);
    } catch (error) {
      wx.showToast({ title: error.message || "手机号绑定失败", icon: "none" });
    }
  },

  async finishJoin(bindResult) {
    const joinResult = await getApp().globalData.api.joinMember({
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

    this.setData({ joined: true, issuedCoupons: checkedResult.issuedCoupons });
    wx.showToast({ title: "入会成功", icon: "success" });
  },

  goHome() {
    wx.reLaunch({
      url: "/pages/home/index"
    });
  }
});
