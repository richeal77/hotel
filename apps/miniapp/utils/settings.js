const config = require("./config");

const API_BASE_URL_KEY = "hotel_api_base_url";
const API_MODE_KEY = "hotel_api_mode";
const HTTP_MODE = "http";
const LOCAL_DEMO_MODE = "local-demo";

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

function normalizeApiMode(value) {
  return value === LOCAL_DEMO_MODE ? LOCAL_DEMO_MODE : HTTP_MODE;
}

function apiModeText(mode) {
  return normalizeApiMode(mode) === LOCAL_DEMO_MODE ? "本地演示数据" : "HTTP 接口（验收）";
}

function createRuntimeSettings(options) {
  const settings = options || {};
  const storage = settings.storage;

  if (!storage || typeof storage.get !== "function" || typeof storage.set !== "function") {
    throw new Error("runtime settings requires storage get/set");
  }

  return {
    getApiBaseUrl() {
      return normalizeApiBaseUrl(storage.get(API_BASE_URL_KEY) || config.apiBaseUrl);
    },
    setApiBaseUrl(value) {
      const normalized = normalizeApiBaseUrl(value);
      storage.set(API_BASE_URL_KEY, normalized);
      return normalized;
    },
    getApiMode() {
      return normalizeApiMode(storage.get(API_MODE_KEY));
    },
    setApiMode(value) {
      const normalized = normalizeApiMode(value);
      storage.set(API_MODE_KEY, normalized);
      return normalized;
    }
  };
}

module.exports = {
  API_BASE_URL_KEY,
  API_MODE_KEY,
  HTTP_MODE,
  LOCAL_DEMO_MODE,
  apiModeText,
  createRuntimeSettings,
  normalizeApiBaseUrl,
  normalizeApiMode
};
