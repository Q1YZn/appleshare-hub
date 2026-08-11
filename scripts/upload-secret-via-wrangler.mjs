import { spawn } from "node:child_process";
import fs from "node:fs";

// Windows wrangler reads secret values from stdin; Node pipes UTF-8 bytes so
// Chinese text and tokens are not corrupted by PowerShell's default encoding.
const target = process.argv[2];
const name = process.argv[3];
const value = process.argv[4];

if (!["pages", "worker"].includes(target) || !name || value === undefined) {
  console.error("usage: node scripts/upload-secret-via-wrangler.mjs <pages|worker> <NAME> <VALUE>");
  process.exit(1);
}

const args =
  target === "pages"
    ? ["wrangler", "pages", "secret", "put", name, "--project-name", "appleshare-hub"]
    : ["wrangler", "secret", "put", name, "--config", "wrangler.worker.toml"];

const child = spawn("npx", args, {
  stdio: ["pipe", "inherit", "inherit"],
  shell: true
});
child.stdin.end(value);

await new Promise((resolve) => child.on("exit", resolve));
