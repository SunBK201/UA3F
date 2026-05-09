# Configuration

UA3F supports YAML configuration files, environment variables, and command-line flags. Configuration is merged with this priority:

```text
defaults < YAML file < environment variables < command-line flags
```

When the same option is set in multiple places, the higher-priority source wins. A practical setup is to keep stable settings in YAML, deployment-specific settings in environment variables, and temporary debugging changes in command-line flags.

## Usage

Start with a configuration file:

```sh
ua3f -c /path/to/config.yaml
```

Generate a template configuration:

```sh
ua3f -g
```

Override with environment variables:

```sh
UA3F_SERVER_MODE=SOCKS5 UA3F_PORT=1080 ua3f
```

Override with command-line flags:

```sh
ua3f --mode SOCKS5 --bind 127.0.0.1 --port 1080 --ua FFF
```

## Basic service

Basic service options control UA3F's server mode, listen address, port, and log level.

```yaml
server-mode: SOCKS5
bind-address: 127.0.0.1
port: 1080
log-level: info
include-lan-routes: false
```

| Feature | YAML | CLI flag | Environment variable | Default |
| --- | --- | --- | --- | --- |
| Config file path | - | `-c`, `--config` | - | - |
| Server mode | `server-mode` | `-m`, `--mode` | `UA3F_SERVER_MODE` | `SOCKS5` |
| Listen address | `bind-address` | `-b`, `--bind` | `UA3F_BIND_ADDRESS` | `127.0.0.1` |
| Listen port | `port` | `-p`, `--port` | `UA3F_PORT` | `1080` |
| Log level | `log-level` | `-l`, `--log-level` | `UA3F_LOG_LEVEL` | `info` |
| Include LAN routes | `include-lan-routes` | `--include-lan-routes` | `UA3F_INCLUDE_LAN_ROUTES` | `false` |
| Show version | - | `-v`, `--version` | - | - |
| Generate template config | - | `-g`, `--generate-config` | - | - |

`server-mode` accepts `HTTP`, `SOCKS5`, `TPROXY`, `REDIRECT`, and `NFQUEUE`.

## HTTP rewrite

HTTP rewrite options control global User-Agent replacement and rewrite mode.

```yaml
rewrite-mode: GLOBAL
user-agent: "FFF"
user-agent-regex: ""
user-agent-partial-replace: false
```

| Feature | YAML | CLI flag | Environment variable | Default |
| --- | --- | --- | --- | --- |
| Rewrite mode | `rewrite-mode` | `-x`, `--rewrite-mode` | `UA3F_REWRITE_MODE` | `GLOBAL` |
| User-Agent value | `user-agent` | `-f`, `--ua` | `UA3F_PAYLOAD_UA` | `FFF` |
| User-Agent regex | `user-agent-regex` | `-r`, `--ua-regex` | `UA3F_UA_REGEX` | empty |
| Partial regex replacement | `user-agent-partial-replace` | `-s`, `--partial` | `UA3F_PARTIAL_REPLACE` | `false` |

`rewrite-mode` accepts `GLOBAL`, `DIRECT`, and `RULE`. See [HTTP Rewrite](/http-rewrite/rewrite-modes.md) for mode behavior.

## Rewrite rules

Rule configuration is used only when `rewrite-mode: RULE` is enabled. YAML is better for maintained configs; CLI flags and environment variables accept JSON strings for automation.

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

| Feature | YAML | CLI flag | Environment variable |
| --- | --- | --- | --- |
| Header rewrite rules | `header-rewrite` | `--header-rewrite` | `UA3F_HEADER_REWRITE` |
| Body rewrite rules | `body-rewrite` | `--body-rewrite` | `UA3F_BODY_REWRITE` |
| URL redirect rules | `url-redirect` | `--url-redirect` | `UA3F_URL_REDIRECT` |

CLI JSON example:

```sh
ua3f --rewrite-mode RULE \
  --header-rewrite '[{"type":"FINAL","action":"REPLACE","rewrite_header":"User-Agent","rewrite_value":"FFF"}]'
```

See [Match Rules](/http-rewrite/match-rules.md) and [Rewrite Actions](/http-rewrite/rewrite-actions.md) for rule fields, match types, and actions.

## API server

The API server provides runtime status, logs, and configuration reload endpoints.

```yaml
api-server: "127.0.0.1:9000"
api-server-secret: "change-me"
```

| Feature | YAML | CLI flag | Environment variable | Default |
| --- | --- | --- | --- | --- |
| API listen address | `api-server` | `--api-server` | `UA3F_API_SERVER` | empty, disabled |
| API auth secret | `api-server-secret` | `--api-server-secret` | `UA3F_API_SERVER_SECRET` | empty, no auth |

When `api-server-secret` is set, pass it as a Bearer token:

```sh
curl -H "Authorization: Bearer change-me" http://127.0.0.1:9000/version
```

## L3 rewrite

L3 rewrite adjusts network-layer characteristics such as TTL, IPID, TCP Timestamp, and TCP Initial Window. Prefer the `l3-rewrite` block. Top-level `ttl`, `ipid`, `tcp_timestamp`, and `tcp_initial_window` remain supported and are merged with `l3-rewrite`.

```yaml
l3-rewrite:
  ttl: true
  ipid: true
  tcpts: true
  tcpwin: true
  bpf-offload: true
```

| Feature | Preferred YAML | Compatible YAML | CLI flag | Environment variable | Default |
| --- | --- | --- | --- | --- | --- |
| TTL rewrite | `l3-rewrite.ttl` | `ttl` | `--ttl`, `--l3-rewrite-ttl` | `UA3F_L3_REWRITE_TTL`, `UA3F_TTL` | `false` |
| IPID rewrite | `l3-rewrite.ipid` | `ipid` | `--ipid`, `--l3-rewrite-ipid` | `UA3F_L3_REWRITE_IPID`, `UA3F_IPID` | `false` |
| Delete TCP Timestamp | `l3-rewrite.tcpts` | `tcp_timestamp` | `--tcpts`, `--l3-rewrite-tcpts` | `UA3F_L3_REWRITE_TCPTS`, `UA3F_TCPTS` | `false` |
| TCP Initial Window rewrite | `l3-rewrite.tcpwin` | `tcp_initial_window` | `--tcpwin`, `--l3-rewrite-tcpwin` | `UA3F_L3_REWRITE_TCPWIN`, `UA3F_TCP_INIT_WINDOW` | `false` |
| L3 eBPF acceleration | `l3-rewrite.bpf-offload` | - | `--l3-rewrite-bpf-offload` | `UA3F_L3_REWRITE_BPF_OFFLOAD` | `false` |

L3 eBPF acceleration requires Linux kernel `>= 5.15`. See [L3 Rewrite](/l3/overview.md) and [eBPF Acceleration](/ebpf/l3-rewrite.md).

## Desync

Desync enables TCP segment reordering and TCP obfuscation injection.

```yaml
desync:
  reorder: false
  reorder-bytes: 8
  reorder-packets: 1500
  inject: false
  inject-ttl: 3
  desync-ports: ""
```

| Feature | YAML | CLI flag | Environment variable | Default |
| --- | --- | --- | --- | --- |
| TCP segment reordering | `desync.reorder` | `--desync-reorder` | `UA3F_DESYNC_REORDER` | `false` |
| Reordered segment bytes | `desync.reorder-bytes` | `--desync-reorder-bytes` | `UA3F_DESYNC_REORDER_BYTES` | `8` |
| Reordered packet size | `desync.reorder-packets` | `--desync-reorder-packets` | `UA3F_DESYNC_REORDER_PACKETS` | `1500` |
| TCP obfuscation injection | `desync.inject` | `--desync-inject` | `UA3F_DESYNC_INJECT` | `false` |
| Injected packet TTL | `desync.inject-ttl` | `--desync-inject-ttl` | `UA3F_DESYNC_INJECT_TTL` | `3` |
| Effective ports | `desync.desync-ports` | `--desync-ports` | `UA3F_DESYNC_PORTS` | empty |

See [Desync](/desync/overview.md).

## MitM

MitM decrypts selected HTTPS hostnames so HTTP rewrite rules can operate on them. Enable it only for trusted targets that need HTTPS rewriting. Clients must trust the CA used by UA3F.

```yaml
mitm:
  enabled: false
  hostname: "*.httpbin.com, example.com:8000"
  ca-p12: ""
  ca-p12-base64: ""
  ca-passphrase: ""
  insecure-skip-verify: false
```

| Feature | YAML | CLI flag | Environment variable | Default |
| --- | --- | --- | --- | --- |
| Enable MitM | `mitm.enabled` | `--mitm` | `UA3F_MITM_ENABLED` | `false` |
| Target hostnames | `mitm.hostname` | `--mitm-hostname` | `UA3F_MITM_HOSTNAME` | empty |
| PKCS#12 CA file | `mitm.ca-p12` | `--mitm-ca-p12` | `UA3F_MITM_CA_P12` | empty |
| Base64 PKCS#12 CA | `mitm.ca-p12-base64` | `--mitm-ca-p12-base64` | `UA3F_MITM_CA_P12_BASE64` | empty |
| CA passphrase | `mitm.ca-passphrase` | `--mitm-ca-passphrase` | `UA3F_MITM_CA_PASSPHRASE` | empty |
| Skip upstream certificate verification | `mitm.insecure-skip-verify` | `--mitm-insecure-skip-verify` | `UA3F_MITM_INSECURE_SKIP_VERIFY` | `false` |

See [HTTPS MitM](/http-rewrite/mitm.md).

## Generic BPF offload

UA3F also keeps a top-level BPF offload switch:

```yaml
bpf-offload: false
```

| Feature | YAML | CLI flag | Environment variable | Default |
| --- | --- | --- | --- | --- |
| Generic BPF offload | `bpf-offload` | `--bpf-offload` | `UA3F_BPF_OFFLOAD` | `false` |

For L3 rewrite, prefer `l3-rewrite.bpf-offload`.
