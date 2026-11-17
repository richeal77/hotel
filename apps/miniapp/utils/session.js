const TOKEN_KEY = "hotel_member_token";
const AUTH_MEMBER_KEY = "hotel_auth_member";
const LAUNCH_CONTEXT_KEY = "hotel_launch_context";
const { createRuntimeSettings } = require("./settings");

function read(key, fallback) {
  try {
    const value = wx.getStorageSync(key);
    return value === "" || value === undefined || value === null ? fallback : value;
  } catch (error) {
    return fallback;
  }
}

function write(key, value) {
  try {
    wx.setStorageSync(key, value);
  } catch (error) {
    // Miniapp storage can fail in preview edge cases; page state still keeps running.
  }
}

function getToken() {
  return read(TOKEN_KEY, "");
}

function setToken(token) {
  write(TOKEN_KEY, token || "");
}

function getAuthMember() {
  return read(AUTH_MEMBER_KEY, null);
}

function setAuthMember(member) {
  write(AUTH_MEMBER_KEY, member || null);
}

function getLaunchContext() {
  return read(LAUNCH_CONTEXT_KEY, {
    source: "frontdesk",
    scene: "frontdesk_qr",
    referrerMemberId: ""
  });
}

function setLaunchContext(context) {
  write(LAUNCH_CONTEXT_KEY, context);
}

function getRuntimeSettings() {
  return createRuntimeSettings({
    storage: createWxStorageAdapter()
  });
}

function getApiMode() {
  return getRuntimeSettings().getApiMode();
}

function setApiMode(mode) {
  return getRuntimeSettings().setApiMode(mode);
}

function getApiBaseUrl() {
  return getRuntimeSettings().getApiBaseUrl();
}

function setApiBaseUrl(url) {
  return getRuntimeSettings().setApiBaseUrl(url);
}

function createWxStorageAdapter() {
  return {
    get(key) {
      return read(key, null);
    },
    set(key, value) {
      write(key, value);
    }
  };
}

module.exports = {
  createWxStorageAdapter,
  getApiBaseUrl,
  getApiMode,
  getAuthMember,
  getLaunchContext,
  getRuntimeSettings,
  getToken,
  setApiBaseUrl,
  setApiMode,
  setAuthMember,
  setLaunchContext,
  setToken
};
