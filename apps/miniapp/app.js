const { createApiClient } = require("./utils/api");
const { createDemoResponder } = require("./utils/demo-api");
const { parseLaunchContext } = require("./utils/scene");
const config = require("./utils/config");
const session = require("./utils/session");
const { LOCAL_DEMO_MODE, apiModeText } = require("./utils/settings");

App({
  globalData: {
    api: null,
    apiBaseUrl: session.getApiBaseUrl(),
    apiMode: session.getApiMode(),
    apiModeText: apiModeText(session.getApiMode()),
    demoCode: config.demoCode,
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
    const demoResponder = createDemoResponder({
      storage: session.createWxStorageAdapter()
    });

    this.globalData.apiBaseUrl = apiBaseUrl;
    this.globalData.apiMode = apiMode;
    this.globalData.apiModeText = apiModeText(apiMode);
    this.globalData.api = createApiClient({
      baseUrl: apiBaseUrl,
      getToken: session.getToken,
      demoOnly: apiMode === LOCAL_DEMO_MODE,
      enableFallback: apiMode === LOCAL_DEMO_MODE,
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
    const context = this.globalData.launchContext || session.getLaunchContext();
    return this.globalData.api
      .mockLogin({
        scene: context.scene,
        referrerMemberId: context.referrerMemberId
      })
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
    this.loginReady = null;
    return this.ensureLogin();
  }
});
