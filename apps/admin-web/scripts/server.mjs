import { createServer } from "node:http";
import { createReadStream } from "node:fs";
import { readFile, stat } from "node:fs/promises";
import { extname, join, normalize, sep } from "node:path";
import { fileURLToPath } from "node:url";
import { adminConfig, configScript, proxyTarget } from "./config.mjs";

const rootDir = join(fileURLToPath(new URL(".", import.meta.url)), "..");
const publicDir = join(rootDir, "src");
const initialPort = Number(process.env.ADMIN_WEB_PORT || process.env.PORT || 5173);
const config = adminConfig(process.env, { dev: true });
const apiOrigin = new URL(process.env.ADMIN_API_ORIGIN || "http://localhost:8080").origin;

const contentTypes = {
  ".html": "text/html; charset=utf-8",
  ".js": "text/javascript; charset=utf-8",
  ".css": "text/css; charset=utf-8",
  ".json": "application/json; charset=utf-8",
  ".svg": "image/svg+xml",
  ".ico": "image/x-icon"
};

function send(res, statusCode, body, headers = {}) {
  res.writeHead(statusCode, {
    "Cache-Control": "no-store",
    ...headers
  });
  res.end(body);
}

function staticPath(urlPath) {
  const decoded = decodeURIComponent(urlPath.split("?")[0]);
  const normalized = normalize(decoded === "/" ? "/index.html" : decoded);
  const relative = normalized.replace(/^([/\\])+/, "");
  const fullPath = join(publicDir, relative);
  if (!fullPath.startsWith(publicDir + sep) && fullPath !== publicDir) {
    return join(publicDir, "index.html");
  }
  return fullPath;
}

async function readRequestBody(req) {
  const chunks = [];
  for await (const chunk of req) {
    chunks.push(chunk);
  }
  return Buffer.concat(chunks);
}

async function proxyRequest(req, res, target) {

  const body = req.method === "GET" || req.method === "HEAD" ? undefined : await readRequestBody(req);
  const headers = {};
  for (const name of ["authorization", "content-type", "idempotency-key"]) {
    if (req.headers[name]) {
      headers[name] = req.headers[name];
    }
  }

  try {
    const upstream = await fetch(target, {
      method: req.method,
      headers,
      body
    });
    const buffer = Buffer.from(await upstream.arrayBuffer());
    res.writeHead(upstream.status, {
      "Content-Type": upstream.headers.get("content-type") || "application/json; charset=utf-8",
      "Cache-Control": "no-store"
    });
    res.end(buffer);
  } catch (error) {
    send(res, 502, JSON.stringify({ code: 50000, message: `API 连接失败：${error.message}`, data: null }), {
      "Content-Type": "application/json; charset=utf-8"
    });
  }
}

async function serveStatic(req, res, requestUrl) {
  let fullPath = staticPath(requestUrl.pathname);
  let fileStat;

  try {
    fileStat = await stat(fullPath);
    if (fileStat.isDirectory()) {
      fullPath = join(fullPath, "index.html");
      fileStat = await stat(fullPath);
    }
  } catch {
    fullPath = join(publicDir, "index.html");
    fileStat = await stat(fullPath);
  }

  if (!fileStat.isFile()) {
    send(res, 404, "Not found");
    return;
  }

  const ext = extname(fullPath);
  res.writeHead(200, {
    "Content-Type": contentTypes[ext] || "application/octet-stream",
    "Content-Length": fileStat.size,
    "Cache-Control": "no-store"
  });
  createReadStream(fullPath).pipe(res);
}

export function createAdminServer() {
  return createServer(async (req, res) => {
    const requestUrl = new URL(req.url || "/", "http://127.0.0.1");
    if (requestUrl.pathname === "/config.js") {
      send(res, 200, configScript(config), { "Content-Type": "text/javascript; charset=utf-8" });
      return;
    }
    if (requestUrl.pathname.startsWith("/__admin_proxy")) {
      const target = proxyTarget(requestUrl.pathname, requestUrl.search, apiOrigin, config.appEnv);
      if (!target) {
        send(res, 404, "Not found");
        return;
      }
      await proxyRequest(req, res, target);
      return;
    }

    await serveStatic(req, res, requestUrl);
  });
}

export async function readIndexHtml() {
  return readFile(join(publicDir, "index.html"), "utf8");
}

async function listen(server, port, attemptsLeft = 10) {
  return new Promise((resolve, reject) => {
    const onError = (error) => {
      server.off("listening", onListening);
      if (error.code === "EADDRINUSE" && attemptsLeft > 1) {
        resolve(listen(createAdminServer(), port + 1, attemptsLeft - 1));
        return;
      }
      reject(error);
    };
    const onListening = () => {
      server.off("error", onError);
      resolve({ server, port });
    };

    server.once("error", onError);
    server.once("listening", onListening);
    server.listen(port, "127.0.0.1");
  });
}

if (process.argv[1] === fileURLToPath(import.meta.url)) {
  const { port } = await listen(createAdminServer(), initialPort);
  console.log(`admin-web dev server: http://localhost:${port}`);
  console.log(config.proxyEnabled ? `API proxy target: ${apiOrigin}/api/v1/` : "API proxy disabled.");
}
