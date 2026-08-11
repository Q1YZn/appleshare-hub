# Cloudflare Agent Setup

本文档记录从 Cloudflare 官方 Agent 安装说明执行后的结果：

<https://developers.cloudflare.com/agent-setup/prompt.md>

## 已安装内容

### Skills

Cloudflare 官方 skills 已安装，当前副本位于：

```text
C:\Users\toufu\.codex\skills\cloudflare
C:\Users\toufu\.codex\skills\wrangler
C:\Users\toufu\.codex\skills\durable-objects
```

安装命令：

```bash
npx -y skills add cloudflare/skills --skill '*' --yes --global
```

### MCP Servers

已写入 `C:\Users\toufu\.codex\config.toml`：

```toml
[mcp_servers.cloudflare]
url = "https://mcp.cloudflare.com/mcp"

[mcp_servers.cloudflare-docs]
url = "https://docs.mcp.cloudflare.com/mcp"

[mcp_servers.cloudflare-bindings]
url = "https://bindings.mcp.cloudflare.com/mcp"

[mcp_servers.cloudflare-builds]
url = "https://builds.mcp.cloudflare.com/mcp"

[mcp_servers.cloudflare-observability]
url = "https://observability.mcp.cloudflare.com/mcp"
```

`cloudflare-docs` 无需登录；其余 Cloudflare MCP 需要 OAuth 登录。

## 登录

```bash
codex mcp login cloudflare
codex mcp login cloudflare-bindings
codex mcp login cloudflare-builds
codex mcp login cloudflare-observability
```

登录命令会打开浏览器授权页，需要手动点击 Allow。授权完成后重启 Codex，MCP 工具才会加载。

Wrangler 登录：

```bash
npx wrangler login
npx wrangler whoami
```

也可以使用 API Token 环境变量：

```powershell
$env:CLOUDFLARE_API_TOKEN = "你的 API Token"
```

## 验证

```bash
codex mcp list
npx wrangler whoami
```

`codex mcp list` 中 Cloudflare MCP 状态应为 Logged in。
