const DEFAULT_USER_AGENT =
  "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0 Safari/537.36";

function timeoutMs(cfg, fallbackSeconds) {
  const raw = cfg && cfg.options && cfg.options.request_timeout_seconds;
  const number = Number(raw);
  return (Number.isFinite(number) && number > 0 ? number : fallbackSeconds) * 1000;
}

function decodeBytes(bytes) {
  try {
    return new TextDecoder("utf-8", { fatal: true }).decode(bytes);
  } catch (_) {
    // Some text sources are GBK; fall back to a lenient utf-8 read when GBK is unavailable.
  }
  try {
    return new TextDecoder("gbk").decode(bytes);
  } catch (_) {
    return new TextDecoder("utf-8").decode(bytes);
  }
}

async function requestBytes(url, { method = "GET", headers = {}, body = null, timeoutMs = 15000 } = {}) {
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), timeoutMs);
  try {
    const response = await fetch(url, {
      method,
      headers,
      body,
      redirect: "follow",
      signal: controller.signal
    });
    if (!response.ok) {
      throw new Error(`HTTP ${response.status} for ${url}`);
    }
    const bytes = new Uint8Array(await response.arrayBuffer());
    return { bytes, response };
  } finally {
    clearTimeout(timer);
  }
}

async function requestText(url, options) {
  const { bytes } = await requestBytes(url, options);
  return decodeBytes(bytes);
}

async function requestJSON(url, options) {
  const text = await requestText(url, options);
  return JSON.parse(text);
}

function stringList(value) {
  if (Array.isArray(value)) {
    return value.map((item) => String(item).trim()).filter(Boolean);
  }
  if (typeof value === "string") {
    return value
      .split(",")
      .map((item) => item.trim())
      .filter(Boolean);
  }
  return [];
}

function collectShaCXUrls(cfg) {
  const urls = [];
  if (typeof cfg.url === "string" && cfg.url.trim()) {
    urls.push(cfg.url.trim());
  }
  if (cfg.options && cfg.options.urls) {
    urls.push(...stringList(cfg.options.urls));
  }
  return [...new Set(urls)];
}

function mapShaStatus(raw) {
  switch (raw) {
    case 0:
      return { status: "checking", status_label: "检测中", status_message: "账号正在检测，请稍后刷新" };
    case 1:
      return { status: "available", status_label: "可用", status_message: "检测正常，可登录 App Store" };
    case 2:
      return { status: "unavailable", status_label: "异常", status_message: "账号异常，请勿使用" };
    case 3:
      return { status: "pending", status_label: "待检测", status_message: "账号等待检测" };
    default:
      return { status: "unknown", status_label: "未知", status_message: "未知状态" };
  }
}

async function fetchShaCX(cfg) {
  const urls = collectShaCXUrls(cfg);
  if (urls.length === 0) {
    throw new Error(`sha_cx provider "${cfg.id}" requires url or options.urls`);
  }

  const results = await Promise.allSettled(
    urls.map(async (url) => {
      const html = await requestText(url, {
        timeoutMs: timeoutMs(cfg, 15),
        headers: {
          "User-Agent": DEFAULT_USER_AGENT,
          Accept: "text/html,application/xhtml+xml"
        }
      });
      const match = html.match(/ad\s*=\s*'([^']*)'/s);
      if (!match) {
        throw new Error("sha.cx payload not found in page");
      }
      const payload = match[1].trim();
      if (!payload) {
        throw new Error("sha.cx payload is empty");
      }
      let raw;
      try {
        raw = JSON.parse(payload);
      } catch (error) {
        throw new Error(`parse account payload: ${error.message}`);
      }
      return raw.map((item) => ({
        id: `${cfg.id}:${item.username}`,
        channel: cfg.id,
        channel_name: cfg.name,
        country: item.country || "",
        username: item.username || "",
        password: item.password || "",
        shadowrocket: true,
        ...mapShaStatus(item.status),
        raw_status: item.status,
        updated_at: item.time || "",
        source_url: url
      }));
    })
  );

  const accounts = [];
  const seen = new Set();
  const errors = [];
  for (const result of results) {
    if (result.status === "rejected") {
      errors.push(result.reason instanceof Error ? result.reason.message : String(result.reason));
      continue;
    }
    for (const account of result.value) {
      if (!account.username || seen.has(account.username)) {
        continue;
      }
      seen.add(account.username);
      accounts.push(account);
    }
  }
  if (errors.length > 0) {
    throw new Error(`fetch ${errors.length}/${urls.length} sha.cx sources: ${errors.join("; ")}`);
  }
  return accounts;
}

function mapTextStatus(raw, normalMessage, abnormalMessage, unknownMessage) {
  const text = String(raw || "");
  if (text.includes("正常")) {
    return { status: "available", status_label: "可用", status_message: normalMessage };
  }
  if (text.includes("异常") || text.includes("失败")) {
    return { status: "unavailable", status_label: "异常", status_message: abnormalMessage };
  }
  return { status: "pending", status_label: "待检测", status_message: unknownMessage };
}

async function fetchFanqiangnan(cfg) {
  if (!cfg.url) {
    throw new Error(`fanqiangnan provider "${cfg.id}" requires url`);
  }
  const raw = await requestJSON(cfg.url, {
    timeoutMs: timeoutMs(cfg, 15),
    headers: {
      "User-Agent": DEFAULT_USER_AGENT,
      Accept: "application/json"
    }
  });
  if (raw.success !== true) {
    throw new Error("fanqiangnan response success=false");
  }

  const groups = Object.keys((raw.data && raw.data.accounts) || {});
  groups.sort();
  const accounts = [];
  for (const group of groups) {
    for (const item of raw.data.accounts[group] || []) {
      const mapped = mapTextStatus(
        item.status,
        "上游检测正常",
        "上游检测异常",
        "上游暂未提供状态"
      );
      const country = String(item.regionName || "").trim() || String(item.region || "").trim();
      accounts.push({
        id: `${cfg.id}:${group}:${item.id}`,
        channel: cfg.id,
        channel_name: cfg.name,
        country,
        username: String(item.fullEmail || "").trim(),
        password: item.password || "",
        shadowrocket: false,
        ...mapped,
        updated_at: item.checkTime || "",
        source_url: cfg.url
      });
    }
  }
  return accounts;
}

function parseSetCookie(line) {
  const parts = line.split(";");
  const pair = (parts[0] || "").trim();
  const equals = pair.indexOf("=");
  if (equals === -1) {
    return null;
  }
  const name = pair.slice(0, equals).trim();
  const value = pair.slice(equals + 1).trim();
  let maxAge = null;
  let expires = null;
  for (const attribute of parts.slice(1)) {
    const separator = attribute.indexOf("=");
    const key = (separator === -1 ? attribute : attribute.slice(0, separator)).trim().toLowerCase();
    const rawValue = separator === -1 ? "" : attribute.slice(separator + 1).trim();
    if (key === "max-age") {
      maxAge = Number(rawValue);
    }
    if (key === "expires") {
      const parsed = Date.parse(rawValue);
      expires = Number.isNaN(parsed) ? null : parsed;
    }
  }
  if (maxAge === 0 || (expires !== null && expires < Date.now())) {
    return { name, remove: true };
  }
  return { name, value, remove: false };
}

function createCookieJar() {
  const cookies = new Map();
  return {
    capture(response) {
      const headers = response.headers;
      let values = [];
      if (typeof headers.getSetCookie === "function") {
        values = headers.getSetCookie();
      }
      if (values.length === 0 && headers.get("set-cookie")) {
        values = [headers.get("set-cookie")];
      }
      for (const line of values) {
        const cookie = parseSetCookie(line);
        if (!cookie) {
          continue;
        }
        if (cookie.remove) {
          cookies.delete(cookie.name);
        } else {
          cookies.set(cookie.name, cookie.value);
        }
      }
    },
    header() {
      return [...cookies.entries()]
        .map(([name, value]) => `${name}=${value}`)
        .join("; ");
    }
  };
}

function idfreeBaseHeaders(baseURL) {
  return {
    "User-Agent": DEFAULT_USER_AGENT,
    Accept: "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
    "Accept-Language": "zh-CN,zh;q=0.9,en;q=0.8",
    Referer: `${baseURL}/`,
    Origin: baseURL,
    "X-Requested-With": "XMLHttpRequest",
    "Cache-Control": "no-cache"
  };
}

function idfreeSolverConfig(cfg) {
  const options = cfg.options || {};
  return {
    solver: String(options.captcha_solver || "").trim().toLowerCase(),
    apiKey: String(options.captcha_api_key || "").trim()
  };
}

async function solveIdFreeTurnstile(cfg, websiteURL, websiteKey) {
  const { solver, apiKey } = idfreeSolverConfig(cfg);
  if (!solver || !apiKey) {
    throw new Error(
      "idfree 上游已开启 Cloudflare Turnstile，需要在 options 配置 captcha_solver 和 captcha_api_key（或环境变量 IDFREE_CAPTCHA_API_KEY）后恢复"
    );
  }
  const timeoutSeconds = Number(cfg.options && cfg.options.captcha_timeout_seconds) > 0
    ? Number(cfg.options.captcha_timeout_seconds)
    : 30;
  const requestTimeoutMs = timeoutMs(cfg, 15);

  if (solver === "capsolver") {
    const apiBase = String((cfg.options && cfg.options.captcha_api_url) || "")
      .trim()
      .replace(/\/+$/, "") || "https://api.capsolver.com";
    return solveCapsolverTurnstile(apiKey, apiBase, websiteURL, websiteKey, timeoutSeconds, requestTimeoutMs);
  }
  if (solver === "2captcha") {
    const apiBase = String((cfg.options && cfg.options.captcha_api_url) || "")
      .trim()
      .replace(/\/+$/, "") || "https://2captcha.com";
    return solve2CaptchaTurnstile(apiKey, apiBase, websiteURL, websiteKey, timeoutSeconds, requestTimeoutMs);
  }
  throw new Error(`idfree unsupported captcha_solver "${solver}" (supported: capsolver, 2captcha)`);
}

async function solveCapsolverTurnstile(apiKey, apiBase, websiteURL, websiteKey, timeoutSeconds, requestTimeoutMs) {
  const createTask = {
    clientKey: apiKey,
    task: {
      type: "AntiTurnstileTaskProxyLess",
      websiteURL,
      websiteKey
    }
  };
  const created = await requestJSON(`${apiBase}/createTask`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(createTask),
    timeoutMs: requestTimeoutMs
  });
  if (!created || !created.taskId) {
    throw new Error(`capsolver createTask failed: ${created && created.errorDescription || "missing taskId"}`);
  }

  const deadline = Date.now() + timeoutSeconds * 1000;
  while (Date.now() < deadline) {
    const result = await requestJSON(`${apiBase}/getTaskResult`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ clientKey: apiKey, taskId: created.taskId }),
      timeoutMs: requestTimeoutMs
    });
    if (result && result.status === "ready" && result.solution && result.solution.token) {
      return result.solution.token;
    }
    if (result && result.status === "failed") {
      throw new Error(`capsolver task failed: ${result.errorDescription || "unknown error"}`);
    }
    await new Promise((resolve) => setTimeout(resolve, 2000));
  }
  throw new Error("capsolver turnstile solve timed out");
}

async function solve2CaptchaTurnstile(apiKey, apiBase, websiteURL, websiteKey, timeoutSeconds, requestTimeoutMs) {
  const createParams = new URLSearchParams({
    key: apiKey,
    method: "turnstile",
    sitekey: websiteKey,
    pageurl: websiteURL,
    json: "1"
  });
  const created = await requestJSON(`${apiBase}/in.php`, {
    method: "POST",
    headers: { "Content-Type": "application/x-www-form-urlencoded" },
    body: createParams.toString(),
    timeoutMs: requestTimeoutMs
  });
  if (!created || created.status !== 1 || !created.request) {
    throw new Error(`2captcha in.php failed: ${created && (created.request || "unknown error")}`);
  }

  const deadline = Date.now() + timeoutSeconds * 1000;
  while (Date.now() < deadline) {
    const pollUrl = new URL(`${apiBase}/res.php`);
    pollUrl.searchParams.set("key", apiKey);
    pollUrl.searchParams.set("action", "get");
    pollUrl.searchParams.set("id", created.request);
    pollUrl.searchParams.set("json", "1");
    const result = await requestJSON(pollUrl.toString(), {
      timeoutMs: requestTimeoutMs
    });
    if (result && result.status === 1 && result.request) {
      return result.request;
    }
    if (result && result.status === 0 && typeof result.request === "string") {
      const code = result.request.toUpperCase();
      if (!["CAPCHA_NOT_READY", "CAPTCHA_NOT_READY"].includes(code)) {
        throw new Error(`2captcha task failed: ${result.request}`);
      }
    }
    await new Promise((resolve) => setTimeout(resolve, 3000));
  }
  throw new Error("2captcha turnstile solve timed out");
}

async function fetchIdFree(cfg) {
  const baseURL = String(cfg.url || "").trim().replace(/\/+$/, "");
  if (!baseURL) {
    throw new Error(`idfree provider "${cfg.id}" requires url`);
  }
  const proxyURL = String((cfg.options && cfg.options.proxy_url) || "")
    .trim()
    .replace(/\/+$/, "");
  const jar = createCookieJar();
  const timeout = timeoutMs(cfg, 20);

  async function idfreeRequest(path, extraHeaders = {}, method = "GET", body = null) {
    const headers = { ...idfreeBaseHeaders(baseURL), ...extraHeaders };
    const cookie = jar.header();
    if (cookie) {
      headers.Cookie = cookie;
    }
    const targetURL = `${baseURL}${path}`;
    const requestURL = proxyURL
      ? `${proxyURL}/fetch?url=${encodeURIComponent(targetURL)}`
      : targetURL;
    const { bytes, response } = await requestBytes(requestURL, {
      method,
      headers,
      body,
      timeoutMs: timeout
    });
    jar.capture(response);
    return bytes;
  }

  const page = decodeBytes(await idfreeRequest("/"));
  const tokenMatch = page.match(/<meta name="x-token" content="([^"]+)"/i);
  if (!tokenMatch) {
    throw new Error("idfree x-token not found in page");
  }
  const token = tokenMatch[1].trim();
  if (!token) {
    throw new Error("idfree x-token is empty");
  }

  const siteKeyMatch = page.match(/data-sitekey="([^"]+)"/i);
  if (siteKeyMatch && siteKeyMatch[1].trim()) {
    const turnstileToken = await solveIdFreeTurnstile(cfg, `${baseURL}/`, siteKeyMatch[1].trim());
    await idfreeRequest(
      "/api/verify-turnstile.php",
      {
        "Content-Type": "application/json"
      },
      "POST",
      JSON.stringify({ token: turnstileToken })
    );
  }

  const sessionBytes = await idfreeRequest(
    "/api/session_verify.php",
    {
      "Content-Type": "application/x-www-form-urlencoded",
      "X-Token": token
    },
    "POST",
    ""
  );
  let sessionToken = token;
  const sessionText = decodeBytes(sessionBytes).trim();
  if (sessionText) {
    let session;
    try {
      session = JSON.parse(sessionText);
    } catch (error) {
      throw new Error(`parse idfree session response: ${error.message}`);
    }
    if (session && session.ok === false) {
      throw new Error(`verify idfree session: ${session.error || "unknown error"}`);
    }
    if (session && typeof session.token === "string" && session.token.trim()) {
      sessionToken = session.token.trim();
    }
  }

  const body = decodeBytes(
    await idfreeRequest("/api/accounts.php", {
      Accept: "application/json, text/javascript, */*; q=0.01",
      "Accept-Language": "zh-CN,zh;q=0.9,en;q=0.8",
      "Sec-Fetch-Dest": "empty",
      "Sec-Fetch-Mode": "cors",
      "Sec-Fetch-Site": "same-origin",
      "X-Token": sessionToken
    })
  );
  if (body.includes("INVALID_BROWSER")) {
    throw new Error("idfree rejected browser headers");
  }

  let raw;
  try {
    raw = JSON.parse(body);
  } catch (error) {
    throw new Error(`parse idfree response: ${error.message}`);
  }

  return raw.map((item) => {
    const status = item.status === true;
    const message = String(item.message || "").trim();
    return {
      id: `${cfg.id}:${item.id}`,
      channel: cfg.id,
      channel_name: cfg.name,
      country: String(item.region_display || "").trim(),
      username: String(item.username || "").trim(),
      password: item.password || "",
      shadowrocket: false,
      status: status ? "available" : "unavailable",
      status_label: status ? "可用" : "异常",
      status_message: status
        ? message || "检测正常，可登录 App Store"
        : message || "账号异常，请勿使用",
      updated_at: item.last_check || "",
      source_url: `${baseURL}/`
    };
  });
}

async function fetchAppleidAPI(cfg) {
  if (!cfg.url) {
    throw new Error(`appleid_api provider "${cfg.id}" requires url`);
  }
  const raw = await requestJSON(cfg.url, {
    timeoutMs: timeoutMs(cfg, 15),
    headers: {
      "User-Agent": DEFAULT_USER_AGENT,
      Accept: "application/json"
    }
  });
  if (raw.success !== true) {
    throw new Error("appleid_api response success=false");
  }

  return (raw.data || []).map((item) => {
    const mapped = mapTextStatus(
      item.status,
      "检测正常，可登录 App Store",
      "账号异常，请勿使用",
      "上游暂未提供状态"
    );
    const accountID = String(item.id || "").trim() || String(item.email || "").trim();
    return {
      id: `${cfg.id}:${accountID}`,
      channel: cfg.id,
      channel_name: cfg.name,
      country: String(item.region || "").trim(),
      username: String(item.email || "").trim(),
      password: item.password || "",
      shadowrocket: false,
      ...mapped,
      updated_at: item.timestamp || item.time || "",
      source_url: cfg.url
    };
  });
}

function parseIOSAppText(content) {
  const account = {};
  for (const rawLine of content.split(/\r?\n/)) {
    const line = rawLine.replace(/^\uFEFF/, "").trim();
    if (!line) {
      continue;
    }
    let separator = line.indexOf(":");
    if (separator === -1) {
      separator = line.indexOf("：");
    }
    if (separator === -1) {
      continue;
    }
    const key = line.slice(0, separator).trim();
    const value = line.slice(separator + 1).trim();
    if (key === "类型") {
      account.Kind = value;
    } else if (key === "账号") {
      account.Account = value;
    } else if (key === "密码") {
      account.Password = value;
    } else if (key === "检查时间") {
      account.CheckTime = value;
    } else if (key === "状态") {
      account.Status = value;
    }
  }
  if (!account.Account) {
    throw new Error("账号字段缺失");
  }
  return account;
}

function mapIOSAppStatus(raw, checkTime) {
  const text = String(raw || "");
  if (text.includes("可用")) {
    return {
      status: "available",
      status_label: "可用",
      status_message: String(checkTime || "").trim()
        ? "文本源标记可用"
        : "文本源标记可用，但未提供检查时间，优先使用带检测时间的账号"
    };
  }
  if (text.includes("异常") || text.includes("失败")) {
    return {
      status: "unavailable",
      status_label: "异常",
      status_message: "文本源标记异常，请勿使用"
    };
  }
  return {
    status: "pending",
    status_label: "待检测",
    status_message: "文本源未提供明确状态"
  };
}

async function fetchIOSAppText(cfg) {
  const urls = stringList(cfg.options && cfg.options.urls);
  if (urls.length === 0 && typeof cfg.url === "string" && cfg.url.trim()) {
    urls.push(cfg.url.trim());
  }
  if (urls.length === 0) {
    throw new Error(`iosapp_text provider "${cfg.id}" requires options.urls`);
  }

  const results = await Promise.allSettled(
    urls.map(async (url) => {
      const text = await requestText(url, {
        timeoutMs: timeoutMs(cfg, 15),
        headers: { "User-Agent": DEFAULT_USER_AGENT }
      });
      const raw = parseIOSAppText(text);
      const accountID = String(url.split("/").pop() || "").replace(/\.[^.]+$/, "");
      return {
        id: `${cfg.id}:${accountID || String(urls.indexOf(url) + 1)}`,
        channel: cfg.id,
        channel_name: cfg.name,
        country: "",
        username: String(raw.Account || "").trim(),
        password: String(raw.Password || "").trim(),
        shadowrocket: false,
        ...mapIOSAppStatus(raw.Status, raw.CheckTime),
        priority: 1,
        updated_at: String(raw.CheckTime || "").trim(),
        source_url: url
      };
    })
  );

  const accounts = [];
  const errors = [];
  for (const result of results) {
    if (result.status === "rejected") {
      errors.push(result.reason instanceof Error ? result.reason.message : String(result.reason));
      continue;
    }
    accounts.push(result.value);
  }
  if (errors.length > 0) {
    throw new Error(`iosapp_text partial failure: ${errors.join("; ")}`);
  }
  return accounts;
}

function collectUnicornAccountObjects(value, accounts, depth = 0) {
  if (depth > 8 || !value || typeof value !== "object") {
    return;
  }
  if (Array.isArray(value)) {
    for (const item of value) {
      collectUnicornAccountObjects(item, accounts, depth + 1);
    }
    return;
  }
  const username = String(
    value.username || value.email || value.account || value.user || ""
  ).trim();
  const password = String(value.password || value.pass || value.pwd || "").trim();
  if (username && password && username.includes("@")) {
    accounts.push(value);
    return;
  }
  for (const key of Object.keys(value)) {
    collectUnicornAccountObjects(value[key], accounts, depth + 1);
  }
}

function parseUnicornAccountText(text) {
  const accounts = [];
  const emailPattern = /[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}/g;
  for (const rawLine of text.split(/\r?\n/)) {
    const line = rawLine
      .replace(/<[^>]+>/g, " ")
      .replace(/\u00a0/g, " ")
      .trim();
    let match;
    emailPattern.lastIndex = 0;
    while ((match = emailPattern.exec(line))) {
      const username = match[0];
      let tail = line
        .slice(match.index + username.length)
        .replace(/^[\s:：|,，、]+/, "");
      tail = tail.replace(/^(密码|password|pwd)([:：]|\s+)\s*/i, "");
      const passwordMatch = tail.match(/^([^\s|,，;；]+)/);
      if (passwordMatch && passwordMatch[1].length >= 4) {
        accounts.push({ username, password: passwordMatch[1] });
      }
    }
  }
  return accounts;
}

function mapUnicornStatus(value) {
  const text = String(value || "").trim().toLowerCase();
  const availableTokens = ["正常", "可用", "成功", "valid", "ok", "available", "success"];
  const unavailableTokens = ["异常", "失效", "不可用", "已失效", "失败", "invalid", "unavailable", "failed", "error"];
  if (availableTokens.some((token) => text.includes(token))) {
    return { status: "available", status_label: "可用", status_message: "知识库标记可用，建议使用前再确认" };
  }
  if (unavailableTokens.some((token) => text.includes(token))) {
    return { status: "unavailable", status_label: "异常", status_message: "知识库标记异常，请勿使用" };
  }
  return { status: "pending", status_label: "待确认", status_message: "知识库未提供明确状态" };
}

async function fetchUnicornKnowledge(cfg) {
  const url = String(cfg.url || "").trim();
  if (!url) {
    throw new Error(`unicorn_knowledge provider "${cfg.id}" requires url`);
  }
  const headers = {
    "User-Agent": DEFAULT_USER_AGENT,
    Accept: "application/json, text/plain, text/html, */*"
  };
  const token = String((cfg.options && cfg.options.token) || "").trim();
  if (token) {
    headers.Authorization = token.startsWith("Bearer ") ? token : `Bearer ${token}`;
  }

  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), timeoutMs(cfg, 20));
  let response;
  try {
    response = await fetch(url, {
      headers,
      redirect: "follow",
      signal: controller.signal
    });
  } finally {
    clearTimeout(timer);
  }
  const contentType = response.headers.get("content-type") || "";
  const bytes = new Uint8Array(await response.arrayBuffer());
  const text = decodeBytes(bytes);

  let parsed = null;
  if (contentType.includes("json") || text.trim().startsWith("{")) {
    try {
      parsed = JSON.parse(text);
    } catch (_) {
      // fall through to plain text parsing below
    }
  }
  if (!response.ok) {
    const hint = unicornErrorHint(parsed);
    throw new Error(
      hint ? `unicorn_knowledge: source returned HTTP ${response.status}: ${hint}` : `unicorn_knowledge: HTTP ${response.status} for ${url}`
    );
  }

  const rawAccounts = [];
  if (parsed) {
    collectUnicornAccountObjects(parsed, rawAccounts);
  }
  if (rawAccounts.length === 0) {
    rawAccounts.push(...parseUnicornAccountText(text));
  }
  if (rawAccounts.length === 0) {
    const hint = unicornErrorHint(parsed);
    throw new Error(
      hint ? `unicorn_knowledge: ${hint}` : "unicorn_knowledge: no accounts found in knowledge response"
    );
  }

  const accounts = [];
  const seen = new Set();
  rawAccounts.forEach((item, index) => {
    const username = String(
      item.username || item.email || item.account || item.user || ""
    ).trim();
    if (!username || seen.has(username)) {
      return;
    }
    seen.add(username);
    const password = String(item.password || item.pass || item.pwd || "").trim();
    const country = String(item.country || item.region || item.location || "").trim();
    const rawStatus = String(item.status || item.state || item.result || "").trim();
    accounts.push({
      id: `${cfg.id}:${username}`,
      channel: cfg.id,
      channel_name: cfg.name,
      country,
      username,
      password,
      shadowrocket: true,
      ...mapUnicornStatus(rawStatus),
      priority: index,
      updated_at: String(item.updated_at || item.time || item.check_time || "").trim(),
      source_url: url
    });
  });
  return accounts;
}

function unicornErrorHint(parsed) {
  if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
    return "";
  }
  for (const key of ["message", "msg", "error", "reason"]) {
    if (typeof parsed[key] === "string" && parsed[key].trim()) {
      return parsed[key].trim();
    }
  }
  return "";
}

export function createProvider(cfg) {
  switch (cfg.type) {
    case "sha_cx":
      return { id: cfg.id, name: cfg.name, fetch: () => fetchShaCX(cfg) };
    case "fanqiangnan":
      return { id: cfg.id, name: cfg.name, fetch: () => fetchFanqiangnan(cfg) };
    case "idfree":
      return { id: cfg.id, name: cfg.name, fetch: () => fetchIdFree(cfg) };
    case "appleid_api":
      return { id: cfg.id, name: cfg.name, fetch: () => fetchAppleidAPI(cfg) };
    case "iosapp_text":
      return { id: cfg.id, name: cfg.name, fetch: () => fetchIOSAppText(cfg) };
    case "unicorn_knowledge":
      return { id: cfg.id, name: cfg.name, fetch: () => fetchUnicornKnowledge(cfg) };
    default:
      throw new Error(`unknown provider type "${cfg.type}"`);
  }
}

export function buildProviders(config) {
  return config.providers.map((provider) => createProvider(provider));
}
