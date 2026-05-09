# Getting Started

UA3F is an HTTP(S) rewriting proxy for modifying request and response headers, body content, and URL routing behavior. In addition to HTTP rewrite, UA3F supports L3 rewrite and Desync: L3 rewrite can adjust network-layer characteristics such as TTL, IPID, TCP Timestamp, and TCP Initial Window, with optional TC eBPF acceleration; Desync can interfere with stream reassembly on some DPI devices through TCP segment reordering and low-TTL obfuscation injection.

## Installation

### Release packages

Download binaries, opkg packages, or apk packages from [GitHub Releases](https://github.com/SunBK201/UA3F/releases).

### Docker

Run UA3F as a SOCKS5 proxy:

```sh
docker run -p 1080:1080 sunbk201/ua3f -f FFF
```

### Build from source

```sh
git clone https://github.com/SunBK201/UA3F.git
cd UA3F/src
go build -o ua3f main.go
```

## First run

Start with defaults:

```sh
ua3f
```

By default, UA3F listens on `127.0.0.1:1080`, runs in `SOCKS5` mode, and uses the `GLOBAL` rewrite mode.

Start with a configuration file:

```sh
ua3f -c /path/to/config.yaml
```

Generate a template configuration:

```sh
ua3f -g
```

## Common deployment choices

| Scenario | Recommended setup |
| --- | --- |
| Local explicit proxy | `SOCKS5` or `HTTP` |
| OpenWrt transparent proxy | `TPROXY` or `REDIRECT` |
| Network-layer queue processing | `NFQUEUE` |
| Coexisting with Clash | Run UA3F as `SOCKS5`, then route HTTP/TCP traffic from Clash to UA3F |
| Rewriting HTTPS headers or bodies | Enable `mitm` for selected hostnames |
| Network-layer characteristic rewriting | Enable L3 rewrite and configure TTL, IPID, TCP Timestamp, or TCP Initial Window as needed |
| DPI stream reassembly interference | Enable Desync and configure TCP segment reordering or TCP obfuscation injection as needed |

## Next steps

- Read [Concepts](./concepts.md) for server modes, rewrite modes, and rules.
- Read [Configuration](./configuration.md) to write a YAML configuration.
- Read [HTTP Rewrite](/http-rewrite/rewrite-modes.md), [L3 Rewrite](/l3/overview.md), and [Desync](/desync/overview.md) for the full feature set.
- Read the [API reference](/api/) for runtime inspection and log access.
