import { existsSync, readFileSync } from "node:fs";
import { join } from "node:path";

const root = process.cwd();

const requiredPaths = [
  "package.json",
  "pnpm-workspace.yaml",
  ".env.example",
  "apps/api",
  "apps/admin-web",
  "apps/miniapp",
  "docs/contracts/openapi.yaml"
];

const requiredEnvKeys = [
  "APP_ENV",
  "HTTP_PORT",
  "DATA_DIR",
  "JWT_SECRET",
  "MOCK_SMS_CODE",
  "MOCK_WECHAT",
  "MOCK_PAYMENT",
  "MOCK_NOTIFICATION",
  "ADMIN_API_ORIGIN",
  "ADMIN_API_BASE_URL",
  "MINIAPP_API_BASE_URL"
];

function fail(message) {
  console.error(`check-workspace failed: ${message}`);
  process.exit(1);
}

for (const relativePath of requiredPaths) {
  if (!existsSync(join(root, relativePath))) {
    fail(`missing ${relativePath}`);
  }
}

const pkg = JSON.parse(readFileSync(join(root, "package.json"), "utf8"));
if (pkg.private !== true) {
  fail("package.json must be private");
}

if (!pkg.packageManager?.startsWith("pnpm@")) {
  fail("package.json packageManager must use pnpm");
}

for (const scriptName of ["check", "dev:api", "dev:admin", "dev:miniapp"]) {
  if (!pkg.scripts?.[scriptName]) {
    fail(`package.json missing script ${scriptName}`);
  }
}

const workspace = readFileSync(join(root, "pnpm-workspace.yaml"), "utf8");
if (!workspace.includes("apps/*")) {
  fail("pnpm-workspace.yaml must include apps/*");
}

const envExample = readFileSync(join(root, ".env.example"), "utf8");
for (const key of requiredEnvKeys) {
  if (!new RegExp(`^${key}=`, "m").test(envExample)) {
    fail(`.env.example missing ${key}`);
  }
}

console.log("workspace foundation check passed");
