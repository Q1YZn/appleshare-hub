# AppleShare Hub

AppleShare Hub 是一个使用 Go + Gin 构建的苹果账号分发状态服务。它从多个上游账号渠道抓取账号与检测状态，以统一的 API 和网页形式展示给用户，并明确提示“只在 App Store 登录、iOS 26 从设置退出”等安全事项。

## 项目特点

- Go + Gin 轻量服务，前端使用 Vite + Vue 3 构建，单二进制即可运行。
- 通过 `Provider` 接口抽象账号渠道，后续接入新的渠道时无需改动页面和 API。
- 短时间缓存上游结果，避免每个用户访问都打上游接口。
- 网页实时展示账号状态、可用数量、渠道健康度、状态说明。
- 支持按国家/地区、渠道和是否附带 Shadowrocket 筛选，渠道状态以字母编号紧凑展示，渠道 A 固定在最前。
- 账号列表默认每页展示 8 条，支持上一页、下一页和页码跳转。
- 内置 iOS 26 登录/退出教程、Apple 官方示意图和 idfree.top 详细图文教程。
- 已接入 sha.cx、翻墙男、小优 ID、云码酷、free.iosapp.icu、91unicorn 知识库等来源，后续可通过 `Provider` 继续扩展。
- 附带架构文档、渠道接入文档和使用教程。

## 重要安全提示

本项目仅用于学习与技术演示。共享账号存在被上游渠道停用或检测异常的风险：

- 只允许在 **App Store** 内登录，不要在设置、iCloud、App 与网站中登录。
- 不要修改共享账号的密码、邮箱、安全信息，不要开启双重认证。
- iOS 26 及之后版本，媒体与购买项目的退出入口在 **设置 > 你的姓名/Apple 账户 > 媒体与购买项目 > 退出登录**。
- 使用后请立即退出账号，避免设备被锁机。

## 快速开始

环境要求：Node.js 18+、npm、Go 1.24 或更高版本。

```bash
git clone https://github.com/Q1YZn/appleshare-hub.git
cd appleshare-hub
npm install
npm run build
go mod tidy
go run .
```

启动后访问：

- 网页：<http://localhost:8080>
- 账号 API：<http://localhost:8080/api/accounts>
- 健康检查：<http://localhost:8080/healthz>

默认端口为 `8080`。配置位于 `config.json`，示例见 `config.example.json`：

```json
{
  "server": {
    "port": 8080,
    "cache_ttl_seconds": 300,
    "request_timeout_seconds": 15
  },
  "providers": [
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
    },
    {
      "id": "fanqiangnan_01",
      "type": "fanqiangnan",
      "name": "翻墙男（fanqiangnan）",
      "url": "https://fanqiangnan.com/data_sync.php",
      "enabled": true
    },
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
    },
    {
      "id": "appleid_api_01",
      "type": "appleid_api",
      "name": "云码酷（appleid.uczyw.us）",
      "url": "https://appleid.uczyw.us/api/accounts",
      "enabled": true
    },
    {
      "id": "iosapp_text_01",
      "type": "iosapp_text",
      "name": "免费文本源（iosapp.icu，低优先级）",
      "enabled": true,
      "options": {
        "urls": [
          "https://free.iosapp.icu/go-rod/1.txt",
          "https://free.iosapp.icu/go-rod/2.txt",
          "https://free.iosapp.icu/go-rod/3.txt"
        ]
      }
    },
    {
      "id": "unicorn_knowledge_01",
      "type": "unicorn_knowledge",
      "name": "独角兽知识库（91unicorn）",
      "url": "https://91unicorn.cloud/api/v1/user/knowledge/fetch?id=34&language=zh-CN",
      "enabled": true,
      "options": {
        "token": ""
      }
    }
  ]
}
```

91unicorn 需要登录态：登录后从浏览器 localStorage 取 token，填入 `options.token`，或设置环境变量 `UNICORN_TOKEN`，程序会自动携带 `Authorization: Bearer <token>`。没有 token 时该渠道会标记为错误，不影响其他渠道。

## API

### `GET /api/accounts`

返回账号列表、渠道状态、状态说明和页面安全提示：

```json
{
  "code": 200,
  "message": "ok",
  "generated_at": "2026-08-07T01:00:00+08:00",
  "cache_ttl_seconds": 300,
  "accounts": [
    {
      "id": "sha_cx_01:trevorthompson2227@outlook.com",
      "channel": "sha_cx_01",
      "channel_name": "渠道 A（sha.cx）",
      "country": "美国",
      "username": "trevorthompson2227@outlook.com",
      "password": "y5#e76CT",
      "status": "available",
      "status_message": "检测正常，可登录 App Store",
      "status_label": "可用",
      "raw_status": 1,
      "shadowrocket": true,
      "updated_at": "2026-08-07 00:50:47",
      "source_url": "https://d8p8e.sha.cx/..."
    }
  ],
  "channels": [
    {
      "id": "sha_cx_01",
      "name": "渠道 A（sha.cx）",
      "order": 0,
      "status": "ok",
      "account_count": 1,
      "shadowrocket": true,
      "updated_at": "2026-08-07T01:00:00+08:00"
    }
  ],
  "warnings": [],
  "status_legend": [],
  "available_count": 1,
  "unavailable_count": 0,
  "pending_count": 0,
  "total_count": 1
}
```

### `GET /healthz`

```json
{ "status": "ok" }
```

## 目录结构

```text
.
├── main.go                       # 启动入口、配置加载、服务组装
├── config.json                   # 当前渠道配置
├── config.example.json           # 配置示例
├── internal/
│   ├── config/                   # 配置文件解析
│   ├── model/                    # 账号、渠道、快照等数据模型
│   ├── provider/                 # 渠道抽象与 sha.cx / fanqiangnan / idfree / appleid_api / iosapp_text / unicorn_knowledge 实现
│   ├── service/                  # 缓存、聚合、状态计算
│   └── httpapi/                  # Gin 路由与 API
├── frontend/                     # Vite + Vue 3 前端源码
│   ├── index.html
│   ├── vite.config.js
│   ├── public/assets/guide/      # Apple 官方与 idfree 教程图片
│   └── src/
│       ├── App.vue
│       ├── components/           # 账号卡片、筛选、分页、教程等组件
│       ├── composables/          # 快照获取与自动刷新逻辑
│       └── utils/                # 渠道字母、时间等格式化工具
├── web/                          # Vite 构建产物，由 //go:embed 内嵌
│   ├── index.html
│   └── assets/
│       ├── index-*.js
│       ├── index-*.css
│       └── guide/                # 教程图片
└── docs/
    ├── ARCHITECTURE.md           # 架构设计
    ├── ACCOUNT_SOURCES.md        # 账号渠道接入
    └── IOS26_LOGOUT_GUIDE.md     # iOS 26 登录/退出教程
```

## 文档

- [架构设计](docs/ARCHITECTURE.md)
- [账号渠道接入](docs/ACCOUNT_SOURCES.md)
- [渠道来源调研](docs/ACCOUNT_SOURCES_RESEARCH.md)
- [Cloudflare Pages + R2 部署](docs/CLOUDFLARE_R2.md)
- [费用与调用优化](docs/COST_OPTIMIZATION.md)
- [iOS 26 登录/退出教程](docs/IOS26_LOGOUT_GUIDE.md)

## 免责声明

本项目仅为技术演示，不对账号可用性、数据来源或使用后果负责。请勿将共享账号用于非法用途，请遵守 Apple 服务条款与当地法律。
