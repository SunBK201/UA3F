# Configuration Examples

This page provides YAML templates for common deployment scenarios. Use each example directly or merge the relevant sections into one `config.yaml`.

## Minimal SOCKS5 proxy

Use this for local applications or upstream proxy integration.

```yaml
server-mode: SOCKS5
bind-address: 127.0.0.1
port: 1080
log-level: info

rewrite-mode: GLOBAL
user-agent: "FFF"
```

## HTTP proxy

Use this for explicit HTTP proxy deployments.

```yaml
server-mode: HTTP
bind-address: 127.0.0.1
port: 8080

rewrite-mode: GLOBAL
user-agent: "FFF"
```

## RULE mode rewrite

Use this when different domains, ports, headers, or URLs need different actions.

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

## Transparent proxy gateway

Use this for Linux/OpenWrt gateway deployments that transparently handle traffic. The actual forwarding rules must be paired with system firewall or OpenWrt scripts.

```yaml
server-mode: TPROXY
bind-address: 0.0.0.0
port: 1080
include-lan-routes: false

rewrite-mode: GLOBAL
user-agent: "FFF"
```

If TPROXY is not available, use `REDIRECT` instead:

```yaml
server-mode: REDIRECT
bind-address: 0.0.0.0
port: 1080

rewrite-mode: GLOBAL
user-agent: "FFF"
```

## L3 rewrite with eBPF acceleration

Use this on Linux gateways that need network-layer characteristic rewriting. `bpf-offload` requires Linux kernel `>= 5.15`.

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

Use this when testing DPI stream reassembly interference. Enable one Desync feature first, check connectivity, then combine them if needed.

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

## API server

Use this when a deployment needs runtime status, log access, or configuration reload.

```yaml
api-server: "127.0.0.1:9000"
api-server-secret: "change-me"

server-mode: SOCKS5
bind-address: 127.0.0.1
port: 1080

rewrite-mode: GLOBAL
user-agent: "FFF"
```

Access the API with authentication:

```sh
curl -H "Authorization: Bearer change-me" http://127.0.0.1:9000/version
```

## Complete RULE template

This template covers header rewrite, body rewrite, and URL redirect rules. Use it as a starting point for larger configurations.

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
