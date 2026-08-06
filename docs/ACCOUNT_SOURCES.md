# 账号渠道接入文档

## 1. 当前渠道

当前项目内置 `sha_cx` 渠道，用于抓取 sha.cx 分享页中的苹果账号。

| 配置 ID | 名称 | 来源 |
| --- | --- | --- |
| `sha_cx_01` | 渠道 A（sha.cx） | <https://d8p8e.sha.cx/51e8990f678655f7749dfa8c5598dfbd> |
| `sha_cx_02` | 渠道 B（sha.cx） | <https://7y6h5.sha.cx/23cfa3c22135050d45f82283f2ef6e7f> |

## 2. sha.cx 页面结构

sha.cx 分享页会把账号数组直接写在页面 JavaScript 变量中，形如：

```html
<script>
  var ad='[...]';
</script>
```

其中 `ad` 是一个 JSON 数组，示例：

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

## 3. 状态映射

`status` 字段由 sha.cx 上游返回，映射规则在 `internal/provider/shacx.go` 的 `mapShaCXStatus` 中：

| 上游 status | 本项目 status | 展示文案 |
| --- | --- | --- |
| `0` | `checking` | 检测中 |
| `1` | `available` | 可用 |
| `2` | `unavailable` | 异常 |
| `3` | `pending` | 待检测 |
| 其他 | `unknown` | 未知 |

## 4. 抓取实现

`internal/provider/shacx.go` 的流程：

1. 使用 HTTP GET 请求配置中的 `url`。
2. 检查 HTTP 状态码必须为 200。
3. 限制响应体读取大小为 2 MB，防止异常页面耗尽内存。
4. 用正则 `ad\s*=\s*'([^']*)'` 提取 `ad` 中的 JSON 字符串。
5. 反序列化为 `[]shaCXRawAccount`。
6. 转换为统一的 `model.Account`。

## 5. 新增渠道

假设新渠道是 JSON API，返回格式与 sha.cx 不同，例如：

```json
{
  "data": {
    "email": "a@example.com",
    "pass": "123456",
    "region": "美国",
    "state": "ok"
  }
}
```

新增 `internal/provider/customapi.go`：

```go
package provider

import (
    "context"
    "fmt"
    "net/http"
    "time"

    "github.com/Q1YZn/appleshare-hub/internal/model"
)

type customAPIProvider struct {
    id     string
    name   string
    url    string
    client *http.Client
}

func init() {
    Register("custom_api", newCustomAPIProvider)
}

func newCustomAPIProvider(cfg Config) (Provider, error) {
    return &customAPIProvider{
        id:     cfg.ID,
        name:   cfg.Name,
        url:    cfg.URL,
        client: &http.Client{Timeout: 15 * time.Second},
    }, nil
}

func (p *customAPIProvider) ID() string { return p.id }
func (p *customAPIProvider) Name() string { return p.name }

func (p *customAPIProvider) Fetch(ctx context.Context) ([]model.Account, error) {
    // 在这里实现自己的 API 请求与字段映射
    return nil, fmt.Errorf("not implemented")
}
```

然后在 `config.json` 中启用：

```json
{
  "id": "custom_01",
  "type": "custom_api",
  "name": "自定义 API 渠道",
  "url": "https://example.com/api/account",
  "enabled": true
}
```

## 6. 注意事项

- 上游页面结构可能变化，建议定期验证 `extractShaCXPayload` 的正则。
- 每个渠道建议设置 `request_timeout_seconds`，避免单渠道拖慢整个刷新。
- 上游返回的账号密码仅用于展示，不要写入日志。
- 如果某个渠道状态字段含义不同，在对应 Provider 中完成映射，不要污染公共模型。
