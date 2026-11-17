const { money, shortDate } = require("../../utils/format");
const session = require("../../utils/session");

Page({
  data: {
    loading: true,
    apiModeText: "HTTP 接口（验收）",
    context: {},
    pageData: {
      share_qr_text: "",
      summary: {
        referred_count: 0,
        referred_orders: 0,
        pending_reward_cent: 0,
        available_reward_cent: 0
      },
      rewards: []
    },
    sharePath: "/pages/home/index",
    availableRewardText: "¥0.00",
    pendingRewardText: "¥0.00",
    rewards: [],
    relationText: ""
  },

  onLoad(options) {
    getApp().capturePageQuery(options);
    this.loadReferral();
  },

  onShow() {
    this.setData({
      apiModeText: getApp().getRuntimeStatus().apiModeText
    });
  },

  onPullDownRefresh() {
    this.loadReferral().finally(() => wx.stopPullDownRefresh());
  },

  async loadReferral() {
    this.setData({ loading: true });

    try {
      const app = getApp();
      await app.ensureLogin();
      const data = await app.globalData.api.getReferral();
      const me = await app.globalData.api.getMe();
      const memberId = me.member_code || (me.member && me.member.id) || "";
      const rewards = (data.rewards || []).map((reward) => ({
        ...reward,
        amountText: money(reward.reward_amount_cent),
        orderAmountText: money(reward.actual_amount_cent),
        createdText: shortDate(reward.created_at)
      }));

      this.setData({
        apiModeText: app.getRuntimeStatus().apiModeText,
        context: session.getLaunchContext(),
        pageData: data,
        sharePath: `/pages/home/index?scene=referral_qr&referrer=${memberId}`,
        availableRewardText: money(data.summary.available_reward_cent),
        pendingRewardText: money(data.summary.pending_reward_cent),
        rewards
      });
    } catch (error) {
      wx.showToast({
        title: error.message || "推荐数据加载失败",
        icon: "none"
      });
    } finally {
      this.setData({ loading: false });
    }
  },

  async bindReferrer() {
    if (!this.data.context.referrerMemberId) {
      wx.showToast({
        title: "未识别推荐人",
        icon: "none"
      });
      return;
    }

    try {
      const app = getApp();
      await app.ensureLogin();
      const data = await app.globalData.api.bindReferral(this.data.context.referrerMemberId);
      this.setData({
        relationText: `${data.relation.referrer_member_id} -> ${data.relation.referred_member_id}`
      });
      wx.showToast({
        title: "推荐关系已绑定",
        icon: "success"
      });
    } catch (error) {
      wx.showToast({
        title: error.message || "绑定失败",
        icon: "none"
      });
    }
  },

  copyShareText() {
    wx.setClipboardData({
      data: this.data.pageData ? this.data.pageData.share_qr_text : this.data.sharePath
    });
  },

  onShareAppMessage() {
    return {
      title: "高碑店尚客优会员推荐",
      path: this.data.sharePath
    };
  }
});
