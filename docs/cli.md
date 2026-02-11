# UA3F 配置文档

UA3F 支持三种配置方式，优先级从低到高为：

**默认值 < YAML 配置文件 < 环境变量 < 命令行参数**

当同一配置项在多处指定时，高优先级的值会覆盖低优先级的值。

---

## 目录

- [命令行参数](#命令行参数)
- [环境变量](#环境变量)
- [YAML 配置文件](#yaml-配置文件)
- [完整配置参考表](#完整配置参考表)
- [重写规则详解](#重写规则详解)
- [配置示例](#配置示例)

---

## 命令行参数

通过 `ua3f [flags]` 的方式传入参数。

### 基础参数

| 短参数 | 长参数 | 说明 | 默认值 |
| ------ | ------ | ---- | ------ |
| `-c` | `--config` | 指定 YAML 配置文件路径 | 无 |
| `-m` | `--mode` | 服务模式：`HTTP`、`SOCKS5`、`TPROXY`、`REDIRECT`、`NFQUEUE` | `SOCKS5` |
| `-b` | `--bind` | 绑定监听地址 | `127.0.0.1` |
| `-p` | `--port` | 监听端口号（1-65535） | `1080` |
| `-l` | `--log-level` | 日志等级：`debug`、`info`、`warn`、`error` | `info` |
| `-x` | `--rewrite-mode` | 重写策略：`GLOBAL`、`DIRECT`、`RULE` | `GLOBAL` |
| `-f` | `--ua` | 自定义 User-Agent | `FFF` |
| `-r` | `--ua-regex` | 自定义正则匹配 User-Agent（为空表示匹配所有） | 空 |
| `-s` | `--partial` | 启用正则部分替换（仅替换正则匹配到的部分） | `false` |
| `-v` | `--version` | 显示版本号 | - |
| `-g` | `--generate-config` | 在当前目录生成模板配置文件 `config.yaml` | - |

### 其他参数

| 长参数 | 说明 | 默认值 |
| ------ | ---- | ------ |
| `--ttl` | 启用 TTL 伪装 | `false` |
| `--ipid` | 启用 IP ID 伪装 | `false` |
| `--tcpts` | 删除 TCP Timestamp | `false` |
| `--tcpwin` | 设置 TCP Initial Window | `false` |

### Desync 参数

| 长参数 | 说明 | 默认值 |
| ------ | ---- | ------ |
| `--desync-reorder` | 启用 TCP 分片乱序发射 | `false` |
| `--desync-reorder-bytes` | 乱序分片字节数 | `8` |
| `--desync-reorder-packets` | 乱序分片包大小 | `1500` |
| `--desync-inject` | 启用 TCP 注入 | `false` |
| `--desync-inject-ttl` | 注入包的 TTL 值 | `3` |
| `--desync-ports` | 指定 Desync 生效的端口 | 空 |

### 重写规则参数

| 长参数 | 说明 | 默认值 |
| ------ | ---- | ------ |
| `--header-rewrite` | Header 重写规则（JSON 字符串），仅在 `RULE` 重写策略下生效 | 空 |
| `--body-rewrite` | Body 重写规则（JSON 字符串），仅在 `RULE` 重写策略下生效 | 空 |
| `--url-redirect` | URL 重定向规则（JSON 字符串），仅在 `RULE` 重写策略下生效 | 空 |

---

## 环境变量

所有环境变量均以 `UA3F_` 为前缀。

| 环境变量 | 对应配置项 | 说明 |
| -------- | ---------- | ---- |
| `UA3F_SERVER_MODE` | `server-mode` | 服务模式 |
| `UA3F_BIND_ADDRESS` | `bind-address` | 绑定地址 |
| `UA3F_PORT` | `port` | 端口号 |
| `UA3F_LOG_LEVEL` | `log-level` | 日志等级 |
| `UA3F_REWRITE_MODE` | `rewrite-mode` | 重写策略 |
| `UA3F_PAYLOAD_UA` | `user-agent` | 自定义 User-Agent |
| `UA3F_UA_REGEX` | `user-agent-regex` | User-Agent 正则 |
| `UA3F_PARTIAL_REPLACE` | `user-agent-partial-replace` | 正则部分替换 |
| `UA3F_TTL` | `ttl` | TTL 伪装 |
| `UA3F_IPID` | `ipid` | IP ID 伪装 |
| `UA3F_TCPTS` | `tcp_timestamp` | 删除 TCP Timestamp |
| `UA3F_TCP_INIT_WINDOW` | `tcp_initial_window` | TCP Initial Window |
| `UA3F_DESYNC_REORDER` | `desync.reorder` | Desync 乱序 |
| `UA3F_DESYNC_REORDER_BYTES` | `desync.reorder-bytes` | Desync 乱序字节数 |
| `UA3F_DESYNC_REORDER_PACKETS` | `desync.reorder-packets` | Desync 乱序包大小 |
| `UA3F_DESYNC_INJECT` | `desync.inject` | Desync 注入 |
| `UA3F_DESYNC_INJECT_TTL` | `desync.inject-ttl` | Desync 注入 TTL |
| `UA3F_DESYNC_PORTS` | `desync.desync-ports` | Desync 端口 |
| `UA3F_HEADER_REWRITE` | `header-rewrite-json` | Header 重写规则 JSON |
| `UA3F_BODY_REWRITE` | `body-rewrite-json` | Body 重写规则 JSON |
| `UA3F_URL_REDIRECT` | `url-redirect-json` | URL 重定向规则 JSON |

**使用示例：**

```sh
UA3F_PAYLOAD_UA=MyUA UA3F_SERVER_MODE=SOCKS5 ua3f
```

---

## YAML 配置文件

通过 `-c <path>` 指定配置文件路径，或使用 `-g` 生成模板配置文件。

### 完整 YAML 配置示例

```yaml
# 服务模式: HTTP, SOCKS5, TPROXY, REDIRECT, NFQUEUE
server-mode: SOCKS5

# 绑定地址
bind-address: 127.0.0.1

# 监听端口 (1-65535)
port: 1080

# 日志等级: debug, info, warn, error
log-level: info

# 重写策略: GLOBAL, DIRECT, RULE
rewrite-mode: GLOBAL

# User-Agent 相关
user-agent: "FFF"
user-agent-regex: ""
user-agent-partial-replace: false

# 网络伪装
ttl: false
ipid: false
tcp_timestamp: false
tcp_initial_window: false

# Desync 配置
desync:
  reorder: false
  reorder-bytes: 8
  reorder-packets: 1500
  inject: false
  inject-ttl: 3
  desync-ports: ""

# Header 重写规则 (仅 RULE 模式)
header-rewrite:
  - type: HEADER-KEYWORD
    match-header: "User-Agent"
    match-value: "MicroMessenger Client"
    action: DIRECT

  - type: HEADER-REGEX
    match-header: "User-Agent"
    match-value: "(Apple|iPhone|iPad|Windows|Linux|Android)"
    action: REPLACE-REGEX
    rewrite-header: "User-Agent"
    rewrite-regex: "(Apple|iPhone|iPad|Windows|Linux|Android)"
    rewrite-value: "FFF"

  - type: FINAL
    action: REPLACE
    rewrite-header: "User-Agent"
    rewrite-value: "FFF"

# Body 重写规则 (仅 RULE 模式)
body-rewrite:
  - type: URL-REGEX
    match-value: "^http://example.com"
    action: REPLACE-REGEX
    rewrite-direction: RESPONSE
    rewrite-regex: "OldString"
    rewrite-value: "NewString"

# URL 重定向规则 (仅 RULE 模式)
url-redirect:
  - type: URL-REGEX
    match-value: "^http://example.com/old-path"
    action: REDIRECT-302
    rewrite-regex: "^http://example.com/old-path(.*)"
    rewrite-value: "http://example.com/new-path$1"
```

---

## 完整配置参考表

下表汇总了所有配置项及其在三种配置方式中的对应关系：

| YAML 配置项 | 命令行参数 | 环境变量 | 类型 | 默认值 | 说明 |
| ----------- | --------- | -------- | ---- | ------ | ---- |
| `server-mode` | `-m` / `--mode` | `UA3F_SERVER_MODE` | string | `SOCKS5` | 服务模式 |
| `bind-address` | `-b` / `--bind` | `UA3F_BIND_ADDRESS` | string | `127.0.0.1` | 绑定地址 |
| `port` | `-p` / `--port` | `UA3F_PORT` | int | `1080` | 监听端口 |
| `log-level` | `-l` / `--log-level` | `UA3F_LOG_LEVEL` | string | `info` | 日志等级 |
| `rewrite-mode` | `-x` / `--rewrite-mode` | `UA3F_REWRITE_MODE` | string | `GLOBAL` | 重写策略 |
| `user-agent` | `-f` / `--ua` | `UA3F_PAYLOAD_UA` | string | `FFF` | User-Agent |
| `user-agent-regex` | `-r` / `--ua-regex` | `UA3F_UA_REGEX` | string | 空 | UA 正则 |
| `user-agent-partial-replace` | `-s` / `--partial` | `UA3F_PARTIAL_REPLACE` | bool | `false` | 正则部分替换 |
| `ttl` | `--ttl` | `UA3F_TTL` | bool | `false` | TTL 伪装 |
| `ipid` | `--ipid` | `UA3F_IPID` | bool | `false` | IP ID 伪装 |
| `tcp_timestamp` | `--tcpts` | `UA3F_TCPTS` | bool | `false` | TCP Timestamp |
| `tcp_initial_window` | `--tcpwin` | `UA3F_TCP_INIT_WINDOW` | bool | `false` | TCP Window |
| `desync.reorder` | `--desync-reorder` | `UA3F_DESYNC_REORDER` | bool | `false` | Desync 乱序 |
| `desync.reorder-bytes` | `--desync-reorder-bytes` | `UA3F_DESYNC_REORDER_BYTES` | uint | `8` | 乱序字节数 |
| `desync.reorder-packets` | `--desync-reorder-packets` | `UA3F_DESYNC_REORDER_PACKETS` | uint | `1500` | 乱序包大小 |
| `desync.inject` | `--desync-inject` | `UA3F_DESYNC_INJECT` | bool | `false` | Desync 注入 |
| `desync.inject-ttl` | `--desync-inject-ttl` | `UA3F_DESYNC_INJECT_TTL` | uint | `3` | 注入 TTL |
| `desync.desync-ports` | `--desync-ports` | `UA3F_DESYNC_PORTS` | string | 空 | Desync 端口 |
| `header-rewrite` | `--header-rewrite` (JSON) | `UA3F_HEADER_REWRITE` | []Rule | 空 | Header 重写规则 |
| `body-rewrite` | `--body-rewrite` (JSON) | `UA3F_BODY_REWRITE` | []Rule | 空 | Body 重写规则 |
| `url-redirect` | `--url-redirect` (JSON) | `UA3F_URL_REDIRECT` | []Rule | 空 | URL 重定向规则 |

---

## 重写规则详解

重写规则仅在 `rewrite-mode` 为 `RULE` 时生效。UA3F 支持三类重写规则：**Header 重写**、**Body 重写**、**URL 重定向**。

### 规则结构

每条规则包含以下字段：

| 字段 | YAML 键名 | JSON 键名 | 类型 | 必填 | 说明 |
| ---- | --------- | --------- | ---- | ---- | ---- |
| 启用 | `enabled` | `enabled` | bool | 否 | 是否启用该规则，默认启用 |
| 匹配类型 | `type` | `type` | string | **是** | 规则匹配类型 |
| 匹配 Header | `match-header` | `match_header` | string | 条件必填 | 匹配的 Header 名称 |
| 匹配值 | `match-value` | `match_value` | string | 条件必填 | 匹配的值 |
| 动作 | `action` | `action` | string | **是** | 匹配后执行的动作 |
| 重写 Header | `rewrite-header` | `rewrite_header` | string | 条件必填 | 要重写的 Header 名称 |
| 重写值 | `rewrite-value` | `rewrite_value` | string | 条件必填 | 重写的目标内容 |
| 重写方向 | `rewrite-direction` | `rewrite_direction` | string | 否 | `REQUEST` 或 `RESPONSE` |
| 重写正则 | `rewrite-regex` | `rewrite_regex` | string | 条件必填 | 用于 `REPLACE-REGEX` 动作 |
| 继续匹配 | `continue` | `continue` | bool | 否 | 匹配后是否继续匹配后续规则 |

### 匹配类型

| 类型 | 说明 | 必填字段 |
| ---- | ---- | -------- |
| `DOMAIN` | 精确匹配域名 | `match-value` |
| `DOMAIN-SUFFIX` | 匹配域名后缀 | `match-value` |
| `DOMAIN-KEYWORD` | 匹配域名关键字 | `match-value` |
| `IP-CIDR` | 匹配目标 IP 段 | `match-value` |
| `SRC-IP` | 匹配源 IP 地址 | `match-value` |
| `DEST-PORT` | 匹配目标端口 | `match-value` |
| `HEADER-KEYWORD` | 匹配 Header 中的关键字 | `match-header`、`match-value` |
| `HEADER-REGEX` | 正则匹配 Header 内容 | `match-header`、`match-value` |
| `URL-REGEX` | 正则匹配请求 URL | `match-value` |
| `FINAL` | 兜底规则，匹配所有请求 | 无 |

### 动作类型

#### Header/Body 重写动作

| 动作 | 说明 | 必填字段 |
| ---- | ---- | -------- |
| `DIRECT` | 直接放行，不进行重写 | 无 |
| `REPLACE` | 替换指定 Header 为指定内容 | `rewrite-header`、`rewrite-value` |
| `REPLACE-REGEX` | 将匹配正则的部分替换为指定内容 | `rewrite-header`、`rewrite-regex`、`rewrite-value` |
| `DELETE` | 删除指定 Header | `rewrite-header` |
| `ADD` | 添加指定 Header | `rewrite-header`、`rewrite-value` |
| `DROP` | 丢弃该请求 | 无 |

#### URL 重定向动作

| 动作 | 说明 | 必填字段 |
| ---- | ---- | -------- |
| `REDIRECT-302` | 返回 302 临时重定向 | `rewrite-regex`、`rewrite-value` |
| `REDIRECT-307` | 返回 307 临时重定向（保持请求方法） | `rewrite-regex`、`rewrite-value` |
| `REDIRECT-HEADER` | 修改请求 Header 进行内部重定向（客户端无感知） | `rewrite-regex`、`rewrite-value` |

---

## 配置示例

### 示例 1：最简启动

使用默认 SOCKS5 模式，将所有 User-Agent 改写为 `FFF`：

```sh
ua3f
```

### 示例 2：命令行参数指定

```sh
ua3f -m SOCKS5 -b 0.0.0.0 -p 1080 -f "FFF" -l debug
```

### 示例 3：环境变量方式

```sh
export UA3F_SERVER_MODE=TPROXY
export UA3F_PORT=9999
export UA3F_PAYLOAD_UA="FFF"
ua3f
```

### 示例 4：使用配置文件

```sh
# 生成模板配置文件
ua3f -g

# 使用指定配置文件启动
ua3f -c /path/to/config.yaml
```

### 示例 5：Docker 部署

```sh
docker run -p 1080:1080 sunbk201/ua3f -f FFF
```

### 示例 6：命令行传入 JSON 规则

```sh
ua3f -m SOCKS5 -x RULE --header-rewrite '[{"type":"FINAL","action":"REPLACE","rewrite_header":"User-Agent","rewrite_value":"FFF"}]'
```

### 示例 7：启用 Desync 对抗 DPI

```sh
ua3f --desync-reorder --desync-inject --desync-inject-ttl 3
```

### 示例 8：混合使用配置文件与命令行覆盖

命令行参数优先级高于配置文件，可以用来临时覆盖某些选项：

```sh
ua3f -c config.yaml -f "FFF" -l debug
```

### 示例 9：YAML 中的 RULE 模式完整规则

```yaml
rewrite-mode: RULE

header-rewrite:
  # 放行特定应用的 User-Agent
  - type: HEADER-KEYWORD
    match-header: "User-Agent"
    match-value: "MicroMessenger Client"
    action: DIRECT

  # 放行 SSH 端口
  - type: DEST-PORT
    match-value: "22"
    action: DIRECT

  # 将包含操作系统关键字的 User-Agent 进行正则替换
  - type: HEADER-REGEX
    match-header: "User-Agent"
    match-value: "(Apple|Windows|Linux|Android)"
    action: REPLACE-REGEX
    rewrite-header: "User-Agent"
    rewrite-regex: "(Apple|Windows|Linux|Android)"
    rewrite-value: "FFF"

  # 兜底规则：替换所有 User-Agent
  - type: FINAL
    action: REPLACE
    rewrite-header: "User-Agent"
    rewrite-value: "FFF"

body-rewrite:
  # 对特定 URL 的响应 Body 进行内容替换
  - type: URL-REGEX
    match-value: "^http://ua-check.stagoh.com"
    action: REPLACE-REGEX
    rewrite-direction: RESPONSE
    rewrite-regex: "UA2F"
    rewrite-value: "UA3F"

url-redirect:
  # 将旧路径 302 重定向到新路径
  - type: URL-REGEX
    match-value: "^http://example.com/old-path"
    action: REDIRECT-302
    rewrite-regex: "^http://example.com/old-path(.*)"
    rewrite-value: "http://example.com/new-path$1"
```

> **注意：** 规则按照列表顺序从上到下匹配，匹配到的第一条规则即生效。如果需要匹配后继续检查后续规则，可以设置 `continue: true`。`FINAL` 类型规则建议放在列表最后作为兜底规则。
