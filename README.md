# UA3F

UA3F 是一个 HTTP Header 重写工具，作为一个 HTTP、SOCKS5、TPROXY、REDIRECT、NFQUEUE 服务对 HTTP 请求 Header (例如 User-Agent) 进行透明重写。

## 特性

- 多种服务模式：HTTP、SOCKS5、TPROXY、REDIRECT、NFQUEUE([UA2F](https://github.com/Zxilly/UA2F))
- 高度灵活的重写规则系统，支持多种规则类型与重写策略
- 实时统计面板，支持流量修改监控与分析
- 支持 opkg 安装、编译安装、Docker 部署多种方式
- 支持 OpenWrt 17.01 及以上版本
- 兼容 Clash Fake-IP & Redir-Host 多种模式伴生运行

<table>
  <tr>
    <td><img src="https://sunbk201.oss-cn-beijing.aliyuncs.com/img/ua3f-luci160.png" alt="UA3F-LuCI"></td>
    <td><img src="https://sunbk201.oss-cn-beijing.aliyuncs.com/img/ua3f-rule160.png" alt="UA3F-Rules"></td>
  </tr>
</table>

![UA3F](https://sunbk201.oss-cn-beijing.aliyuncs.com/img/ua3f-160.png)

## 部署

提供 3 种部署方式：

- 使用 ipk 安装包进行部署：

  [Release](https://github.com/SunBK201/UA3F/releases) 页面已经提供常见架构的编译版本，可以根据自己设备的架构下载到 OpenWrt 上使用 `opkg install` 进行安装。

- OpenWrt 编译安装：

  ```sh
  git clone https://github.com/openwrt/openwrt.git && cd openwrt
  git checkout openwrt-22.03
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

具体使用教程详见：[猴子也能看懂的 UA3F 使用教程](https://sunbk201public.notion.site/UA3F-2a21f32cbb4b80669e04ec1f053d0333)

> [!NOTE]
> 设置说明：
>
> - Server Mode (服务模式): 支持 `HTTP`、`SOCKS5`、`TPROXY`、`REDIRECT`、`NFQUEUE`，默认 `SOCKS5`
> - Port (监听端口): 默认 `1080`
> - Bind Address (绑定地址): 默认 `0.0.0.0`
> - Log Level (日志等级): 默认 `error`, 如果需要调试排查错误可以设置为 `debug`
> - Rewrite Mode (重写策略): 默认 `GLOBAL`, 支持 `GLOBAL`、`DIRECT`、`RULES`
> - User-Agent (自定义重写 User-Agent): 默认 `FFF`
> - User-Agent Regex (User-Agent 正则表达式): 只重写匹配成功的 User-Agent。如果为空，全部重写
> - Partial Replace (部分替换): 只替换正则表达式匹配的部分。该选项仅在 User-Agent 正则表达式非空时生效

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
- `-x`: 重写策略，支持 GLOBAL、DIRECT、RULES，默认 GLOBAL
- `-f <UA>`: 自定义 UA，默认 FFF
- `-r <regex>`: 自定义正则匹配 User-Agent, 默认为空, 表示所有 User-Agent 都会被重写
- `-s`: 部分替换，仅替换正则匹配到的部分
- `-z`: 重写规则，json string 格式，仅在 RULES 重写策略模式下生效
</details>

### 服务模式说明

UA3F 支持 5 种不同的服务模式，各模式的特点和使用场景如下：

| 服务模式     | 工作原理           | 是否依赖 Clash 等 | 兼容性 | 效能 | 能否与 Clash 等伴生运行 |
| ------------ | ------------------ | ----------------- | ------ | ---- | ----------------------- |
| **HTTP**     | HTTP 代理          | 是                | 高     | 低   | 能                      |
| **SOCKS5**   | SOCKS5 代理        | 是                | 高     | 低   | 能                      |
| **TPROXY**   | netfilter TPROXY   | 否                | 中     | 中   | 能                      |
| **REDIRECT** | netfilter REDIRECT | 否                | 中     | 中   | 能                      |
| **NFQUEUE**  | netfilter NFQUEUE  | 否                | 中     | 高   | 能                      |

### 重写策略说明

UA3F 支持 3 种不同的重写策略：

| 重写策略   | 重写行为             | 适用服务模式                       |
| ---------- | -------------------- | ---------------------------------- |
| **GLOBAL** | 所有请求均进行重写   | 适用于所有服务模式                 |
| **DIRECT** | 不进行重写，纯转发   | 适用于所有服务模式                 |
| **RULES**  | 根据重写规则进行重写 | 适用于 HTTP/SOCKS5/TPROXY/REDIRECT |

## Clash 配置

Clash 与 UA3F 的配置部署教程详见：[UA3F 与 Clash 从零开始的部署教程](https://sunbk201public.notion.site/UA3F-Clash-16d60a7b5f0e457a9ee97a3be7cbf557?pvs=4)

```yaml
proxies:
  - name: "ua3f"
    type: socks5
    server: 127.0.0.1
    port: 1080
    url: http://connectivitycheck.platform.hicloud.com/generate_204
    udp: false

rules:
  - NETWORK,udp,DIRECT
  - MATCH,ua3f
```

> [!IMPORTANT]
> HTTP/SOCKS5 模式下，如果 Clash 使用 Fake-IP 模式，确保 OpenClash 本地 DNS 劫持选择「使用防火墙转发」，不要使用「Dnsmasq 转发」。

### Clash 参考配置

<table>
  <tr>
    <th>版本</th>
    <th>配置文件</th>
    <th>UA3F 运行模式</th>
    <th>说明</th>
  </tr>
  <tr>
    <td>国内版</td>
    <td><a href="https://cdn.jsdelivr.net/gh/SunBK201/UA3F@master/clash/ua3f-socks5-cn.yaml">ua3f-socks5-cn.yaml</a></td>
    <td>SOCKS5</td>
    <td>无需进行任何修改，可直接使用</td>
  </tr>
  <tr>
    <td>代理支持</td>
    <td><a href="https://cdn.jsdelivr.net/gh/SunBK201/UA3F@master/clash/ua3f-socks5-global.yaml">ua3f-socks5-global.yaml</a></td>
    <td>SOCKS5</td>
    <td>注意需要在 proxy-providers > Global-ISP > url 中（第 23 行）加入你的代理订阅链接</td>
  </tr>
  <tr>
    <td>抗 DPI + 代理支持</td>
    <td><a href="https://cdn.jsdelivr.net/gh/SunBK201/UA3F@master/clash/ua3f-socks5-global-dpi.yaml">ua3f-socks5-global-dpi.yaml</a></td>
    <td>SOCKS5</td>
    <td>注意需要在 proxy-providers > Global-ISP > url 中（第 23 行）加入你的代理订阅链接</td>
  </tr>
  <tr>
    <td>代理支持</td>
    <td><a href="https://cdn.jsdelivr.net/gh/SunBK201/UA3F@master/clash/ua3f-tproxy-cn-dpi.yaml">ua3f-tproxy-cn-dpi.yaml</a></td>
    <td>TPROXY/REDIRECT/NFQUEUE</td>
    <td>注意需要在 proxy-providers > Global-ISP > url 中（第 13 行）加入你的代理订阅链接</td>
  </tr>
  <tr>
    <td>抗 DPI + 代理支持</td>
    <td><a href="https://cdn.jsdelivr.net/gh/SunBK201/UA3F@master/clash/ua3f-tproxy-global-dpi.yaml">ua3f-tproxy-global-dpi.yaml</a></td>
    <td>TPROXY/REDIRECT/NFQUEUE</td>
    <td>注意需要在 proxy-providers > Global-ISP > url 中（第 18 行）加入你的代理订阅链接</td>
  </tr>
</table>

## References & Thanks

- [UA2F](https://github.com/Zxilly/UA2F)
- [uaProxy](https://github.com/huhu415/uaProxy)
- [xmurp-ua](https://github.com/CHN-beta/xmurp-ua)
- [Clash](https://github.com/Dreamacro/clash)
- [mihomo](https://github.com/MetaCubeX/mihomo)
