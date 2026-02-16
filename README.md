# UA3F

![Release](https://img.shields.io/github/v/release/SunBK201/UA3F?display_name=tag&label=UA3F&link=https%3A%2F%2Fgithub.com%2FSunBK201%2FUA3F%2Freleases%2Flatest)
[![CodeQL](https://github.com/SunBK201/UA3F/actions/workflows/github-code-scanning/codeql/badge.svg)](https://github.com/SunBK201/UA3F/actions/workflows/github-code-scanning/codeql)
[![License](https://img.shields.io/github/license/SunBK201/UA3F)](https://github.com/SunBK201/UA3F/blob/master/LICENSE)
![GitHub Downloads (all assets, all releases)](https://img.shields.io/github/downloads/SunBK201/UA3F/total?label=GitHub%20Downloads&link=https%3A%2F%2Fgithub.com%2FSunBK201%2FUA3F%2Freleases)
[![Telegram group](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fapi.swo.moe%2Fstats%2Ftelegram%2Fcrack_campus_network&query=count&color=2CA5E0&label=Telegram%20Group&logo=telegram&cacheSeconds=3600)](https://t.me/crack_campus_network)

<img align="right" src="https://sunbk201.oss-cn-beijing.aliyuncs.com/img/ua3f-210.png" alt="UA3F" width="300">

[English](README_EN.md) | 简体中文

UA3F 是一个 HTTP(S) 重写工具，作为一个 HTTP、SOCKS5、TPROXY、REDIRECT、NFQUEUE 服务对 HTTP(S) 流量 (例如 User-Agent) 进行高效透明重写。

- 支持 HTTP(S) 请求与响应的 Header、Body 双向重写
- 支持 HTTP(S) URL 重定向：302、307、Header
- 支持 HTTPS MitM 流量解密重写
- 应用层服务模式：HTTP、SOCKS5
- 传输层服务模式：TPROXY、REDIRECT
- 网络层服务模式：NFQUEUE(<a href="https://github.com/Zxilly/UA2F">UA2F</a>)
- 高度灵活的重写规则系统，支持多种规则类型与重写策略
- 实时统计面板，支持流量修改监控与分析
- 支持 opkg 安装、编译安装、Docker 部署多种方式
- 兼容 Clash Fake-IP & Redir-Host 多种模式伴生运行
- 支持 TTL，TCP Timestamp，TCP Window，IPID 伪装
- 支持 Desync 分片乱序发射与混淆，用于对抗深度包检测（DPI）

## 部署

提供 4 种部署方式：

- 使用 OpenWrt 安装包进行部署：

  [Release](https://github.com/SunBK201/UA3F/releases) 页面已经提供常见架构的打包版本，可以根据自己设备的架构下载到 OpenWrt 上进行安装。

- OpenWrt 编译安装：

  ```sh
  git clone https://github.com/openwrt/openwrt.git && cd openwrt
  git checkout openwrt-24.10
  ./scripts/feeds update -a && ./scripts/feeds install -a
  git clone https://github.com/SunBK201/UA3F.git package/UA3F
  make menuconfig # 勾选 Network->Web Servers/Proxies->ua3f
  make download -j$(nproc) V=s
  make -j$(nproc) || make -j1 || make -j1 V=sc # make package/UA3F/openwrt/compile -j1 V=sc # 单独编译 UA3F
  ```

- Docker 部署：

  ```sh
  docker run -p 1080:1080 sunbk201/ua3f -f FFF
  ```

- 二进制文件下载

  [Release](https://github.com/SunBK201/UA3F/releases) 页面已经提供常见架构的编译版本，可以根据自己设备的架构下载对应的二进制文件使用。

## 使用

UA3F 支持 OpenWrt LuCI Web 页面，可以打开 Services -> UA3F 进行相关配置。

快速使用教程详见：[猴子也能看懂的 UA3F 使用教程](https://sunbk201public.notion.site/UA3F-2a21f32cbb4b80669e04ec1f053d0333)

UA3F 支持 yaml 文件进行配置，通过 `-c` 参数指定配置文件路径， 通过 `-g` 参数生成模板配置文件，配置文件示例见 [config.yaml](docs/config.yaml)

详细命令行配置说明见 [CLI.md](docs/cli.md)

设备与系统信息正则表达式参考：

```regex
(Apple|iPhone|iPad|Macintosh|Mac OS X|Mac|Darwin|Microsoft|Windows|Linux|Android|OpenHarmony|HUAWEI|OPPO|Vivo|XiaoMi|Mobile|Dalvik)
```

<details>
<summary>手动命令行启动</summary>

```sh
opkg install sudo
sudo -u nobody /usr/bin/ua3f
```

shellclash/shellcrash 用户建议使用以下命令启动:

```sh
sudo -u shellclash /usr/bin/ua3f
# 如果上面命令报错执行下面该命令
sudo -u shellcrash /usr/bin/ua3f
```

相关命令行启动参数:

- `-c <config path>`: 自定义配置文件路径
- `-g`: 在当前目录生成模板配置文件 config.yaml
- `-m <mode>`: 服务模式，支持 HTTP、SOCKS5、TPROXY、REDIRECT，默认 SOCKS5
- `-b <bind addr>`: 自定义绑定监听地址，默认 127.0.0.1
- `-p <port>`: 端口号，默认 1080
- `-l <log level>`: 日志等级，默认 info，可选：debug，默认日志位置：`/var/log/ua3f.log`
- `-x`: 重写策略，支持 GLOBAL、DIRECT、RULE，默认 GLOBAL
- `-f <UA>`: 自定义 UA，默认 FFF
- `-r <regex>`: 自定义正则匹配 User-Agent, 默认为空, 表示所有 User-Agent 都会被重写
- `-s`: 部分替换，仅替换正则匹配到的部分
- `-z`: 重写规则，json string 格式，仅在 RULE 重写策略模式下生效

</details>

### API Server

UA3F 内置 API Server 控制器，提供 UA3F 运行状态、配置规则等信息查询与控制接口，可以通过 `--api-server <addr:port>` 参数启用。

API 文档见 [API.md](docs/api.md)

### 服务模式说明

UA3F 支持 5 种不同的服务模式，各模式的特点和使用场景如下：

| 服务模式     | 工作原理           | 是否依赖 Clash 等 | 兼容性 | 性能 | 能否与 Clash 等伴生运行 |
| ------------ | ------------------ | ----------------- | ------ | ---- | ----------------------- |
| **HTTP**     | HTTP 代理          | 是                | 高     | 低   | 能                      |
| **SOCKS5**   | SOCKS5 代理        | 是                | 高     | 低   | 能                      |
| **TPROXY**   | netfilter TPROXY   | 否                | 中     | 中   | 能                      |
| **REDIRECT** | netfilter REDIRECT | 否                | 中     | 中   | 能                      |
| **NFQUEUE**  | netfilter NFQUEUE  | 否                | 低     | 高   | 能                      |

### 重写策略说明

UA3F 支持 3 种不同的重写策略：

| 重写策略   | 重写行为             | 重写 Header | 适用服务模式                       |
| ---------- | -------------------- | ----------- | ---------------------------------- |
| **GLOBAL** | 所有请求均进行重写   | User-Agent  | 适用于所有服务模式                 |
| **DIRECT** | 不进行重写，纯转发   | 无          | 适用于所有服务模式                 |
| **RULE**   | 根据重写规则进行重写 | 自定义      | 适用于 HTTP/SOCKS5/TPROXY/REDIRECT |

规则类型：

| 规则类型       | 说明                           |
| -------------- | ------------------------------ |
| DOMAIN         | 根据域名进行匹配               |
| DOMAIN-SUFFIX  | 根据域名后缀进行匹配           |
| DOMAIN-KEYWORD | 根据域名关键字进行匹配         |
| DOMAIN-SET     | 根据域名集合进行匹配           |
| IP-CIDR        | 根据 IP 地址段进行匹配         |
| SRC-IP         | 根据源 IP 地址进行匹配         |
| DST-PORT       | 根据目标端口进行匹配           |
| HEADER-KEYWORD | 根据请求 Header 关键字进行匹配 |
| HEADER-REGEX   | 根据请求 Header 进行正则匹配   |
| URL-REGEX      | 根据请求 URL 进行正则匹配      |

重写动作：

| 动作类型      | 说明                           |
| ------------- | ------------------------------ |
| DIRECT        | 直接放行，不进行重写           |
| DELETE        | 删除指定 Header                |
| ADD           | 添加指定 Header 为指定内容     |
| REPLACE       | 替换指定 Header 为指定内容     |
| REPLACE-REGEX | 将匹配正则的部分替换为指定内容 |
| REJECT        | 拒绝该请求                     |
| DROP          | 丢弃该请求                     |

URL 重定向动作：
| 动作类型 | 说明 |
| -------- | ------------------------------ |
| REDIRECT-302 | 返回 302 重定向响应 |
| REDIRECT-307 | 返回 307 重定向响应 |
| REDIRECT-HEADER | 修改请求 Header 进行重定向，客户端无感知 |

## Desync 说明

详见 [UA3F Desync](docs/desync.md)

## Clash 配置建议

见 [Clash 配置](docs/clash/Clash.md)

## Credits

- [Zxilly/UA2F](https://github.com/Zxilly/UA2F)
- [huhu415/uaProxy](https://github.com/huhu415/uaProxy)
- [CHN-beta/xmurp-ua](https://github.com/CHN-beta/xmurp-ua)
- [Dreamacro/clash](https://github.com/Dreamacro/clash)
- [MetaCubeX/mihomo](https://github.com/MetaCubeX/mihomo)
