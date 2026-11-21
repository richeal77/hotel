const config = require("./config");

const API_BASE_URL_KEY = "hotel_api_base_url";
const LIVE_API_BASE_URL_KEY = "hotel_live_api_base_url";
const API_MODE_KEY = "hotel_api_mode";
const HTTP_MODE = "http";
const LOCAL_DEMO_MODE = "local-demo";
const LIVE_MODE = "live";

function normalizeApiBaseUrl(value) {
  const url = String(value || config.apiBaseUrl).trim().replace(/\/+$/, "");

  if (!/^https?:\/\//.test(url)) {
    throw new Error("API 地址必须以 http:// 或 https:// 开头");
  }

  if (!/\/api\/v1$/.test(url)) {
    throw new Error("API 地址必须以 /api/v1 结尾");
  }

  return url;
}

function normalizeLiveApiBaseUrl(value) {
  const url = String(value || "").trim().replace(/\/+$/, "");
  if (!/^https:\/\//.test(url)) throw new Error("真实接口必须使用 HTTPS API 域名");
  const match = /^https:\/\/([^/?#]+)(\/[^?#]*)$/.exec(url);
  const host = match && match[1];
  if (!host || host.includes("@") || !/^[a-z0-9-]+(?:\.[a-z0-9-]+)+$/i.test(host) ||
      host === "localhost" || /^(?:\d+\.){3}\d+$/.test(host)) {
    throw new Error("真实接口必须填写已配置的 HTTPS API 域名");
  }
  if (!/\/api\/v1$/.test(url)) throw new Error("API 地址必须以 /api/v1 结尾");
  return url;
}

function normalizeApiMode(value) {
  if (value === LOCAL_DEMO_MODE) return LOCAL_DEMO_MODE;
  if (value === LIVE_MODE) return LIVE_MODE;
  return HTTP_MODE;
}

function apiModeText(mode) {
  if (normalizeApiMode(mode) === LOCAL_DEMO_MODE) return "本地演示数据";
  if (normalizeApiMode(mode) === LIVE_MODE) return "真实微信接口";
  return "HTTP 接口（验收）";
}

function createRuntimeSettings(options) {
  const settings = options || {};
  const storage = settings.storage;

  if (!storage || typeof storage.get !== "function" || typeof storage.set !== "function") {
    throw new Error("runtime settings requires storage get/set");
  }

  return {
    getHttpApiBaseUrl() {
      return normalizeApiBaseUrl(storage.get(API_BASE_URL_KEY) || config.apiBaseUrl);
    },
    getApiBaseUrl() {
      if (this.getApiMode() === LIVE_MODE) return this.getLiveApiBaseUrl();
      return this.getHttpApiBaseUrl();
    },
    setApiBaseUrl(value) {
      const normalized = normalizeApiBaseUrl(value);
      storage.set(API_BASE_URL_KEY, normalized);
      return normalized;
    },
    getLiveApiBaseUrl() {
      const value = storage.get(LIVE_API_BASE_URL_KEY);
      return value ? normalizeLiveApiBaseUrl(value) : "";
    },
    setLiveApiBaseUrl(value) {
      const normalized = normalizeLiveApiBaseUrl(value);
      storage.set(LIVE_API_BASE_URL_KEY, normalized);
      return normalized;
    },
    getApiMode() {
      return normalizeApiMode(storage.get(API_MODE_KEY));
    },
    setApiMode(value) {
      const normalized = normalizeApiMode(value);
      if (normalized === LIVE_MODE && !this.getLiveApiBaseUrl()) {
        throw new Error("请先配置真实 HTTPS API 域名");
      }
      storage.set(API_MODE_KEY, normalized);
      return normalized;
    }
  };
}

module.exports = {
  API_BASE_URL_KEY,
  LIVE_API_BASE_URL_KEY,
  API_MODE_KEY,
  HTTP_MODE,
  LOCAL_DEMO_MODE,
  LIVE_MODE,
  apiModeText,
  createRuntimeSettings,
  normalizeApiBaseUrl,
  normalizeLiveApiBaseUrl,
  normalizeApiMode
};
