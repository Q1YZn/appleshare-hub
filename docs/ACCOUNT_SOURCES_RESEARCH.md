# 苹果共享账号来源调研

> 状态：调研完成，fanqiangnan、idfree、appleid.uczyw.us、free.iosapp.icu、91unicorn 知识库已接入代码。

## 1. 调研说明

- 调研日期：2026-08-07
- 调研方式：谷歌搜索语法、目标站点实测、接口请求验证
- 验证环境：Windows PowerShell + `curl.exe`
- 结论：找到 5 个可脚本抓取的候选来源，其中 3 个为公开 JSON / 文本接口，1 个需要模拟浏览器会话，1 个需要登录 token。

## 2. 使用的搜索语法

本轮实际使用的语法：

```text
"苹果账号" "免费共享" "App Store"
"Apple ID" "共享账号" password "App Store"
"苹果共享账号" "自动检测" 免费
"免费苹果ID" "共享" 密码
"美区Apple ID" 共享 自动检测
苹果账号 分享 网站 API 账号池
```

后续可继续尝试：

```text
site:github.com apple id 共享 账号
"免费苹果ID" "共享" 密码 更新时间
"Apple ID" "共享账号" password 更新时间
"美区Apple ID" 共享 自动检测 2026
苹果账号 分享 网站 接口 账号池
```

## 3. 候选来源一览

| 来源 | 入口 / 接口 | 鉴权 | 数据格式 | 实测结果 | 抓取难度 | 状态 |
| --- | --- | --- | --- | --- | --- | --- |
| 翻墙男 | `https://fanqiangnan.com/appleid.html`<br>`https://fanqiangnan.com/data_sync.php` | 无 | JSON | 200，约 41 个账号 | 低 | 已接入 |
| 小优 ID | `https://idfree.top/`<br>`https://idfree.top/api/accounts.php` | Turnstile + Cookie + 会话 token | JSON 数组 | 需 Turnstile 求解 | 中 | 已接入（需验证服务 key） |
| CCKDN 云码酷 | `https://appleid.uczyw.us/api/accounts`<br>`https://appleid2.uczyw.us/api/accounts` | 无 | JSON | 200，count=42 | 低 | 已接入（备用源默认关闭） |
| free.iosapp.icu | `https://free.iosapp.icu/go-rod/1.txt`<br>`.../2.txt`<br>`.../3.txt` | 无 | 纯文本 | 200，每文件 1 个账号 | 低 | 已接入（低优先级） |
| 91unicorn 知识库 | `https://91unicorn.cloud/api/v1/user/knowledge/fetch?id=34&language=zh-CN` | 登录 token | JSON/富文本 | 403，`未登录或登陆已过期` | 中 | 已接入（需登录 token） |
| Applo / mp.499599.xyz | 需账号登录、订单购买 | 高 | JSON | 无公开账号 | 高 | 暂不接入 |
| GitHub 仓库 Markdown | `janhaas1980-south/Apple-ID-Public-Share` | 无 | Markdown 表格 | 200，密码未完全公开 | 中 | 暂不接入 |
| 论坛 / 聚合页 | resohub、linux.do、nodeloc、moyunews | 无 | 页面 | 403 或仅为推广介绍 | - | 暂不接入 |

## 4. 各来源接口详情

### 4.1 翻墙男 `fanqiangnan.com`

数据接口：

```text
GET https://fanqiangnan.com/data_sync.php
```

验证命令：

```powershell
curl.exe -s -L --max-time 15 https://fanqiangnan.com/data_sync.php
```

返回结构（密码已脱敏）：

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

要点：

- 无鉴权、无复杂风控，直接返回 JSON。
- `group1` 是账号主体，`group2` 当前为空。
- `vpn_ads` 是广告数据，接入时忽略。
- 上游声称每 30 分钟自动检测一次，实测 `checkTime` 字段存在且为当天。

### 4.2 小优 ID `idfree.top`

完整流程（2026-08-12 起新增 Cloudflare Turnstile）：

```text
1. GET  https://idfree.top/                          获取 Cookie、<meta name="x-token">、data-sitekey
2. POST https://idfree.top/api/verify-turnstile.php  携带 `{"token":"<验证服务解出的 token>"}`
3. POST https://idfree.top/api/session_verify.php    换取会话 token
4. GET  https://idfree.top/api/accounts.php          携带 Cookie、会话 token、浏览器头
```

不经过 Turnstile 时，`session_verify.php` 返回 403：

```json
{"ok":false,"error":"请先完成人机验证","code":"TURNSTILE_REQUIRED"}
```

`accounts.php` 直接请求返回 403：

```json
{"error":"未通过安全验证，或验证已过期","code":"VERIFICATION_REQUIRED"}
```

旧验证命令（PowerShell，已无法直接跑通，仅保留协议参考）：

```powershell
$jar = Join-Path $env:TEMP "idfree_cookies.txt"
$ua = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0 Safari/537.36"

curl.exe -s -c $jar -A $ua https://idfree.top/ | Out-Null

$page = curl.exe -s -c $jar -A $ua https://idfree.top/

$token = [regex]::Match(
  ($page -join "`n"),
  '<meta name="x-token" content="([^"]+)"'
).Groups[1].Value

curl.exe -s -b $jar -c $jar -A $ua `
  -H "X-Token: $token" `
  -H "Referer: https://idfree.top/" `
  -H "Origin: https://idfree.top" `
  -H "X-Requested-With: XMLHttpRequest" `
  -X POST https://idfree.top/api/session_verify.php

curl.exe -s -b $jar -A $ua `
  -H "Accept: application/json, text/javascript, */*; q=0.01" `
  -H "Accept-Language: zh-CN,zh;q=0.9,en;q=0.8" `
  -H "Sec-Fetch-Dest: empty" `
  -H "Sec-Fetch-Mode: cors" `
  -H "Sec-Fetch-Site: same-origin" `
  -H "X-Token: $token" `
  -H "Referer: https://idfree.top/" `
  -H "X-Requested-With: XMLHttpRequest" `
  https://idfree.top/api/accounts.php
```

返回结构（密码已脱敏）：

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

要点：

- 缺少 `Accept` / `Sec-Fetch-*` / `X-Requested-With` 等浏览器头会返回 `INVALID_BROWSER`。
- 需要维护 Cookie 和会话 token，建议接入时复用同一个 HTTP Client。
- Turnstile 求解需要配置 `captcha_solver`（`capsolver` / `2captcha`）和 `captcha_api_key`，或用环境变量 `IDFREE_CAPTCHA_SOLVER` / `IDFREE_CAPTCHA_API_KEY` 注入。
- 2026-08-12 实测 Cloudflare 数据中心出口 IP 被上游拉黑（`Blocked: blacklisted IP`）；本机普通出口（LAX）与日本住宅/ISP 出口（NRT）均返回 200 且带 `x-token`/`data-sitekey`，说明上游按 IP 黑名单放行，不是“仅国内可访问”。Worker 侧需要配置 `IDFREE_PROXY_URL` 或 `options.proxy_url` 走非数据中心代理。
- `IDFREE_PROXY_URL` 必须是一个 fetch 代理服务地址（程序拼 `<proxy_url>/fetch?url=<目标地址>`），不能直接填标准 HTTP 代理 `http://user:pass@host:port`。
- 当前账号量较小，但字段规范、状态明确。

### 4.3 CCKDN 云码酷 `appleid.uczyw.us`

公开接口：

```text
GET https://appleid.uczyw.us/api/accounts
GET https://appleid2.uczyw.us/api/accounts
```

验证命令：

```powershell
curl.exe -s -L --max-time 15 https://appleid.uczyw.us/api/accounts
```

返回结构（密码已脱敏）：

```json
{
  "success": true,
  "count": 42,
  "data": [
    {
      "id": "481c86cd",
      "email": "user@example.com",
      "password": "****",
      "masked_email": "us****@example.com",
      "region": "美国",
      "status": "正常",
      "source": "ccbaohe",
      "timestamp": "2026-08-07T03:29:46.423Z"
    }
  ]
}
```

要点：

- 无鉴权 JSON，字段完整，适合做第二或第三数据源。
- 两个域名返回结构一致，可配置为主备。
- 页面另有密码验证接口 `https://www.cckdn.cn/source/plugin/webzxnet_urlattach/mima.php`，实测可返回纯文本密码，但 API 已直接给出密码，不必再依赖页面接口。

### 4.4 `free.iosapp.icu`

纯文本文件：

```text
https://free.iosapp.icu/go-rod/1.txt
https://free.iosapp.icu/go-rod/2.txt
https://free.iosapp.icu/go-rod/3.txt
```

验证命令：

```powershell
curl.exe -s -L --max-time 15 https://free.iosapp.icu/go-rod/1.txt
```

返回示例（密码已脱敏）：

```text
类型:
账号: user@example.com
密码: ****
检查时间:
状态: 账号可用
```

要点：

- 每文件仅 1 个账号，数量少，只适合作为补充源。
- 首页为 GBK 编码，但文本文件内容简单，按行解析即可。

### 4.5 91unicorn 知识库

`https://91unicorn.cloud/#/knowledge` 是登录后的知识库页面，接口：

```text
GET https://91unicorn.cloud/api/v1/user/knowledge/fetch?id=34&language=zh-CN
Authorization: Bearer <token>
```

2026-08-12 实测：

- 未登录直接请求返回 HTTP 403，正文 `{"message":"未登录或登陆已过期"}`。
- 前端源码确认接口要求 `Authorization`，token 存放在浏览器 `localStorage`。
- 登录后抓到的真实账号来自个人知识库（作者整理），不是公开接口；页面会按顺序展示账号条目，接口成功响应结构需要在登录态下抓一次真实样例后按实际字段完善解析。
- 注意：本机 `curl.exe` 能到达接口并拿到 403；Node.js `fetch` 直连该域名曾出现 `ConnectTimeoutError`，属于出口网络对 91unicorn 的连通性差异。页面渠道错误会显示上游原文或网络错误，不会拖垮其他渠道。

验证命令（替换 token）：

```powershell
curl.exe -s --max-time 20 `
  -H "Authorization: Bearer <token>" `
  -H "Accept: application/json, text/plain, text/html, */*" `
  "https://91unicorn.cloud/api/v1/user/knowledge/fetch?id=34&language=zh-CN"
```

接入方式：

- 配置 `UNICORN_TOKEN` 环境变量或渠道 `options.token`。
- 无 token 时渠道显示“未登录或登陆已过期”，不影响其他渠道。
- 当前实现标记为确认有 Shadowrocket。

## 5. 与现有 sha.cx 渠道的对比

| 对比项 | sha.cx | fanqiangnan | idfree.top | appleid.uczyw.us | free.iosapp.icu | 91unicorn |
| --- | --- | --- | --- | --- | --- | --- |
| 响应形式 | 页面内嵌 `ad` JSON | 直接 JSON | JSON 数组 | JSON | 纯文本 | JSON/富文本 |
| 鉴权 | 无 | 无 | Cookie + Token + 浏览器头 | 无 | 无 | Bearer token |
| 账号量 | 2 个分享页 | 约 41 个 | 少量 | 42 个 | 3 个 | 待登录后实测 |
| 更新时间 | 页面内字段 | `checkTime` | `last_check` | `timestamp` | 文本内字段 | 待登录后实测 |
| 接入成本 | 已实现 | 低 | 中 | 低 | 低 | 中 |

现有 `sha_cx` 是 HTML 解析，候选源大多是 JSON，接入时只需各自实现 `Provider` 的 `Fetch` 并映射到统一的 `model.Account`，前端和 API 无需改动。

## 6. 接入优先级建议

1. `fanqiangnan.com/data_sync.php`：无鉴权、数据量大、成本最低，已接入。
2. `appleid.uczyw.us/api/accounts`：公开 JSON，两个域名可做冗余，已接入。
3. `idfree.top/api/accounts.php`：已有浏览器风控，2026-08-12 起新增 Turnstile，需要配置 Capsolver / 2captcha 验证服务后恢复。
4. `free.iosapp.icu/go-rod/*.txt`：解析简单，数据量小，已作为低优先级补充源接入。
5. 91unicorn 知识库：需要登录 token，已在代码中接入；用户登录后配置 `UNICORN_TOKEN` 即可启用，成功后建议再按真实响应结构完善解析。
6. Applo、GitHub Markdown、论坛聚合页：暂不接入，原因见候选来源一览。

## 7. 风险提示

- 上游接口可能随时变更结构、增加风控或关闭访问，接入时保留超时、失败降级和缓存。
- 共享账号本身存在被上游停用、被他人修改密码、触发苹果锁定的风险，页面必须持续展示“仅 App Store 登录、iOS 26 从设置退出”的提示。
- 不要在日志、文档或仓库中保存真实账号密码；本调研文档中的密码均为脱敏示例。
- 数据仅用于技术测试，请遵守上游站点规则、Apple 服务条款和当地法律。
