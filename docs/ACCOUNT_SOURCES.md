# 账号渠道接入文档

## 1. 当前渠道

项目通过 `internal/provider` 下的 `Provider` 接口统一接入多渠道，当前默认启用以下来源：

| 配置 ID | 类型 | 名称 | 来源 |
| --- | --- | --- | --- |
| `sha_cx_01` | `sha_cx` | 渠道 A（sha.cx） | <https://d8p8e.sha.cx/51e8990f678655f7749dfa8c5598dfbd> |
| `sha_cx_02` | `sha_cx` | 渠道 B（sha.cx） | <https://7y6h5.sha.cx/23cfa3c22135050d45f82283f2ef6e7f> |
| `fanqiangnan_01` | `fanqiangnan` | 翻墙男（fanqiangnan） | <https://fanqiangnan.com/data_sync.php> |
| `idfree_01` | `idfree` | 小优 ID（idfree） | <https://idfree.top/> |
| `appleid_api_01` | `appleid_api` | 云码酷（appleid.uczyw.us） | <https://appleid.uczyw.us/api/accounts> |
| `iosapp_text_01` | `iosapp_text` | 免费文本源（iosapp.icu） | <https://free.iosapp.icu/go-rod/1.txt> 等 1-3 |

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

需要三步会话流程：

```text
1. GET  https://idfree.top/                    获取 Cookie 和 <meta name="x-token">
2. POST https://idfree.top/api/session_verify.php
3. GET  https://idfree.top/api/accounts.php    携带 Cookie、X-Token、浏览器头
```

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
- 缺 `Accept`、`Sec-Fetch-*`、`X-Requested-With` 等浏览器头会返回 `INVALID_BROWSER`。
- `status: true` 映射为 `available`，`last_check` 作为更新时间。
- 完整请求头在 `internal/provider/idfree.go` 的 `baseHeaders` 中维护。

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
