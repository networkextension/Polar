import { cp, mkdir, readFile, rm, writeFile } from "node:fs/promises";
import path from "node:path";
import { spawn } from "node:child_process";
import { fileURLToPath } from "node:url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const rootDir = path.resolve(__dirname, "..");
const publicDir = path.join(rootDir, "public");
const distDir = path.join(rootDir, "dist");
const pkgPath = path.join(rootDir, "package.json");
const tsconfigPath = path.join(rootDir, "tsconfig.json");
const publicScriptsDir = path.join(publicDir, "scripts");

async function runTypeScriptBuild() {
  await rm(publicScriptsDir, { recursive: true, force: true });

  await new Promise((resolve, reject) => {
    const child = spawn(
      process.execPath,
      [path.join(rootDir, "node_modules", "typescript", "bin", "tsc"), "-p", tsconfigPath],
      {
        cwd: rootDir,
        stdio: "inherit",
      }
    );

    child.on("exit", (code) => {
      if (code === 0) {
        resolve();
        return;
      }
      reject(new Error(`TypeScript build failed with code ${code}`));
    });
    child.on("error", reject);
  });
}

async function clean() {
  await rm(distDir, { recursive: true, force: true });
  await rm(publicScriptsDir, { recursive: true, force: true });
  console.log("Cleaned ui/dist");
}

async function build() {
  await runTypeScriptBuild();
  await rm(distDir, { recursive: true, force: true });
  await mkdir(distDir, { recursive: true });
  await cp(publicDir, distDir, { recursive: true });

  const pkg = JSON.parse(await readFile(pkgPath, "utf8"));
  const meta = {
    name: pkg.name,
    version: pkg.version,
    builtAt: new Date().toISOString(),
    output: "dist",
    deployHint: "Copy the contents of ui/dist into your web server ui directory.",
  };

  await writeFile(
    path.join(distDir, "build-meta.json"),
    `${JSON.stringify(meta, null, 2)}\n`,
    "utf8"
  );

  console.log("Built UI into ui/dist");
}

if (process.argv.includes("--clean")) {
  await clean();
} else {
  await build();
}
