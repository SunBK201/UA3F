# UA3F

![Release](https://img.shields.io/github/v/release/SunBK201/UA3F?display_name=tag&label=UA3F&link=https%3A%2F%2Fgithub.com%2FSunBK201%2FUA3F%2Freleases%2Flatest)
[![CodeQL](https://github.com/SunBK201/UA3F/actions/workflows/github-code-scanning/codeql/badge.svg)](https://github.com/SunBK201/UA3F/actions/workflows/github-code-scanning/codeql)
[![License](https://img.shields.io/github/license/SunBK201/UA3F)](https://github.com/SunBK201/UA3F/blob/master/LICENSE)
![GitHub Downloads (all assets, all releases)](https://img.shields.io/github/downloads/SunBK201/UA3F/total?label=GitHub%20Downloads&link=https%3A%2F%2Fgithub.com%2FSunBK201%2FUA3F%2Freleases)
[![Telegram group](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fapi.swo.moe%2Fstats%2Ftelegram%2Fcrack_campus_network&query=count&color=2CA5E0&label=Telegram%20Group&logo=telegram&cacheSeconds=3600)](https://t.me/crack_campus_network)

<img align="right" src="https://sunbk201.oss-cn-beijing.aliyuncs.com/img/ua3f-210.png" alt="UA3F" width="300">

English | [简体中文](README.md)

UA3F is an HTTP(S) rewriting proxy that transparently rewrites HTTP(S) (e.g., User-Agent) as an HTTP, SOCKS5, TPROXY, REDIRECT, or NFQUEUE server.

- Supports bidirectional rewriting of HTTP(S) Headers and Body
- Supports HTTP(S) URL redirection: 302, 307, Header
- Supports HTTPS MitM traffic decryption and rewriting
- Multiple server modes: HTTP, SOCKS5, TPROXY, REDIRECT, NFQUEUE([UA2F](https://github.com/Zxilly/UA2F))
- Highly flexible rewriting rule system with multiple rule types and rewriting strategies
- Real-time statistics dashboard with traffic modification monitoring and analysis
- Multiple deployment options: opkg installation, compilation, and Docker deployment
- Compatible with Clash Fake-IP & Redir-Host modes for coexistence
- Supports L3 rewriting: TTL, IPID, TCP Timestamp, TCP Window
- Supports Desync fragment reordering and obfuscation to evade Deep Packet Inspection (DPI)
- Supports eBPF offloading for accelerated forwarding performance

## Installation

Multiple installation methods are available:

- Binary Files and Packages

  Pre-compiled binaries and opkg/apk packages for common architectures are available on the [Release](https://github.com/SunBK201/UA3F/releases) page. Download the appropriate package for your platform and install it directly.

- Docker Deployment:

  Run as a SOCKS5 proxy:

  ```sh
  docker run -p 1080:1080 sunbk201/ua3f -f FFF
  ```

- Source Compilation:

  Build UA3F binary:

  ```sh
  git clone https://github.com/SunBK201/UA3F.git && cd UA3F/src
  go build -o ua3f main.go
  ```

  OpenWrt Compilation:

  ```sh
  git clone https://github.com/openwrt/openwrt.git && cd openwrt
  git checkout openwrt-24.10
  ./scripts/feeds update -a && ./scripts/feeds install -a
  git clone https://github.com/SunBK201/UA3F.git package/UA3F
  make menuconfig # Select Network->Web Servers/Proxies->ua3f
  make download -j$(nproc) V=s
  make -j$(nproc) || make -j1 || make -j1 V=sc
  # make package/UA3F/openwrt/compile -j1 V=sc # Compile single UA3F package
  ```

## Usage

### Command Line Usage

Start UA3F with default configuration:

```sh
ua3f
```

Start with a specified configuration file:

```sh
ua3f -c /path/to/config.yaml
```

Generate a template configuration file:

```sh
ua3f -g
```

For detailed CLI parameters, see [CLI.md](docs/cli.md). For a configuration file example, see [config.yaml](docs/config.yaml).

### OpenWrt LuCI Web Interface

UA3F supports OpenWrt LuCI Web interface. Navigate to Services -> UA3F for configuration.

For detailed tutorial, please visit: [UA3F User Guide](https://sunbk201public.notion.site/UA3F-2a21f32cbb4b80669e04ec1f053d0333)

<details>
<summary>Manual Command Line Launch</summary>

```sh
opkg install sudo
sudo -u nobody /usr/bin/ua3f
```

For shellclash/shellcrash users, use the following command:

```sh
sudo -u shellclash /usr/bin/ua3f
# If the above command fails, use this one
sudo -u shellcrash /usr/bin/ua3f
```

Command line parameters:

- `-c <config path>`: Custom configuration file path
- `-g`: Generate a template configuration file config.yaml in the current directory
- `-m <mode>`: Server mode. Supports HTTP, SOCKS5, TPROXY, REDIRECT. Default: SOCKS5
- `-b <bind addr>`: Custom bind address. Default: 127.0.0.1
- `-p <port>`: Port number. Default: 1080
- `-l <log level>`: Log level. Default: info. Options: debug. Default log location: `/var/log/ua3f.log`
- `-x`: Rewrite mode. Supports GLOBAL, DIRECT, RULE. Default: GLOBAL
- `-f <UA>`: Custom User-Agent. Default: FFF
- `-r <regex>`: Custom regex to match User-Agent. Default: empty (all User-Agents will be rewritten)
- `-s`: Partial replacement, only replace the regex matched portion
- `-z`: Rewrite rules in JSON string format. Only effective in RULE rewrite mode

</details>

### API Server

UA3F includes a built-in API Server controller that provides query and control interfaces for UA3F runtime status, configuration rules, and more. Enable it with the `--api-server` parameter:

```sh
ua3f --api-server <addr:port>
```

API documentation: [API.md](docs/api.md)

### Server Mode Description

UA3F supports 5 different server modes, each with unique characteristics:

| Server Mode  | Working Principle  | Clash Dependency | Compatibility | Coexist with Clash |
| ------------ | ------------------ | ---------------- | ------------- | ------------------ |
| **HTTP**     | HTTP Proxy         | Yes              | High          | Yes                |
| **SOCKS5**   | SOCKS5 Proxy       | Yes              | High          | Yes                |
| **TPROXY**   | netfilter TPROXY   | No               | Medium        | Yes                |
| **REDIRECT** | netfilter REDIRECT | No               | Medium        | Yes                |
| **NFQUEUE**  | netfilter NFQUEUE  | No               | Low           | Yes                |

### Rewrite Strategy Description

UA3F supports 3 different rewrite strategies:

| Rewrite Strategy | Rewrite Behavior                 | Rewrite Headers | Applicable Modes            |
| ---------------- | -------------------------------- | --------------- | --------------------------- |
| **GLOBAL**       | Rewrite all requests             | User-Agent      | All server modes            |
| **DIRECT**       | No rewriting, pure forwarding    | None            | All server modes            |
| **RULE**         | Rewrite based on rewriting rules | Customizable    | HTTP/SOCKS5/TPROXY/REDIRECT |

Rule Types:

| Rule Type      | Description                                       |
| -------------- | ------------------------------------------------- |
| DOMAIN         | Match based on domain name                        |
| DOMAIN-SUFFIX  | Match based on domain suffix                      |
| DOMAIN-KEYWORD | Match based on domain keyword                     |
| DOMAIN-SET     | Match based on a set of domain names              |
| IP-CIDR        | Match based on IP address range                   |
| SRC-IP         | Match based on source IP address                  |
| DST-PORT       | Match based on destination port                   |
| HEADER-KEYWORD | Match based on request header keyword             |
| HEADER-REGEX   | Match using regular expression on request headers |
| URL-REGEX      | Match using regular expression on request URL     |

Rewrite Actions:

| Action Type   | Description                                                   |
| ------------- | ------------------------------------------------------------- |
| DIRECT        | Allow directly without rewriting                              |
| DELETE        | Delete the specified header                                   |
| ADD           | Add the specified header with the given content               |
| REPLACE       | Replace the specified header with the given content           |
| REPLACE-REGEX | Replace the part of the specified header that matches a regex |
| REJECT        | Reject the request                                            |
| DROP          | Drop the request                                              |

URL Redirection Actions:
| Action Type     | Description                                                  |
| --------------- | ------------------------------------------------------------ |
| REDIRECT-302    | Return a 302 redirect response                               |
| REDIRECT-307    | Return a 307 redirect response                               |
| REDIRECT-HEADER | Modify request Header for redirection, transparent to client |

## Desync

See [UA3F Desync](docs/desync.md)

## Clash Configuration

See [Clash Configuration](docs/clash/Clash.md)

## Credits

- [Zxilly/UA2F](https://github.com/Zxilly/UA2F)
- [huhu415/uaProxy](https://github.com/huhu415/uaProxy)
- [CHN-beta/xmurp-ua](https://github.com/CHN-beta/xmurp-ua)
- [Dreamacro/clash](https://github.com/Dreamacro/clash)
- [MetaCubeX/mihomo](https://github.com/MetaCubeX/mihomo)
