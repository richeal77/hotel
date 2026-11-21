import { cp, mkdir, rm, writeFile } from "node:fs/promises";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { adminConfig, configScript } from "./config.mjs";

const currentFile = fileURLToPath(import.meta.url);
const defaultRootDir = join(dirname(currentFile), "..");

export async function buildAdmin({ rootDir = defaultRootDir } = {}) {
  const sourceDir = join(rootDir, "src");
  const distDir = join(rootDir, "dist");

  await rm(distDir, { recursive: true, force: true });
  await mkdir(distDir, { recursive: true });
  await cp(sourceDir, distDir, { recursive: true });
  await writeFile(join(distDir, "config.js"), configScript(adminConfig()));

  return { sourceDir, distDir };
}

if (process.argv[1] === currentFile) {
  const { distDir } = await buildAdmin();
  console.log(`admin-web built: ${distDir}`);
}
