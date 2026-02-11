# UA3F API 文档

UA3F 提供 RESTful API 用于查询运行状态、获取配置信息、管理规则和实时查看日志。

API 服务器通过配置项 `api-server` 指定监听地址（如 `127.0.0.1:9000`），留空则不启用。

---

## 目录

- [认证](#认证)
- [API 端点](#api-端点)
  - [GET /version](#get-version)
  - [GET /config](#get-config)
  - [GET /rules](#get-rules)
  - [GET /rules/header](#get-rulesheader)
  - [GET /rules/body](#get-rulesbody)
  - [GET /rules/redirect](#get-rulesredirect)
  - [GET /logs](#get-logs)
  - [GET /restart](#get-restart)
- [pprof 调试端点](#pprof-调试端点)

---

## 认证

当配置项 `api-server-secret` 不为空时，所有 API 请求需要携带认证信息。支持以下两种方式：

### 方式一：Authorization Header

```
Authorization: Bearer <secret>
```

或直接传递 token：

```
Authorization: <secret>
```

### 方式二：URL Query 参数

```
GET /version?secret=<secret>
```

未认证或认证失败时返回：

```
HTTP/1.1 401 Unauthorized

{"error":"unauthorized"}
```

---

## API 端点

### GET /version

获取 UA3F 当前运行版本。

**请求示例：**

```bash
curl http://127.0.0.1:9000/version
```

**响应：**

```json
{
  "version": "0.7.0"
}
```

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `version` | string | UA3F 版本号 |

---

### GET /config

获取当前运行配置的完整内容。

**请求示例：**

```bash
curl http://127.0.0.1:9000/config
```

**响应：**

```json
{
  "ServerMode": "SOCKS5",
  "BindAddress": "127.0.0.1",
  "Port": 1080,
  "APIServer": "127.0.0.1:9000",
  "APIServerSecret": "",
  "LogLevel": "info",
  "RewriteMode": "RULE",
  "UserAgent": "FFF",
  "UserAgentRegex": "",
  "UserAgentPartialReplace": false,
  "TTL": false,
  "IPID": false,
  "TCPTimeStamp": false,
  "TCPInitialWindow": false,
  "MitM": {
    "Enabled": false,
    "Hostname": "",
    "CAP12": "",
    "CAP12Base64": "",
    "CAPassphrase": "",
    "InsecureSkipVerify": false
  },
  "Desync": {
    "Reorder": false,
    "ReorderBytes": 8,
    "ReorderPackets": 1500,
    "Inject": false,
    "InjectTTL": 3,
    "DesyncPorts": ""
  },
  "HeaderRules": [],
  "BodyRules": [],
  "URLRedirectRules": []
}
```

返回的 JSON 结构与 `Config` 结构体一一对应，包含所有运行时配置项。

---

### GET /rules

获取所有重写规则（Header / Body / Redirect）。

**请求示例：**

```bash
curl http://127.0.0.1:9000/rules
```

**响应：**

```json
{
  "header": [ ... ],
  "body": [ ... ],
  "redirect": [ ... ]
}
```

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `header` | array | Header 重写规则列表 |
| `body` | array | Body 重写规则列表 |
| `redirect` | array | URL 重定向规则列表 |

每条规则的结构如下：

```json
{
  "enabled": true,
  "type": "HEADER-KEYWORD",
  "match_header": "User-Agent",
  "match_value": "MicroMessenger",
  "action": "REPLACE",
  "rewrite_header": "User-Agent",
  "rewrite_value": "FFF",
  "rewrite_direction": "REQUEST",
  "rewrite_regex": "",
  "continue": false
}
```

**规则字段说明：**

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `enabled` | bool | 规则是否启用 |
| `type` | string | 匹配类型，可选值：`HEADER-KEYWORD`、`HEADER-REGEX`、`DEST-PORT`、`IP-CIDR`、`SRC-IP`、`DOMAIN-SUFFIX`、`DOMAIN-KEYWORD`、`DOMAIN`、`URL-REGEX`、`FINAL` |
| `match_header` | string | 匹配的 Header 名称（`HEADER-KEYWORD` / `HEADER-REGEX` 类型必填） |
| `match_value` | string | 匹配的值 |
| `action` | string | 动作类型，可选值：`DIRECT`、`REPLACE`、`REPLACE-REGEX`、`DELETE`、`ADD`、`DROP`、`REDIRECT-302`、`REDIRECT-307`、`REDIRECT-HEADER` |
| `rewrite_header` | string | 重写的 Header 名称 |
| `rewrite_value` | string | 重写的值 |
| `rewrite_direction` | string | 重写方向，可选值：`REQUEST`、`RESPONSE` |
| `rewrite_regex` | string | 正则表达式（`REPLACE-REGEX` 动作必填） |
| `continue` | bool | 匹配后是否继续匹配后续规则 |

---

### GET /rules/header

仅获取 Header 重写规则。

**请求示例：**

```bash
curl http://127.0.0.1:9000/rules/header
```

**响应：**

```json
[
  {
    "type": "HEADER-KEYWORD",
    "match_header": "User-Agent",
    "match_value": "MicroMessenger",
    "action": "REPLACE",
    "rewrite_header": "User-Agent",
    "rewrite_value": "FFF"
  }
]
```

---

### GET /rules/body

仅获取 Body 重写规则。

**请求示例：**

```bash
curl http://127.0.0.1:9000/rules/body
```

**响应：**

```json
[
  {
    "type": "DOMAIN-SUFFIX",
    "match_value": "example.com",
    "action": "REPLACE-REGEX",
    "rewrite_regex": "old_text",
    "rewrite_value": "new_text"
  }
]
```

---

### GET /rules/redirect

仅获取 URL 重定向规则。

**请求示例：**

```bash
curl http://127.0.0.1:9000/rules/redirect
```

**响应：**

```json
[
  {
    "type": "URL-REGEX",
    "match_value": "^http://example\\.com/old",
    "action": "REDIRECT-302",
    "rewrite_value": "http://example.com/new"
  }
]
```

---

### GET /logs

实时获取 UA3F 日志输出。支持 **WebSocket** 和 **HTTP 长连接（Chunked Transfer）** 两种模式。

#### WebSocket 模式

当请求包含 WebSocket Upgrade 头时，自动升级为 WebSocket 连接，日志以文本消息逐行推送。

**请求示例：**

```bash
wscat -c ws://127.0.0.1:9000/logs
```

```javascript
const ws = new WebSocket("ws://127.0.0.1:9000/logs");
ws.onmessage = (event) => {
  console.log(event.data);
};
```

#### HTTP Chunked 模式

普通 HTTP 请求将以 `text/plain` 流式传输日志，使用 Chunked Transfer Encoding 逐行推送。

**请求示例：**

```bash
curl -N http://127.0.0.1:9000/logs
```

**响应 Headers：**

```
Content-Type: text/plain; charset=utf-8
Cache-Control: no-cache
Connection: keep-alive
X-Content-Type-Options: nosniff
```

连接将保持打开状态，持续输出日志内容，直到客户端主动断开。

---

### GET /restart

重新加载配置文件并热重启所有服务组件。

**请求示例：**

```bash
curl http://127.0.0.1:9000/restart
```

**成功响应：**

```
HTTP/1.1 200 OK

success
```

**失败响应：**

```
HTTP/1.1 500 Internal Server Error

<error message>
```

---

## pprof 调试端点

API 服务器内置了 Go pprof 性能分析端点，可用于调试和性能优化。

| 端点 | 说明 |
| --- | --- |
| `GET /debug/pprof/` | pprof 索引页 |
| `GET /debug/pprof/cmdline` | 命令行参数 |
| `GET /debug/pprof/profile` | CPU Profile（默认 30 秒） |
| `GET /debug/pprof/symbol` | 符号查询 |
| `GET /debug/pprof/trace` | 执行 Trace |
| `GET /debug/pprof/goroutine` | Goroutine 状态 |
| `GET /debug/pprof/heap` | 堆内存分配 |
| `GET /debug/pprof/allocs` | 历史内存分配 |
| `GET /debug/pprof/threadcreate` | 线程创建 |
| `GET /debug/pprof/block` | 阻塞分析 |
| `GET /debug/pprof/mutex` | 互斥锁分析 |

**使用示例：**

```bash
# 获取 30 秒 CPU Profile
go tool pprof http://127.0.0.1:9000/debug/pprof/profile

# 查看堆内存
go tool pprof http://127.0.0.1:9000/debug/pprof/heap

# 查看 goroutine
curl http://127.0.0.1:9000/debug/pprof/goroutine?debug=1
```
