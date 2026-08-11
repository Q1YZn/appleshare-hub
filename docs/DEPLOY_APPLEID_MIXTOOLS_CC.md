# 挂载 appleid.mixtools.cc

目标：把 Cloudflare Pages 项目 `appleshare-hub` 挂到 `appleid.mixtools.cc`。

`mixtools.cc` 已经是 Cloudflare 管理的 zone，NS 指向：

```text
rosalyn.ns.cloudflare.com
vern.ns.cloudflare.com
```

所以不需要手动去 DNS 商加 CNAME，只要在同一个 Cloudflare 账户里完成 Pages 项目自定义域名即可。

## 1. 开通 R2 订阅（必须先做）

如果执行 `npx wrangler r2 bucket list` 报错：

```text
Please enable R2 through the Cloudflare Dashboard. [code: 10042]
```

说明账号还没有 R2 订阅，需要先完成 Dashboard 的开通流程：

1. 打开 [R2 Overview](https://dash.cloudflare.com/e23a0b1aabd2176a708ca168b943fffd/r2/overview)
2. 点击 **Add R2 subscription to my account** / **Purchase R2**
3. 完成 checkout 流程
4. 回到这里执行：

```bash
npx wrangler r2 bucket list
```

看到 Buckets 列表为空即开通成功。

官方说明：<https://developers.cloudflare.com/r2/get-started/>

## 2. 登录并确认账号

```bash
npx wrangler whoami
```

如果未登录：

```bash
npx wrangler login
```

## 3. 创建 R2 Bucket

```bash
npx wrangler r2 bucket create appleshare-hub
```

已存在会提示 Bucket already exists，不影响。

## 4. 部署 Pages

```bash
npm install
npm run pages:deploy
```

这会创建 Pages 项目 `appleshare-hub`，默认域名：

```text
https://appleshare-hub.pages.dev
```

## 5. 配置 Pages 环境变量和 R2 Binding

在 Cloudflare Dashboard：

1. Workers & Pages -> `appleshare-hub`
2. Settings -> Functions -> R2 bucket bindings
3. 变量名填 `BUCKET`，选择 `appleshare-hub`
4. Settings -> Environment variables 添加：
   - `PROVIDER_CONFIG`：把 `cloudflare.config.example.json` 内容整理为单行 JSON
   - `REFRESH_TOKEN`：随机字符串，用于保护 `/api/refresh`
   - 可选：`RETAIN_HISTORY=0`

Windows 下不要用 PowerShell 管道执行 `wrangler pages secret put`，否则中文和 token 会被写坏（常见症状：渠道名变成 `?`、token 前面多出 BOM）。推荐使用仓库里的 UTF-8 管道脚本：

```bash
node -e "const fs=require('fs');const v=fs.readFileSync('cloudflare.config.example.json','utf8').trim();require('child_process').spawnSync(process.execPath,['scripts/upload-secret-via-wrangler.mjs','pages','PROVIDER_CONFIG',v],{stdio:'inherit'})"
node -e "const fs=require('fs');const s=fs.readFileSync('.env','utf8');const t=s.match(/REFRESH_TOKEN=(.+)/)[1].trim();require('child_process').spawnSync(process.execPath,['scripts/upload-secret-via-wrangler.mjs','pages','REFRESH_TOKEN',t],{stdio:'inherit'})"
```

## 6. 部署 Cron Worker

```bash
npm run worker:deploy
```

Worker 名称是 `appleshare-hub-cron`。部署后在 Dashboard 确认：

```text
Triggers -> Cron Triggers -> */5 * * * *
```

Worker 也配置 `PROVIDER_CONFIG` 和 `REFRESH_TOKEN`，R2 binding 与 Pages 一致。

Worker 的 secret 同样用 UTF-8 管道脚本，参数改为 `worker`：

```bash
node -e "const fs=require('fs');const v=fs.readFileSync('cloudflare.config.example.json','utf8').trim();require('child_process').spawnSync(process.execPath,['scripts/upload-secret-via-wrangler.mjs','worker','PROVIDER_CONFIG',v],{stdio:'inherit'})"
node -e "const fs=require('fs');const s=fs.readFileSync('.env','utf8');const t=s.match(/REFRESH_TOKEN=(.+)/)[1].trim();require('child_process').spawnSync(process.execPath,['scripts/upload-secret-via-wrangler.mjs','worker','REFRESH_TOKEN',t],{stdio:'inherit'})"
```

## 7. 添加自定义域名

Cloudflare Dashboard 操作：

1. Workers & Pages -> `appleshare-hub`
2. 进入 `Custom domains`
3. `Set up a domain`
4. 输入 `appleid.mixtools.cc`
5. `Continue`
6. 因为 `mixtools.cc` 已在当前 Cloudflare 账户，确认 DNS 记录后会自动创建：

```text
CNAME appleid.mixtools.cc -> appleshare-hub.pages.dev
```

当前状态：自定义域名已在 Pages 项目里创建，但处于 `pending`，`verification_data` 显示 `CNAME record not set`。wrangler 登录令牌只有 `zone:read`，无法通过 API 补 DNS，需要到 Dashboard 完成：

1. 打开 [mixtools.cc DNS](https://dash.cloudflare.com/e23a0b1aabd2176a708ca168b943fffd/mixtools.cc/dns/records)
2. 添加一条 CNAME 记录：
   - 名称：`appleid`
   - 目标：`appleshare-hub.pages.dev`
   - 代理状态：DNS only（先不点亮橙云）
3. 回到 [Pages Custom domains](https://dash.cloudflare.com/e23a0b1aabd2176a708ca168b943fffd/pages/view/appleshare-hub/domains)，等状态从 `pending` 变为 `active`

或者新建一个带 `Zone > DNS > Edit` 权限的 API Token，之后可以执行 API 方式添加。

官方文档：

<https://developers.cloudflare.com/pages/configuration/custom-domains/>

## 8. API 方式添加（可选）

```bash
curl -X POST \
  "https://api.cloudflare.com/client/v4/accounts/<ACCOUNT_ID>/pages/projects/appleshare-hub/domains" \
  -H "Authorization: Bearer <API_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"name":"appleid.mixtools.cc"}'
```

## 9. 验证

```bash
curl -I https://appleid.mixtools.cc/
curl https://appleid.mixtools.cc/api/accounts
```

`/api/accounts` 返回 JSON 即挂载成功。
渠道名应能看到中文（如 `渠道 A（sha.cx）`、`云码酷（appleid.uczyw.us）`）。Windows PowerShell 直接管道读接口会把 UTF-8 显示成 `?`，验证时先保存到文件再检查：

```powershell
curl.exe -sS https://appleid.mixtools.cc/api/accounts -o accounts.json
Get-Content accounts.json -Encoding UTF8
```

## 注意

- 如果 zone 里存在 CAA 记录，需要允许 Cloudflare 签发证书，否则自定义域名无法激活。
- 不要在 DNS 里手动建 CNAME 绕过 Pages 的 Custom domains 流程，否则可能出现 522。
- 生产环境必须配置 `REFRESH_TOKEN`，避免 `/api/refresh` 被公开调用。
