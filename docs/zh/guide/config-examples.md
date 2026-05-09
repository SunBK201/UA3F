# 配置示例

本页提供常见场景的 YAML 配置模板。示例可以单独使用，也可以把需要的配置段合并到同一个 `config.yaml` 中。

## 最小 SOCKS5 代理

适合本机应用或上游代理接入。

```yaml
server-mode: SOCKS5
bind-address: 127.0.0.1
port: 1080
log-level: info

rewrite-mode: GLOBAL
user-agent: "FFF"
```

## HTTP 代理

适合显式 HTTP 代理场景。

```yaml
server-mode: HTTP
bind-address: 127.0.0.1
port: 8080

rewrite-mode: GLOBAL
user-agent: "FFF"
```

## RULE 模式重写

适合按域名、端口、Header 或 URL 条件执行不同动作。

```yaml
server-mode: SOCKS5
bind-address: 127.0.0.1
port: 1080

rewrite-mode: RULE

header-rewrite:
  - type: DEST-PORT
    match-value: "22"
    action: DIRECT

  - type: DOMAIN-SUFFIX
    match-value: "ua-check.stagoh.com"
    action: REPLACE
    rewrite-direction: REQUEST
    rewrite-header: "User-Agent"
    rewrite-value: "UA3F"

  - type: FINAL
    action: REPLACE
    rewrite-header: "User-Agent"
    rewrite-value: "FFF"
```

## 透明代理网关

适合 Linux/OpenWrt 网关透明接管流量。实际转发规则需要配合系统防火墙或 OpenWrt 脚本。

```yaml
server-mode: TPROXY
bind-address: 0.0.0.0
port: 1080
include-lan-routes: false

rewrite-mode: GLOBAL
user-agent: "FFF"
```

如果环境不能使用 TPROXY，可改用 `REDIRECT`：

```yaml
server-mode: REDIRECT
bind-address: 0.0.0.0
port: 1080

rewrite-mode: GLOBAL
user-agent: "FFF"
```

## L3 重写与 eBPF 加速

适合需要调整网络层特征的 Linux 网关。`bpf-offload` 要求 Linux 内核版本 `>= 5.15`。

```yaml
server-mode: TPROXY
bind-address: 0.0.0.0
port: 1080

rewrite-mode: GLOBAL
user-agent: "FFF"

l3-rewrite:
  ttl: true
  ipid: true
  tcpts: true
  tcpwin: true
  bpf-offload: true
```

## Desync

适合需要尝试 DPI 流重组干扰的场景。建议先只开启一个 Desync 功能，确认连通性后再组合使用。

```yaml
server-mode: TPROXY
bind-address: 0.0.0.0
port: 1080

rewrite-mode: GLOBAL
user-agent: "FFF"

desync:
  reorder: true
  reorder-bytes: 8
  reorder-packets: 1500
  inject: true
  inject-ttl: 3
  desync-ports: "80,443"
```

## API Server

适合需要运行时查看状态、读取日志或重载配置的部署。

```yaml
api-server: "127.0.0.1:9000"
api-server-secret: "change-me"

server-mode: SOCKS5
bind-address: 127.0.0.1
port: 1080

rewrite-mode: GLOBAL
user-agent: "FFF"
```

访问 API 时携带鉴权：

```sh
curl -H "Authorization: Bearer change-me" http://127.0.0.1:9000/version
```

## 完整 RULE 模板

这个模板覆盖 Header、Body 和 URL 重定向规则，适合作为复杂配置的起点。

```yaml
server-mode: SOCKS5
bind-address: 127.0.0.1
port: 1080
log-level: info

rewrite-mode: RULE
user-agent: "FFF"
user-agent-regex: ""
user-agent-partial-replace: false

header-rewrite:
  - type: HEADER-KEYWORD
    match-header: "User-Agent"
    match-value: "MicroMessenger Client"
    action: DIRECT

  - type: HEADER-REGEX
    match-header: "User-Agent"
    match-value: "(Apple|iPhone|iPad|Windows|Linux|Android)"
    action: REPLACE-REGEX
    rewrite-direction: REQUEST
    rewrite-header: "User-Agent"
    rewrite-regex: "(Apple|iPhone|iPad|Windows|Linux|Android)"
    rewrite-value: "FFF"

  - type: FINAL
    action: REPLACE
    rewrite-header: "User-Agent"
    rewrite-value: "FFF"

body-rewrite:
  - type: URL-REGEX
    match-value: "^http://ua-check.stagoh.com"
    action: REPLACE-REGEX
    rewrite-direction: RESPONSE
    rewrite-regex: "UA2F"
    rewrite-value: "UA3F"

url-redirect:
  - type: URL-REGEX
    match-value: "^http://example.com/"
    action: REDIRECT-307
    rewrite-regex: "^http://example.com/(.*)"
    rewrite-value: "https://example.com/$1"
```
