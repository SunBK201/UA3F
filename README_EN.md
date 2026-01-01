# UA3F

![Release](https://img.shields.io/github/v/release/SunBK201/UA3F?display_name=tag&label=UA3F&link=https%3A%2F%2Fgithub.com%2FSunBK201%2FUA3F%2Freleases%2Flatest)
[![CodeQL](https://github.com/SunBK201/UA3F/actions/workflows/github-code-scanning/codeql/badge.svg)](https://github.com/SunBK201/UA3F/actions/workflows/github-code-scanning/codeql)
[![License](https://img.shields.io/github/license/SunBK201/UA3F)](https://github.com/SunBK201/UA3F/blob/master/LICENSE)
![GitHub Downloads (all assets, all releases)](https://img.shields.io/github/downloads/SunBK201/UA3F/total?label=GitHub%20Downloads&link=https%3A%2F%2Fgithub.com%2FSunBK201%2FUA3F%2Freleases)
[![Telegram group](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fapi.swo.moe%2Fstats%2Ftelegram%2Fcrack_campus_network&query=count&color=2CA5E0&label=Telegram%20Group&logo=telegram&cacheSeconds=3600)](https://t.me/crack_campus_network)

English | [简体中文](README.md)

UA3F is an HTTP Header rewriting tool that transparently rewrites HTTP request headers (such as User-Agent) as an HTTP, SOCKS5, TPROXY, REDIRECT, or NFQUEUE server.

## Features

- Multiple server modes: HTTP, SOCKS5, TPROXY, REDIRECT, NFQUEUE([UA2F](https://github.com/Zxilly/UA2F))
- Highly flexible rewriting rule system with multiple rule types and rewriting strategies
- Real-time statistics dashboard with traffic modification monitoring and analysis
- Multiple deployment options: opkg installation, compilation, and Docker deployment
- Compatible with Clash Fake-IP & Redir-Host modes for coexistence
- Supports TTL, TCP Timestamp, TCP Window and IP ID obfuscation
- Supports TCP Desync fragment reordering to evade Deep Packet Inspection (DPI)

<div align="center">
<img src="https://sunbk201.oss-cn-beijing.aliyuncs.com/img/ua3f-210.png" alt="UA3F" style="width:40%;">
</div>

<table>
  <tr>
    <td><img src="https://sunbk201.oss-cn-beijing.aliyuncs.com/img/ua3f-luci160.png" alt="UA3F-LuCI"></td>
    <td><img src="https://sunbk201.oss-cn-beijing.aliyuncs.com/img/ua3f-rule160.png" alt="UA3F-Rules"></td>
  </tr>
</table>

## Deployment

Three deployment methods are available:

- **IPK Package Installation:**

  Pre-compiled versions for common architectures are available on the [Release](https://github.com/SunBK201/UA3F/releases) page. Download the appropriate version for your device architecture and install it on OpenWrt using `opkg install`.

- **OpenWrt Compilation:**

  ```sh
  git clone https://github.com/openwrt/openwrt.git && cd openwrt
  git checkout openwrt-24.10
  ./scripts/feeds update -a && ./scripts/feeds install -a
  git clone https://github.com/SunBK201/UA3F.git package/UA3F
  make menuconfig # Select Network->Web Servers/Proxies->ua3f
  make download -j$(nproc) V=s
  make -j$(nproc) || make -j1 || make -j1 V=sc # make package/UA3F/openwrt/compile -j1 V=sc # Compile single package
  ```

- **Docker Deployment:**

  ```sh
  docker run -p 1080:1080 sunbk201/ua3f -f FFF
  ```

## Usage

UA3F supports OpenWrt LuCI Web interface. Navigate to Services -> UA3F for configuration.

For detailed tutorial, please visit: [UA3F User Guide](https://sunbk201public.notion.site/UA3F-2a21f32cbb4b80669e04ec1f053d0333)

UA3F supports configuration via a YAML file. You can specify the configuration file path using the `-c` option, and generate a template configuration file using the `-g` option. An example configuration file can be found at [docs/config.yaml](docs/config.yaml).

Device and system information regex reference:

```regex
(Apple|iPhone|iPad|Macintosh|Mac OS X|Mac|Darwin|Microsoft|Windows|Linux|Android|OpenHarmony|HUAWEI|OPPO|Vivo|XiaoMi|Mobile|Dalvik)
```

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

### Server Mode Description

UA3F supports 5 different server modes, each with unique characteristics:

| Server Mode  | Working Principle  | Clash Dependency | Compatibility | Performance | Coexist with Clash |
| ------------ | ------------------ | ---------------- | ------------- | ----------- | ------------------ |
| **HTTP**     | HTTP Proxy         | Yes              | High          | Low         | Yes                |
| **SOCKS5**   | SOCKS5 Proxy       | Yes              | High          | Low         | Yes                |
| **TPROXY**   | netfilter TPROXY   | No               | Medium        | Medium      | Yes                |
| **REDIRECT** | netfilter REDIRECT | No               | Medium        | Medium      | Yes                |
| **NFQUEUE**  | netfilter NFQUEUE  | No               | Low           | High        | Yes                |

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
| DROP          | Drop the request                                              |

## Clash Configuration

See [Clash Configuration](docs/Clash.md)

## References & Thanks

- [UA2F](https://github.com/Zxilly/UA2F)
- [uaProxy](https://github.com/huhu415/uaProxy)
- [xmurp-ua](https://github.com/CHN-beta/xmurp-ua)
- [Clash](https://github.com/Dreamacro/clash)
- [mihomo](https://github.com/MetaCubeX/mihomo)
