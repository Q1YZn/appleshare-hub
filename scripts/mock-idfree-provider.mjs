import http from "node:http";
import { createProvider } from "../functions/_lib/providers.js";

function listen(handler) {
  return new Promise((resolve) => {
    const server = http.createServer(handler);
    server.listen(0, "127.0.0.1", () => resolve(server));
  });
}

function readBody(request) {
  return new Promise((resolve, reject) => {
    let data = "";
    request.on("data", (chunk) => {
      data += chunk;
    });
    request.on("end", () => resolve(data));
    request.on("error", reject);
  });
}

function json(response, value, status = 200) {
  const body = JSON.stringify(value);
  response.writeHead(status, {
    "Content-Type": "application/json",
    "Content-Length": Buffer.byteLength(body)
  });
  response.end(body);
}

const solver = await listen(async (request, response) => {
  const body = await readBody(request);
  const payload = JSON.parse(body || "{}");
  if (request.url === "/createTask") {
    if (payload.task?.type !== "AntiTurnstileTaskProxyLess" || payload.task?.websiteKey !== "test-site-key") {
      json(response, { errorDescription: "bad task payload" }, 400);
      return;
    }
    json(response, { taskId: "task-123" });
    return;
  }
  if (request.url === "/getTaskResult") {
    json(response, { status: "ready", solution: { token: "turnstile-solved" } });
    return;
  }
  json(response, { errorDescription: "unknown solver endpoint" }, 404);
});

const proxy = await listen(async (request, response) => {
  const url = new URL(request.url, "http://localhost");
  if (url.pathname !== "/fetch") {
    json(response, { error: "not found" }, 404);
    return;
  }
  const target = url.searchParams.get("url");
  if (!target) {
    json(response, { error: "missing target url" }, 400);
    return;
  }
  const upstream = await fetch(target, {
    method: request.method,
    headers: request.headers,
    body: ["GET", "HEAD"].includes(request.method) ? undefined : await readBody(request),
    redirect: "follow"
  });
  const body = await upstream.arrayBuffer();
  response.writeHead(upstream.status, {
    "Content-Type": upstream.headers.get("content-type") || "application/octet-stream"
  });
  response.end(Buffer.from(body));
});

const site = await listen(async (request, response) => {
  const url = new URL(request.url, "http://localhost");
  if (url.pathname === "/") {
    const body = `<html>
      <meta name="x-token" content="page-token-123">
      <div class="cf-turnstile" data-sitekey="test-site-key"></div>
    </html>`;
    response.writeHead(200, {
      "Content-Type": "text/html; charset=utf-8",
      "Content-Length": Buffer.byteLength(body)
    });
    response.end(body);
    return;
  }

  if (url.pathname === "/api/verify-turnstile.php") {
    const payload = JSON.parse(await readBody(request));
    if (payload.token !== "turnstile-solved") {
      json(response, { ok: false, error: "bad turnstile token" }, 403);
      return;
    }
    json(response, { ok: true });
    return;
  }

  if (url.pathname === "/api/session_verify.php") {
    if (request.headers["x-token"] !== "page-token-123") {
      json(response, { ok: false, error: "bad page token" }, 403);
      return;
    }
    json(response, { ok: true, token: "session-token-456" });
    return;
  }

  if (url.pathname === "/api/accounts.php") {
    if (request.headers["x-token"] !== "session-token-456") {
      json(response, { error: "bad session token" }, 403);
      return;
    }
    json(response, [
      {
        id: 1,
        username: "demo@example.com",
        password: "abc123",
        message: "正常",
        last_check: "2026-08-12 10:00:00",
        last_check_success: 1,
        region_display: "美国",
        status: true
      }
    ]);
    return;
  }

  json(response, { error: "not found" }, 404);
});

try {
  const cfg = {
    id: "idfree_01",
    name: "小优 ID（idfree）",
    url: `http://127.0.0.1:${site.address().port}`,
    options: {
      captcha_solver: "capsolver",
      captcha_api_key: "test-key",
      captcha_api_url: `http://127.0.0.1:${solver.address().port}`,
      proxy_url: `http://127.0.0.1:${proxy.address().port}`,
      captcha_timeout_seconds: 10
    }
  };
  const provider = createProvider({ ...cfg, type: "idfree" });
  const accounts = await provider.fetch();
  if (accounts.length !== 1 || accounts[0].username !== "demo@example.com" || accounts[0].status !== "available") {
    throw new Error(`unexpected accounts: ${JSON.stringify(accounts)}`);
  }
  console.log(`[ok] mock idfree turnstile flow: ${accounts.length} account (${accounts[0].country})`);
} finally {
  site.close();
  proxy.close();
  solver.close();
}
