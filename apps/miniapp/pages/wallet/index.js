const { money, shortDate } = require("../../utils/format");

const LEDGER_TYPE_TEXT = {
  stay_cashback: "入住返住金",
  referral_reward: "推荐奖励",
  manual_grant: "手工发放",
  redeem: "核销使用",
  expire: "过期",
  reverse: "冲正"
};

const STATUS_TEXT = {
  frozen: "冻结中",
  available: "可用",
  used: "已使用",
  expired: "已过期",
  reversed: "已冲正"
};

Page({
  data: {
    loading: true,
    apiModeText: "HTTP 接口（验收）",
    balanceText: "¥0.00",
    grantedText: "¥0.00",
    usedText: "¥0.00",
    ledgers: []
  },

  onLoad() {
    this.loadWallet();
  },

  onShow() {
    this.setData({
      apiModeText: getApp().getRuntimeStatus().apiModeText
    });
  },

  onPullDownRefresh() {
    this.loadWallet().finally(() => wx.stopPullDownRefresh());
  },

  async loadWallet() {
    this.setData({ loading: true });

    try {
      const app = getApp();
      await app.ensureLogin();
      const wallet = await app.globalData.api.getWallet();
      const ledgers = (wallet.ledgers || []).map((ledger) => ({
        ...ledger,
        typeText: LEDGER_TYPE_TEXT[ledger.type] || ledger.type,
        statusText: STATUS_TEXT[ledger.status] || ledger.status,
        amountText: money(ledger.amount_cent),
        expireText: shortDate(ledger.expire_at),
        createdText: shortDate(ledger.created_at)
      }));

      this.setData({
        apiModeText: app.getRuntimeStatus().apiModeText,
        balanceText: money(wallet.balance_cent),
        grantedText: money(wallet.total_granted_cent),
        usedText: money(wallet.total_used_cent),
        ledgers
      });
    } catch (error) {
      wx.showToast({
        title: error.message || "返住金加载失败",
        icon: "none"
      });
    } finally {
      this.setData({ loading: false });
    }
  },

  goOrders() {
    wx.navigateTo({
      url: "/pages/orders/index"
    });
  }
});
