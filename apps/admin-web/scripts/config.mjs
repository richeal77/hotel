export function adminConfig(env = process.env, { dev = false } = {}) {
  const appEnv = env.APP_ENV || "local-demo";
  if (!["local-demo", "integration", "production"].includes(appEnv)) {
    throw new Error(`Unsupported APP_ENV: ${appEnv}`);
  }

  const origin = new URL(env.ADMIN_API_ORIGIN || "http://localhost:8080");
  if (!["http:", "https:"].includes(origin.protocol) || origin.pathname !== "/" || origin.search || origin.hash || origin.username || origin.password) {
    throw new Error("ADMIN_API_ORIGIN must be an HTTP(S) origin");
  }
  const apiBaseUrl = dev && appEnv !== "production"
    ? `${origin.origin}/api/v1`
    : (env.ADMIN_API_BASE_URL || (appEnv === "local-demo" ? `${origin.origin}/api/v1` : "/api/v1"));
  if (appEnv === "production" && !/^\/api\/v1\/?$/.test(apiBaseUrl) && !apiBaseUrl.startsWith("https://")) {
    throw new Error("Production ADMIN_API_BASE_URL must be same-origin /api/v1 or HTTPS");
  }
  if (apiBaseUrl.startsWith("https://")) {
    const url = new URL(apiBaseUrl);
    if ((url.pathname !== "/api/v1" && url.pathname !== "/api/v1/") || url.username || url.password || url.search || url.hash) {
      throw new Error("ADMIN_API_BASE_URL must end at /api/v1");
    }
  }

  return { appEnv, apiBaseUrl: apiBaseUrl.replace(/\/+$/, ""), proxyEnabled: dev && appEnv !== "production" };
}

export function configScript(config) {
  return `window.ADMIN_CONFIG = Object.freeze(${JSON.stringify(config).replaceAll("<", "\\u003c")});\n`;
}

export function proxyTarget(pathname, search, origin, appEnv) {
  if (appEnv === "production" || !pathname.startsWith("/__admin_proxy/api/v1/")) {
    return null;
  }
  const target = new URL(pathname.slice("/__admin_proxy".length) + search, origin);
  return target.pathname.startsWith("/api/v1/") ? target.href : null;
}
