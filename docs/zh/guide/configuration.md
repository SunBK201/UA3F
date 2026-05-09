# 配置说明

UA3F 支持 YAML 配置文件、环境变量和命令行参数。配置合并优先级为：

```text
默认值 < YAML 配置文件 < 环境变量 < 命令行参数
```

当同一配置项在多处指定时，高优先级的值会覆盖低优先级的值。推荐把长期配置写入 YAML，把部署环境差异放到环境变量，把临时调试项放到命令行参数。

## 使用方式

通过配置文件启动：

```sh
ua3f -c /path/to/config.yaml
```

生成模板配置：

```sh
ua3f -g
```

使用环境变量覆盖：

```sh
UA3F_SERVER_MODE=SOCKS5 UA3F_PORT=1080 ua3f
```

使用命令行参数覆盖：

```sh
ua3f --mode SOCKS5 --bind 127.0.0.1 --port 1080 --ua FFF
```

## 基础服务

基础服务配置控制 UA3F 的运行模式、监听地址、端口和日志级别。

```yaml
server-mode: SOCKS5
bind-address: 127.0.0.1
port: 1080
log-level: info
include-lan-routes: false
```

| 功能 | YAML | 命令行参数 | 环境变量 | 默认值 |
| --- | --- | --- | --- | --- |
| 配置文件路径 | - | `-c`, `--config` | - | - |
| 服务模式 | `server-mode` | `-m`, `--mode` | `UA3F_SERVER_MODE` | `SOCKS5` |
| 监听地址 | `bind-address` | `-b`, `--bind` | `UA3F_BIND_ADDRESS` | `127.0.0.1` |
| 监听端口 | `port` | `-p`, `--port` | `UA3F_PORT` | `1080` |
| 日志等级 | `log-level` | `-l`, `--log-level` | `UA3F_LOG_LEVEL` | `info` |
| 包含 LAN 路由 | `include-lan-routes` | `--include-lan-routes` | `UA3F_INCLUDE_LAN_ROUTES` | `false` |
| 显示版本 | - | `-v`, `--version` | - | - |
| 生成模板配置 | - | `-g`, `--generate-config` | - | - |

`server-mode` 可选值为 `HTTP`、`SOCKS5`、`TPROXY`、`REDIRECT`、`NFQUEUE`。

## HTTP 重写

HTTP 重写配置控制全局 User-Agent 替换和重写模式。

```yaml
rewrite-mode: GLOBAL
user-agent: "FFF"
user-agent-regex: ""
user-agent-partial-replace: false
```

| 功能 | YAML | 命令行参数 | 环境变量 | 默认值 |
| --- | --- | --- | --- | --- |
| 重写模式 | `rewrite-mode` | `-x`, `--rewrite-mode` | `UA3F_REWRITE_MODE` | `GLOBAL` |
| User-Agent 目标值 | `user-agent` | `-f`, `--ua` | `UA3F_PAYLOAD_UA` | `FFF` |
| User-Agent 匹配正则 | `user-agent-regex` | `-r`, `--ua-regex` | `UA3F_UA_REGEX` | 空 |
| 正则部分替换 | `user-agent-partial-replace` | `-s`, `--partial` | `UA3F_PARTIAL_REPLACE` | `false` |

`rewrite-mode` 可选值为 `GLOBAL`、`DIRECT`、`RULE`。规则匹配和动作详见 [HTTP 重写](/zh/http-rewrite/rewrite-modes.md)。

## 重写规则

规则配置仅在 `rewrite-mode: RULE` 时使用。YAML 适合长期维护；命令行参数和环境变量接收 JSON 字符串，适合自动化注入。

```yaml
rewrite-mode: RULE

header-rewrite:
  - type: DOMAIN-SUFFIX
    match-value: "example.com"
    action: REPLACE
    rewrite-direction: REQUEST
    rewrite-header: "User-Agent"
    rewrite-value: "FFF"

body-rewrite:
  - type: URL-REGEX
    match-value: "^http://example.com"
    action: REPLACE-REGEX
    rewrite-direction: RESPONSE
    rewrite-regex: "OldString"
    rewrite-value: "NewString"

url-redirect:
  - type: URL-REGEX
    match-value: "^http://example.com/old"
    action: REDIRECT-302
    rewrite-regex: "^http://example.com/old(.*)"
    rewrite-value: "http://example.com/new$1"
```

| 功能 | YAML | 命令行参数 | 环境变量 |
| --- | --- | --- | --- |
| Header 重写规则 | `header-rewrite` | `--header-rewrite` | `UA3F_HEADER_REWRITE` |
| Body 重写规则 | `body-rewrite` | `--body-rewrite` | `UA3F_BODY_REWRITE` |
| URL 重定向规则 | `url-redirect` | `--url-redirect` | `UA3F_URL_REDIRECT` |

命令行 JSON 示例：

```sh
ua3f --rewrite-mode RULE \
  --header-rewrite '[{"type":"FINAL","action":"REPLACE","rewrite_header":"User-Agent","rewrite_value":"FFF"}]'
```

规则字段、匹配类型和动作详见 [匹配规则](/zh/http-rewrite/match-rules.md) 与 [重写动作](/zh/http-rewrite/rewrite-actions.md)。

## API Server

API Server 用于运行时状态查询、日志读取和配置重载。

```yaml
api-server: "127.0.0.1:9000"
api-server-secret: "change-me"
```

| 功能 | YAML | 命令行参数 | 环境变量 | 默认值 |
| --- | --- | --- | --- | --- |
| API 监听地址 | `api-server` | `--api-server` | `UA3F_API_SERVER` | 空，禁用 |
| API 鉴权密钥 | `api-server-secret` | `--api-server-secret` | `UA3F_API_SERVER_SECRET` | 空，不鉴权 |

设置 `api-server-secret` 后，请求需要携带 Bearer Token：

```sh
curl -H "Authorization: Bearer change-me" http://127.0.0.1:9000/version
```

## L3 重写

L3 重写用于调整 TTL、IPID、TCP Timestamp、TCP 初始窗口等网络层特征。推荐使用 `l3-rewrite` 配置块；顶层 `ttl`、`ipid`、`tcp_timestamp`、`tcp_initial_window` 仍可用，并会与 `l3-rewrite` 合并。

```yaml
l3-rewrite:
  ttl: true
  ipid: true
  tcpts: true
  tcpwin: true
  bpf-offload: true
```

| 功能 | 推荐 YAML | 兼容 YAML | 命令行参数 | 环境变量 | 默认值 |
| --- | --- | --- | --- | --- | --- |
| TTL 重写 | `l3-rewrite.ttl` | `ttl` | `--ttl`, `--l3-rewrite-ttl` | `UA3F_L3_REWRITE_TTL`, `UA3F_TTL` | `false` |
| IPID 重写 | `l3-rewrite.ipid` | `ipid` | `--ipid`, `--l3-rewrite-ipid` | `UA3F_L3_REWRITE_IPID`, `UA3F_IPID` | `false` |
| 删除 TCP Timestamp | `l3-rewrite.tcpts` | `tcp_timestamp` | `--tcpts`, `--l3-rewrite-tcpts` | `UA3F_L3_REWRITE_TCPTS`, `UA3F_TCPTS` | `false` |
| 修改 TCP 初始窗口 | `l3-rewrite.tcpwin` | `tcp_initial_window` | `--tcpwin`, `--l3-rewrite-tcpwin` | `UA3F_L3_REWRITE_TCPWIN`, `UA3F_TCP_INIT_WINDOW` | `false` |
| L3 eBPF 加速 | `l3-rewrite.bpf-offload` | - | `--l3-rewrite-bpf-offload` | `UA3F_L3_REWRITE_BPF_OFFLOAD` | `false` |

L3 eBPF 加速要求 Linux 内核 `>= 5.15`。详见 [L3 重写](/zh/l3/overview.md) 与 [eBPF 加速](/zh/ebpf/l3-rewrite.md)。

## Desync

Desync 用于 TCP 分片乱序发射和 TCP 混淆注入。

```yaml
desync:
  reorder: false
  reorder-bytes: 8
  reorder-packets: 1500
  inject: false
  inject-ttl: 3
  desync-ports: ""
```

| 功能 | YAML | 命令行参数 | 环境变量 | 默认值 |
| --- | --- | --- | --- | --- |
| TCP 分片乱序发射 | `desync.reorder` | `--desync-reorder` | `UA3F_DESYNC_REORDER` | `false` |
| 乱序分片字节数 | `desync.reorder-bytes` | `--desync-reorder-bytes` | `UA3F_DESYNC_REORDER_BYTES` | `8` |
| 乱序包大小 | `desync.reorder-packets` | `--desync-reorder-packets` | `UA3F_DESYNC_REORDER_PACKETS` | `1500` |
| TCP 混淆注入 | `desync.inject` | `--desync-inject` | `UA3F_DESYNC_INJECT` | `false` |
| 注入包 TTL | `desync.inject-ttl` | `--desync-inject-ttl` | `UA3F_DESYNC_INJECT_TTL` | `3` |
| 生效端口 | `desync.desync-ports` | `--desync-ports` | `UA3F_DESYNC_PORTS` | 空 |

详见 [Desync](/zh/desync/overview.md)。

## MitM

MitM 用于对指定 HTTPS 主机名进行解密和 HTTP 重写。只应对明确需要重写的可信目标启用，客户端需要信任 UA3F 使用的 CA。

```yaml
mitm:
  enabled: false
  hostname: "*.httpbin.com, example.com:8000"
  ca-p12: ""
  ca-p12-base64: ""
  ca-passphrase: ""
  insecure-skip-verify: false
```

| 功能 | YAML | 命令行参数 | 环境变量 | 默认值 |
| --- | --- | --- | --- | --- |
| 启用 MitM | `mitm.enabled` | `--mitm` | `UA3F_MITM_ENABLED` | `false` |
| 目标主机名 | `mitm.hostname` | `--mitm-hostname` | `UA3F_MITM_HOSTNAME` | 空 |
| PKCS#12 CA 文件 | `mitm.ca-p12` | `--mitm-ca-p12` | `UA3F_MITM_CA_P12` | 空 |
| Base64 PKCS#12 CA | `mitm.ca-p12-base64` | `--mitm-ca-p12-base64` | `UA3F_MITM_CA_P12_BASE64` | 空 |
| CA 密码 | `mitm.ca-passphrase` | `--mitm-ca-passphrase` | `UA3F_MITM_CA_PASSPHRASE` | 空 |
| 跳过上游证书校验 | `mitm.insecure-skip-verify` | `--mitm-insecure-skip-verify` | `UA3F_MITM_INSECURE_SKIP_VERIFY` | `false` |

详见 [HTTPS MitM](/zh/http-rewrite/mitm.md)。

## 通用 BPF 卸载

UA3F 还保留了顶层 BPF 卸载开关：

```yaml
bpf-offload: false
```

| 功能 | YAML | 命令行参数 | 环境变量 | 默认值 |
| --- | --- | --- | --- | --- |
| 通用 BPF 卸载 | `bpf-offload` | `--bpf-offload` | `UA3F_BPF_OFFLOAD` | `false` |

L3 重写场景优先使用 `l3-rewrite.bpf-offload`。
