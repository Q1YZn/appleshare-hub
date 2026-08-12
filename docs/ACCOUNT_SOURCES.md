# 账号渠道接入文档

## 1. 当前渠道

项目通过 `internal/provider` 下的 `Provider` 接口统一接入多渠道，当前默认启用以下来源：

| 配置 ID | 类型 | 名称 | 来源 |
| --- | --- | --- | --- |
| `sha_cx_01` | `sha_cx` | 渠道 A（sha.cx） | <https://d8p8e.sha.cx/51e8990f678655f7749dfa8c5598dfbd> 与 <https://7y6h5.sha.cx/23cfa3c22135050d45f82283f2ef6e7f> |
| `fanqiangnan_01` | `fanqiangnan` | 翻墙男（fanqiangnan） | <https://fanqiangnan.com/data_sync.php> |
| `idfree_01` | `idfree` | 小优 ID（idfree） | <https://idfree.top/> |
| `appleid_api_01` | `appleid_api` | 云码酷（appleid.uczyw.us） | <https://appleid.uczyw.us/api/accounts> |
| `iosapp_text_01` | `iosapp_text` | 免费文本源（iosapp.icu） | <https://free.iosapp.icu/go-rod/1.txt> 等 1-3 |
| `unicorn_knowledge_01` | `unicorn_knowledge` | 独角兽知识库（91unicorn） | <https://91unicorn.cloud/api/v1/user/knowledge/fetch?id=34&language=zh-CN> |

另有已配置但默认关闭的备用渠道：

| 配置 ID | 类型 | 名称 | 来源 |
| --- | --- | --- | --- |
| `appleid_api_02` | `appleid_api` | 云码酷备用 | <https://appleid2.uczyw.us/api/accounts> |

## 2. 各渠道响应格式

### 2.1 sha.cx

分享页会把账号数组直接写在页面 JavaScript 变量中：

```html
<script>
  var ad='[...]';
</script>
```

其中 `ad` 是 JSON 数组，示例：

```json
[
  {
    "check": 1786035047,
    "country": "美国",
    "msg": "解锁成功",
    "password": "y5#e76CT",
    "status": 1,
    "time": "2026-08-07 00:50:47",
    "user": 34,
    "username": "trevorthompson2227@outlook.com"
  }
]
```

`status` 映射规则见 `internal/provider/shacx.go` 的 `mapShaCXStatus`。

两个 sha.cx 分享页默认合并为同一个“渠道 A”，配置方式如下，账号按用户名去重：

```json
{
  "id": "sha_cx_01",
  "type": "sha_cx",
  "name": "渠道 A（sha.cx）",
  "enabled": true,
  "options": {
    "urls": [
      "https://d8p8e.sha.cx/51e8990f678655f7749dfa8c5598dfbd",
      "https://7y6h5.sha.cx/23cfa3c22135050d45f82283f2ef6e7f"
    ]
  }
}
```

兼容旧的单个 `url` 字段；同时配置 `url` 与 `options.urls` 时会合并并去重。

### 2.2 fanqiangnan

`GET https://fanqiangnan.com/data_sync.php` 返回无鉴权 JSON：

```json
{
  "success": true,
  "timestamp": 1786071481,
  "data": {
    "accounts": {
      "group1": [
        {
          "id": "1-1",
          "fullEmail": "user@example.com",
          "password": "****",
          "status": "正常",
          "checkTime": "2026-08-07 01:59:50",
          "region": "US",
          "regionName": "美国"
        }
      ],
      "group2": []
    },
    "vpn_ads": []
  }
}
```

接入要点：

- `data.accounts` 是分组 Map，遍历时需要忽略顺序差异，`vpn_ads` 直接忽略。
- `status == "正常"` 映射为 `available`，`checkTime` 作为更新时间。
- 地区优先使用 `regionName`，缺失时退回 `region`。

### 2.3 idfree.top

上游在 2026-08-12 新增了 Cloudflare Turnstile 人机验证，当前抓取流程如下：

```text
1. GET  https://idfree.top/                          获取 Cookie、<meta name="x-token">、data-sitekey
2. POST https://idfree.top/api/verify-turnstile.php  携带 `{"token":"<验证服务解出的 token>"}`
3. POST https://idfree.top/api/session_verify.php    换取会话 token
4. GET  https://idfree.top/api/accounts.php          携带 Cookie、会话 token、浏览器头
```

`session_verify.php` 未完成 Turnstile 时会返回 `403`，内容为 `{"ok":false,"error":"请先完成人机验证","code":"TURNSTILE_REQUIRED"}`；`accounts.php` 未通过验证时返回 `{"error":"未通过安全验证，或验证已过期","code":"VERIFICATION_REQUIRED"}`。因此必须先通过验证服务（Capsolver / 2captcha）解出 Turnstile token，再执行会话流程。

返回顶层 JSON 数组：

```json
[
  {
    "id": 9900,
    "username": "user@example.com",
    "password": "****",
    "message": "正常",
    "last_check": "2026-08-07 11:27:26",
    "last_check_success": 1,
    "region_display": "美国",
    "status": true
  }
]
```

接入要点：

- 使用同一个 `http.Client` 和 Cookie Jar，保证会话 cookie 连续。
- 首页 HTML 存在 `data-sitekey` 时，先用验证服务解 Turnstile token，再 POST `/api/verify-turnstile.php`。
- `session_verify.php` 返回的 `token` 会作为后续 `/api/accounts.php` 的 `X-Token`；没有返回时沿用首页 `x-token`。
- 缺 `Accept`、`Sec-Fetch-*`、`X-Requested-With` 等浏览器头会返回 `INVALID_BROWSER`。
- `status: true` 映射为 `available`，`last_check` 作为更新时间。
- 完整请求头在 `internal/provider/idfree.go` 的 `baseHeaders` 中维护。

渠道配置示例（`options` 中的 API key 留空时，程序会给出配置类错误而不是误报网络错误）：

```json
{
  "id": "idfree_01",
  "type": "idfree",
  "name": "小优 ID（idfree）",
  "url": "https://idfree.top/",
  "enabled": true,
  "options": {
    "captcha_solver": "capsolver",
    "captcha_api_key": "",
    "proxy_url": "",
    "captcha_timeout_seconds": 30
  }
}
```

也可用环境变量 `IDFREE_CAPTCHA_SOLVER`、`IDFREE_CAPTCHA_API_KEY` 和 `IDFREE_PROXY_URL` 覆盖，避免把付费 key 写进仓库或配置快照。

2026-08-12 实测：idfree 不是“仅国内可访问”，而是按出口 IP 黑名单放行；本机普通出口（LAX）与日本住宅/ISP 出口（NRT）均返回 200 且带 `x-token`/`data-sitekey`，Cloudflare 数据中心出口返回 `Blocked: blacklisted IP`。所以 Cloudflare Worker 抓取时必须在 `proxy_url` 配置一个未被拉黑的代理出口，否则会在第一步 GET 首页就拿到 403。

注意 `IDFREE_PROXY_URL` 不是 `http://user:pass@host:port` 这种标准 HTTP 代理格式。Worker 的 `fetch()` 不支持标准代理环境变量，代码会把请求拼成 `<proxy_url>/fetch?url=<URL编码后的目标地址>`（见 `functions/_lib/providers.js` 的 `fetchIdFree` 与 `internal/provider/idfree.go` 的 `requestURL`）。也就是说需要提供一个 fetch 代理服务：接收 `GET /fetch?url=...`，代为请求目标并原样返回响应头与正文。可自建一个仅做转发的小型服务/Worker，只要出口 IP 不被 idfree 拉黑即可。

### 2.4 appleid.uczyw.us

`GET https://appleid.uczyw.us/api/accounts` 返回无鉴权 JSON：

```json
{
  "success": true,
  "count": 42,
  "data": [
    {
      "id": "481c86cd",
      "email": "user@example.com",
      "password": "****",
      "region": "美国",
      "status": "正常",
      "source": "ccbaohe",
      "timestamp": "2026-08-07T03:29:46.423Z"
    }
  ]
}
```

接入要点：

- 解析 `data` 数组，`status == "正常"` 映射为 `available`。
- `timestamp` 作为更新时间，`region` 作为地区。
- 备用域名 `https://appleid2.uczyw.us/api/accounts` 结构一致。

### 2.5 free.iosapp.icu

纯文本文件，默认抓取 1-3 号：

```text
类型:
账号: user@example.com
密码: ****
检查时间:
状态: 账号可用
```

接入要点：

- 每文件 1 个账号，按 `账号`、`密码`、`检查时间`、`状态` 行解析。
- 当前文件未提供检查时间，页面会标注“文本源，未提供检查时间，优先使用带检测时间的账号”，属于低优先级补充源。
- `model.Account.Priority = 1` 会让这类账号在可用状态下排到带检测时间的账号后面。
- 首页是 GBK，文本文件本身可能是 UTF-8 或 GBK；`decodeMaybeGBK` 会自动处理。

### 2.6 91unicorn 知识库

`GET https://91unicorn.cloud/api/v1/user/knowledge/fetch?id=34&language=zh-CN` 是登录后的知识库接口，需要携带登录 token：

```text
Authorization: Bearer <登录后从 localStorage 取到的 token>
```

未登录或 token 过期时返回 `403`：

```json
{
  "message": "未登录或登陆已过期"
}
```

接入要点：

- 该接口有 Cloudflare 防护且必须登录，程序在无 token 时会把渠道标记为错误并返回上游原文提示，不影响其他渠道。
- token 可通过配置 `options.token` 注入，或设置环境变量 `UNICORN_TOKEN`，程序会自动补成 `Bearer <token>`。
- 返回结构暂未拿到真实成功样例，代码同时支持 JSON 递归查找（`username/email/account` + `password/pass/pwd`）和纯文本邮箱密码行兜底。
- 91unicorn 知识库是作者整理的小火箭账号来源，当前实现把该渠道账号标记为“确认有 Shadowrocket”，前端可用“有Shadowrocket / 不确定”筛选。
- 该渠道需要登录态且结构可能随知识库页面变化，优先级低于公开 JSON 接口。

## 3. 状态映射

| 本项目 status | 展示文案 | 说明 |
| --- | --- | --- |
| `available` | 可用 | 上游检测正常，可登录 App Store |
| `checking` | 检测中 | 账号正在检测，请稍后刷新 |
| `pending` | 待检测 | 账号等待检测或状态不明 |
| `unavailable` | 异常 | 账号异常，请勿使用 |
| `unknown` | 未知 | 未知状态 |

## 4. 新增渠道

新增渠道只需要三步：

1. 在 `internal/provider` 下新建 `xxx.go`，实现 `Provider` 接口。
2. 在 `init()` 中调用 `Register("xxx", factory)`。
3. 在 `config.json` 中增加 `providers` 配置项。

前端和 API 依赖统一的 `model.Account`，不需要改动。

## 5. 关于账号密码验证

本项目不直接调用 Apple 登录接口验证共享账号，原因如下：

- 服务器 IP 批量登录 Apple 账号容易触发 Apple 风控、双重认证和设备锁机。
- 共享账号的密码一旦被上游修改，探测登录还可能把正常账号推向锁定状态。

当前策略是把各上游自己的 `checkTime / last_check / timestamp / status` 当作可用性来源，并在页面展示上游更新时间。如果后续确定要接入 Apple 登录探测，需要先评估风控风险并单独设计节流、代理和验证频率。

## 6. 注意事项

- 上游页面结构可能变化，建议定期验证各 Provider 的解析逻辑。
- 每个渠道可配置 `request_timeout_seconds`，避免单渠道拖慢整个刷新。
- 上游返回的账号密码仅用于展示，不要写入日志。
- 如果某个渠道状态字段含义不同，在对应 Provider 中完成映射，不要污染公共模型。
