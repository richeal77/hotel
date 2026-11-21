const { createApiClient } = require("./utils/api");
const { createDemoResponder } = require("./utils/demo-api");
const { parseLaunchContext } = require("./utils/scene");
const session = require("./utils/session");
const { LOCAL_DEMO_MODE, LIVE_MODE, apiModeText } = require("./utils/settings");

App({
  globalData: {
    api: null,
    apiBaseUrl: session.getApiBaseUrl(),
    apiMode: session.getApiMode(),
    apiModeText: apiModeText(session.getApiMode()),
    launchContext: session.getLaunchContext()
  },

  onLaunch(options) {
    this.setupApi();
    this.captureLaunch(options || {});
  },

  onShow(options) {
    if (options && options.query && Object.keys(options.query).length) {
      this.captureLaunch(options);
    }
  },

  setupApi() {
    const apiMode = session.getApiMode();
    const apiBaseUrl = session.getApiBaseUrl();
    const demoResponder = apiMode === LOCAL_DEMO_MODE
      ? createDemoResponder({ storage: session.createWxStorageAdapter() })
      : null;

    this.globalData.apiBaseUrl = apiBaseUrl;
    this.globalData.apiMode = apiMode;
    this.globalData.apiModeText = apiModeText(apiMode);
    this.globalData.api = apiMode === LIVE_MODE && !apiBaseUrl ? null : createApiClient({
      baseUrl: apiBaseUrl,
      getToken: session.getToken,
      demoOnly: apiMode === LOCAL_DEMO_MODE,
      fallback: demoResponder,
      onFallback: () => {}
    });
  },

  applyRuntimeSettings() {
    this.setupApi();
    this.loginReady = null;
    return this.getRuntimeStatus();
  },

  getRuntimeStatus() {
    return {
      apiBaseUrl: this.globalData.apiBaseUrl,
      apiMode: this.globalData.apiMode,
      apiModeText: this.globalData.apiModeText
    };
  },

  captureLaunch(options) {
    const context = parseLaunchContext(options || {});
    this.globalData.launchContext = context;
    session.setLaunchContext(context);
    return context;
  },

  capturePageQuery(query) {
    if (!query || !Object.keys(query).length) {
      return this.globalData.launchContext;
    }

    return this.captureLaunch({
      scene: this.globalData.launchContext.launchScene,
      query
    });
  },

  login() {
    if (!this.globalData.api) return Promise.reject(new Error("请先在我的页面配置真实 HTTPS API 域名"));
    const context = this.globalData.launchContext || session.getLaunchContext();
    const payload = {
        scene: context.scene,
        referrerMemberId: context.referrerMemberId
    };
    const loginRequest = this.globalData.apiMode === LIVE_MODE
      ? new Promise((resolve, reject) => wx.login({ success: resolve, fail: reject }))
        .then((result) => {
          if (!result.code) throw new Error("微信登录失败");
          return this.globalData.api.wxLogin(result.code, payload);
        })
      : this.globalData.api.mockLogin(payload);
    return loginRequest
      .then((data) => {
        session.setToken(data.member_token);
        session.setAuthMember(data.member);
        return data;
      });
  },

  ensureLogin() {
    if (!this.loginReady) {
      this.setupApi();
      this.loginReady = this.login().catch((error) => {
        this.loginReady = null;
        throw error;
      });
    }

    return this.loginReady;
  },

  refreshLogin() {
    if (this.globalData.apiMode === LIVE_MODE && session.getToken()) {
      return this.globalData.api.logout().catch(() => {}).then(() => {
        session.setToken("");
        session.setAuthMember(null);
        this.loginReady = null;
        return this.ensureLogin();
      });
    }
    this.loginReady = null;
    return this.ensureLogin();
  }
});
