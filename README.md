# UA3F

![Release](https://img.shields.io/github/v/release/SunBK201/UA3F?display_name=tag&label=UA3F&link=https%3A%2F%2Fgithub.com%2FSunBK201%2FUA3F%2Freleases%2Flatest)
[![CodeQL](https://github.com/SunBK201/UA3F/actions/workflows/github-code-scanning/codeql/badge.svg)](https://github.com/SunBK201/UA3F/actions/workflows/github-code-scanning/codeql)
[![License](https://img.shields.io/github/license/SunBK201/UA3F)](https://github.com/SunBK201/UA3F/blob/master/LICENSE)
![GitHub Downloads (all assets, all releases)](https://img.shields.io/github/downloads/SunBK201/UA3F/total?label=GitHub%20Downloads&link=https%3A%2F%2Fgithub.com%2FSunBK201%2FUA3F%2Freleases)
[![Telegram group](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fapi.swo.moe%2Fstats%2Ftelegram%2Fcrack_campus_network&query=count&color=2CA5E0&label=Telegram%20Group&logo=telegram&cacheSeconds=3600)](https://t.me/crack_campus_network)

[English](README_EN.md) | 简体中文

UA3F 是一个 HTTP Header 重写工具，作为一个 HTTP、SOCKS5、TPROXY、REDIRECT、NFQUEUE 服务对 HTTP 请求 Header (例如 User-Agent) 进行透明重写。

<table>
  <tr>
    <td>
      <ul>
        <li>应用层服务模式：HTTP、SOCKS5</li>
        <li>传输层服务模式：TPROXY、REDIRECT</li>
        <li>网络层服务模式：NFQUEUE(<a href="https://github.com/Zxilly/UA2F">UA2F</a>)</li>
        <li>高度灵活的重写规则系统，支持多种规则类型与重写策略</li>
        <li>实时统计面板，支持流量修改监控与分析</li>
        <li>支持 opkg 安装、编译安装、Docker 部署多种方式</li>
        <li>兼容 Clash Fake-IP & Redir-Host 多种模式伴生运行</li>
        <li>支持 TTL，TCP Timestamp，TCP Window，IPID 伪装</li>
        <li>支持 TCP Desync 分片乱序发射，用于对抗深度包检测（DPI）</li>
      </ul>
    </td>
    <td>
      <img src="https://sunbk201.oss-cn-beijing.aliyuncs.com/img/ua3f-210.png" alt="UA3F" width="300">
    </td>
  </tr>
</table>

<table>
  <tr>
    <td><img src="https://sunbk201.oss-cn-beijing.aliyuncs.com/img/ua3f-luci160.png" alt="UA3F-LuCI"></td>
    <td><img src="https://sunbk201.oss-cn-beijing.aliyuncs.com/img/ua3f-rule160.png" alt="UA3F-Rules"></td>
  </tr>
</table>

## 部署

提供 3 种部署方式：

- 使用 ipk 安装包进行部署：

  [Release](https://github.com/SunBK201/UA3F/releases) 页面已经提供常见架构的编译版本，可以根据自己设备的架构下载到 OpenWrt 上使用 `opkg install` 进行安装。

- OpenWrt 编译安装：

  ```sh
  git clone https://github.com/openwrt/openwrt.git && cd openwrt
  git checkout openwrt-24.10
  ./scripts/feeds update -a && ./scripts/feeds install -a
  git clone https://github.com/SunBK201/UA3F.git package/UA3F
  make menuconfig # 勾选 Network->Web Servers/Proxies->ua3f
  make download -j$(nproc) V=s
  make -j$(nproc) || make -j1 || make -j1 V=sc # make package/UA3F/openwrt/compile -j1 V=sc # 编译单个包
  ```

- Docker 部署：

  ```sh
  docker run -p 1080:1080 sunbk201/ua3f -f FFF
  ```

## 使用

UA3F 支持 LuCI Web 页面，可以打开 Services -> UA3F 进行相关配置。

快速使用教程详见：[猴子也能看懂的 UA3F 使用教程](https://sunbk201public.notion.site/UA3F-2a21f32cbb4b80669e04ec1f053d0333)

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

UA3F 支持以下规则类型：

- DOMAIN: 根据域名进行匹配
- DOMAIN-SUFFIX: 根据域名后缀进行匹配
- DOMAIN-KEYWORD: 根据域名关键字进行匹配
- IP-CIDR: 根据 IP 地址段进行匹配
- SRC-IP: 根据源 IP 地址进行匹配
- DST-PORT: 根据目标端口进行匹配
- HEADER-KEYWORD: 根据请求 Header 关键字进行匹配
- HEADER-REGEX: 根据请求 Header 进行正则匹配

UA3F 支持以下重写动作：

- DIRECT: 直接放行，不进行重写
- DELETE: 删除指定 Header
- REPLACE: 替换指定 Header 为指定内容
- REPLACE-REGEX: 替换指定 Header 中匹配正则的部分为指定内容
- DROP: 丢弃该请求

## Clash 配置建议

见 [Clash 配置](docs/Clash.md)

## References & Thanks

- [UA2F](https://github.com/Zxilly/UA2F)
- [uaProxy](https://github.com/huhu415/uaProxy)
- [xmurp-ua](https://github.com/CHN-beta/xmurp-ua)
- [Clash](https://github.com/Dreamacro/clash)
- [mihomo](https://github.com/MetaCubeX/mihomo)
